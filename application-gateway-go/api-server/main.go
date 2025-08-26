package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"

	"api-server/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID        = "ISMSP"
	cryptoPath   = "/home/salman/go/src/github.com/135518004/fabric-samples/network/organizations/peerOrganizations/is.example.com"
	certPath     = cryptoPath + "/users/Admin@is.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/Admin@is.example.com/msp/keystore"
	tlsCertPath  = "/home/salman/go/src/github.com/135518004/fabric-samples/network/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem"
	peerEndpoint = "localhost:7051"
	gatewayPeer  = "peer0.is.example.com"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

var fcmClient *messaging.Client

// corsMiddleware sets CORS headers for all requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[CORS] Request received: %s %s", r.Method, r.URL.Path)
		// Allow requests from your React frontend origin
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		// Allow specific methods
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		// Allow specific headers
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("[AUTH] AuthMiddleware called")
		// Skip authentication for login and register routes
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" || r.URL.Path == "/api/status" {
			next.ServeHTTP(w, r)
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			sendJSONError(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Expecting "Bearer TOKEN_STRING"
		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			sendJSONError(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString = parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				sendJSONError(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}
			sendJSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			sendJSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Set the username in the request context
		ctx := context.WithValue(r.Context(), "username", claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	// Initialize database
	database.InitDB()

	// --- NEW: Update coordinator status from JSON file ---
	coordinatorsJSONPath := "/home/salman/go/src/github.com/135518004/fabric-samples/senior/application-gateway-go/api-server/coordinators.json"
	if err := database.UpdateCoordinatorStatusFromJSON(coordinatorsJSONPath); err != nil {
		log.Fatalf("Failed to update coordinator status from JSON: %v", err)
	}
	// --- END NEW ---

	// Initialize Firebase Admin SDK
	credentialsPath := "firebase-adminsdk.json"
	log.Printf("Attempting to load Firebase credentials from: %s", credentialsPath)
	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app from credentials file %s: %v", credentialsPath, err)
	}

	fcmClient, err = app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firebase Messaging client: %v", err)
	}

	conn := newGrpcConnection()
	defer conn.Close()

	idObj := newIdentity()
	sign := newSign()
	gw, err := client.Connect(
		idObj,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gw.Close()

	network := gw.GetNetwork("mychannel")
	contract := network.GetContract("documentApproval")

	r := mux.NewRouter()
	r.HandleFunc("/api/documents", getAllDocumentsHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/received/latest", getReceivedDocumentsWithDelayHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/received", getReceivedDocumentsHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/uploaded", getUploadedDocumentsHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/uploaded/latest", getUploadedDocumentsWithDelayHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/search", searchDocumentHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/autocomplete/{query}", autocompleteHandler).Methods("GET")
	r.HandleFunc("/api/documents/uploadedBy/{username}", getDocumentsByUploaderHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/search/suggestions", searchSuggestionsHandler).Methods("GET")
	r.HandleFunc("/api/documents/hash/{hash}", getDocumentByHashHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/view/{documentId}", verifyDocumentHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/{id}", getDocumentHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents", submitDocumentHandler(contract)).Methods("POST")
	r.HandleFunc("/api/documents/{id}/approve", approveDocumentHandler(contract)).Methods("POST")
	r.HandleFunc("/api/documents/{id}/editors", addEditorHandler(contract)).Methods("POST")
	r.HandleFunc("/api/documents/{id}/approvers", updateApproversHandler(contract)).Methods("PUT")
	r.HandleFunc("/api/documents/{documentId}", deleteDocumentHandler).Methods("DELETE")
	r.HandleFunc("/api/documents/{docId}/history", getDocumentHistoryHandler(contract)).Methods("GET")
	r.HandleFunc("/api/documents/{documentId:.+}/status", getDocumentStatusHandler).Methods("GET")

	// Authentication routes
	r.HandleFunc("/api/auth/register", registerHandler).Methods("POST")
	r.HandleFunc("/api/auth/login", loginHandler).Methods("POST")

	// Route for training applications
	r.HandleFunc("/api/applications", CreateApplicationHandler).Methods("POST")
	r.HandleFunc("/api/applications", GetApplicationsHandler).Methods("GET")
		r.HandleFunc("/api/applications/student/{studentId}", GetApplicationByStudentIDHandler).Methods("GET")
	r.HandleFunc("/api/training-applications/notifications", GetTrainingApplicationNotificationsHandler).Methods("GET")
	r.HandleFunc("/api/applications/{id}", DeleteApplicationHandler).Methods("DELETE")
	r.HandleFunc("/api/applications/{id}/status", UpdateApplicationStatusHandler).Methods("PUT")

	r.HandleFunc("/api/status", statusHandler).Methods("GET")
	r.HandleFunc("/api/users", getUsersHandler).Methods("GET")

	r.HandleFunc("/api/notifications", getNotificationsHandler(contract)).Methods("GET")
	r.HandleFunc("/api/sender/notifications", getSenderNotificationsHandler(contract)).Methods("GET")
	r.HandleFunc("/api/notifications/unread/count", getUnreadNotificationCountHandler).Methods("GET")
	r.HandleFunc("/api/sender/notifications/unread/count", getUnreadSenderNotificationCountHandler).Methods("GET")
	r.HandleFunc("/api/training-applications/unread/count", GetUnreadTrainingApplicationCountHandler).Methods("GET")
	r.HandleFunc("/api/notifications/{id}/view", markNotificationAsReadHandler).Methods("POST")
	r.HandleFunc("/api/sender/notifications/{id}/view", markSenderNotificationAsReadHandler).Methods("POST")
	r.HandleFunc("/api/training-notifications/{id}/view", MarkTrainingNotificationAsReadHandler).Methods("POST")
	

	// Ensure uploads directory exists
	uploadsDir := "../../uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadsDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create uploads directory: %v", err)
		}
	}

	r.HandleFunc("/api/documents/upload", uploadDocumentHandler).Methods("POST")
	r.PathPrefix("/api/documents/content/").Handler(http.StripPrefix("/api/documents/content/", serveDocumentHandler(uploadsDir)))

	// Add /api/auth/me endpoint
	r.HandleFunc("/api/auth/me", authMeHandler).Methods("GET")

	// Catch-all handler for debugging
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[CATCH_ALL] Unmatched request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 Not Found: %s", r.URL.Path)
	})

	log.Println("Server starting on port 8080...")
	// Apply CORS and Auth middleware to the router
	log.Println("[MAIN] Applying CORS and Auth middleware...")
	if err := http.ListenAndServe(":8080", corsMiddleware(AuthMiddleware(r))); err != nil {
		log.Fatal(err)
	}
	
}

func UpdateApplicationStatusHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    appIDStr := vars["id"]
    appID, err := strconv.Atoi(appIDStr)
    if err != nil {
        sendJSONError(w, "Invalid application ID", http.StatusBadRequest)
        return
    }

    var req struct {
        Status string `json:"status"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendJSONError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    err = database.UpdateTrainingApplicationStatus(appID, req.Status)
    if err != nil {
        log.Printf("Error updating application status %d: %v", appID, err)
        sendJSONError(w, "Failed to update application status", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Application status updated successfully"})
}

func CreateApplicationHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the incoming JSON from the request body
	var app database.TrainingApplication
	err := json.NewDecoder(r.Body).Decode(&app)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Basic validation (can be expanded)
	if app.StudentName == "" || app.StudentID == "" || app.CourseCode == "" {
		http.Error(w, "Missing required application fields", http.StatusBadRequest)
		return
	}

	// 3. Call the database function to save the application
	err = database.CreateTrainingApplication(&app)
	if err != nil {
		// Log the detailed error on the server
		log.Printf("Error creating application in DB: %v", err)
		// Send a generic error to the client
		http.Error(w, "Failed to save application", http.StatusInternalServerError)
		return
	}

	// 4. (Optional) Trigger Firebase notification to the coordinator here
	notificationTitle := "New Training Application"
	notificationBody := fmt.Sprintf("You have a new application from %s for course %s.", app.StudentName, app.CourseCode)
	sendFCMNotification(app.CoordinatorEmail, notificationTitle, notificationBody)


	// 5. Send a success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Application submitted successfully!"})
}

// GetApplicationsHandler handles fetching training applications for a coordinator
func GetApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	user, err := database.GetUserByUsername(username)
	if err != nil || user == nil {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}
	coordinatorEmail := user.Email // Get coordinator email from authenticated user's email

	if coordinatorEmail == "" {
		sendJSONError(w, "Coordinator email is required", http.StatusBadRequest)
		return
	}

	applications, err := database.GetTrainingApplicationsByCoordinatorEmail(coordinatorEmail)
	if err != nil {
		log.Printf("Error fetching applications for %s: %v", coordinatorEmail, err)
		sendJSONError(w, "Failed to fetch applications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(applications)
}

// GetApplicationByStudentEmailHandler handles fetching a training application by student email
func GetApplicationByStudentEmailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentEmail := vars["studentId"] // The frontend sends email in this parameter

	if studentEmail == "" {
		sendJSONError(w, "Student email is required", http.StatusBadRequest)
		return
	}

	app, err := database.GetTrainingApplicationByEmail(studentEmail)
	if err != nil {
		// This handles actual database errors, but not "not found".
		log.Printf("Error fetching application for student email %s: %v", studentEmail, err)
		sendJSONError(w, "Failed to fetch application", http.StatusInternalServerError)
		return
	}

	if app == nil {
		// This is the crucial check. If the app is not found, send a 404.
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Application not found for this student"})
		return
	}

	// If we found an application, return it
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

// GetApplicationByStudentIDHandler handles fetching a training application by student ID
func GetApplicationByStudentIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentID := vars["studentId"]

	if studentID == "" {
		sendJSONError(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	app, err := database.GetTrainingApplicationByStudentID(studentID)
	if err != nil {
		log.Printf("Error fetching application for student ID %s: %v", studentID, err)
		sendJSONError(w, "Failed to fetch application", http.StatusInternalServerError)
		return
	}

	if app == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Application not found for this student ID"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

// GetTrainingApplicationNotificationsHandler fetches all training applications for notification display
func GetTrainingApplicationNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	user, err := database.GetUserByUsername(username)
	if err != nil || user == nil {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}
	coordinatorEmail := user.Email // Get coordinator email from authenticated user's email
	if coordinatorEmail == "" {
		sendJSONError(w, "Coordinator email is required", http.StatusBadRequest)
		return
	}

	applications, err := database.GetTrainingApplicationsByCoordinatorEmail(coordinatorEmail)
	if err != nil {
		log.Printf("Error fetching training application notifications for %s: %v", coordinatorEmail, err)
		sendJSONError(w, "Failed to fetch training application notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(applications)
}

// DeleteApplicationHandler handles deleting a training application
func DeleteApplicationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appIDStr := vars["id"]
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		sendJSONError(w, "Invalid application ID", http.StatusBadRequest)
		return
	}

	err = database.DeleteTrainingApplication(appID)
	if err != nil {
		log.Printf("Error deleting application %d: %v", appID, err)
		sendJSONError(w, "Failed to delete application", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Application deleted successfully"})
}

func verifyDocumentHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		documentID := vars["documentId"]
		log.Printf("[VERIFY] Received request for documentId: %s", documentID)

		// 1. Get document hash and name from the database
		var dbHash, docName string
		err := database.DB.QueryRow("SELECT hash, doc_name FROM documents WHERE doc_id = $1", documentID).Scan(&dbHash, &docName)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("[VERIFY-FAIL] Document not found in database for doc_id: %s", documentID)
				sendJSONError(w, "Document not found in the database", http.StatusNotFound)
				return
			}
			log.Printf("[VERIFY-FAIL] Database error for doc_id: %s: %v", documentID, err)
			sendJSONError(w, "Failed to retrieve document from the database", http.StatusInternalServerError)
			return
		}
		log.Printf("[VERIFY] Database check OK for doc_id: %s. DB Hash: %s, Doc Name: %s", documentID, dbHash, docName)

		// 2. Check Fabric for document status
		networkResponse, err := contract.EvaluateTransaction("QueryDocumentStatus", documentID)
		if err != nil {
			log.Printf("[VERIFY-FAIL] Document not found in network for doc_id: %s: %v", documentID, err)
			sendJSONError(w, fmt.Sprintf("Document not found in the network: %v", err), http.StatusNotFound)
			return
		}

		log.Printf("[VERIFY] Raw network response for doc_id %s: %s", documentID, string(networkResponse))

		var networkDoc struct {
			Versions []struct {
				Hash string `json:"Hash"`
			} `json:"Versions"`
		}
		if err := json.Unmarshal(networkResponse, &networkDoc); err != nil {
			log.Printf("[VERIFY-FAIL] Failed to parse network response for doc_id: %s: %v", documentID, err)
			sendJSONError(w, "Failed to parse network response", http.StatusInternalServerError)
			return
		}

		var networkDocHash string
		if len(networkDoc.Versions) > 0 {
			networkDocHash = networkDoc.Versions[len(networkDoc.Versions)-1].Hash
		}

		log.Printf("[VERIFY] Network check OK for doc_id: %s. Latest Network Hash: %s", documentID, networkDocHash)

		// 3. Compare hashes
		if networkDocHash == "" {
			log.Printf("[VERIFY-FAIL] Network hash missing from chaincode response for doc_id: %s. Network response: %s", documentID, string(networkResponse))
			sendJSONError(w, "Network hash missing from chaincode response. Document integrity cannot be verified.", http.StatusConflict)
			return
		}

		if dbHash != networkDocHash {
			log.Printf("[VERIFY-FAIL] Hash mismatch for doc_id: %s. DB Hash: %s, Network Hash: %s. Network response: %s", documentID, dbHash, networkDocHash, string(networkResponse))
			sendJSONError(w, "Hash mismatch: Document integrity check failed", http.StatusConflict)
			return
		}
		log.Printf("[VERIFY-SUCCESS] Hashes match for doc_id: %s. Sending doc_name: %s. Full path: %s", documentID, docName, path.Join("../uploads", docName))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"doc_name": docName})
	}
}

func getUploadedDocumentsWithDelayHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Add a 2-second delay
		getUploadedDocumentsHandler(contract)(w, r)
	}
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Printf("[ERROR] Sending JSON error: %s (Status: %d)", message, statusCode)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func getAllDocumentsHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("[INFO] getAllDocumentsHandler called")

		// 1. Fetch all documents from the database
		rows, err := database.DB.Query(`
			SELECT id, doc_id, doc_name, upload_time, doc_path, hash, uploader_username, approved_count, rejected_count, pending_count
			FROM documents
		`)
		if err != nil {
			log.Printf("[ERROR] Failed to query documents from DB: %v", err)
			sendJSONError(w, "Failed to fetch documents from database", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 2. Process each document
		var documents []database.Document
		for rows.Next() {
			var doc database.Document
			if err := rows.Scan(
				&doc.ID,
				&doc.DocID,
				&doc.DocName,
				&doc.UploadTime,
				&doc.DocPath,
				&doc.Hash,
				&doc.SenderUsername,
				&doc.ApprovedCount,
				&doc.RejectedCount,
				&doc.PendingCount,
			); err != nil {
				log.Printf("[ERROR] Failed to scan document row: %v", err)
				continue // Skip to the next document
			}

			// Query blockchain for the latest status and approver map
			chaincodeResult, err := contract.EvaluateTransaction("QueryDocumentStatus", doc.DocID)
			if err != nil {
				log.Printf("[WARNING] Failed to query chaincode for doc %s: %v. Counts and map may be stale.", doc.DocID, err)
			} else {
				var statusResp struct {
					ApprovedCount int               `json:"ApprovedCount"`
					RejectedCount int               `json:"RejectedCount"`
					PendingCount  int               `json:"PendingCount"`
					ApprovalsMap  map[string]string `json:"ApprovalsMap"`
				}
				if err := json.Unmarshal(chaincodeResult, &statusResp); err != nil {
					log.Printf("[ERROR] Failed to unmarshal chaincode response for doc %s: %v", doc.DocID, err)
				} else {
					doc.ApprovedCount = statusResp.ApprovedCount
					doc.RejectedCount = statusResp.RejectedCount
					doc.PendingCount = statusResp.PendingCount
					doc.ApprovalsMap = statusResp.ApprovalsMap
				}
			}
			documents = append(documents, doc)
		}

		// 3. Return the list of documents
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(documents)
	}
}

func getDocumentHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// 1. Get full status from the blockchain
		chaincodeResult, err := contract.EvaluateTransaction("QueryDocumentStatus", id)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to evaluate transaction: %v", err), http.StatusInternalServerError)
			return
		}

		var chaincodeData map[string]interface{}
		if err := json.Unmarshal(chaincodeResult, &chaincodeData); err != nil {
			sendJSONError(w, "Failed to parse chaincode response", http.StatusInternalServerError)
			return
		}

		// 2. Get off-chain data from the database
		var docName, docPath string
		err = database.DB.QueryRow("SELECT doc_name, doc_path FROM documents WHERE doc_id = $1", id).Scan(&docName, &docPath)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Document %s not found in DB, returning chaincode data only", id)
				w.Header().Set("Content-Type", "application/json")
				w.Write(chaincodeResult)
				return
			}
			sendJSONError(w, "Failed to retrieve document from database", http.StatusInternalServerError)
			return
		}

		// 3. Combine and return
		combinedData := map[string]interface{}{
			"onChain":  chaincodeData,
			"offChain": map[string]string{
				"doc_name": docName,
				"doc_path": docPath,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(combinedData)
	}
}

func submitDocumentHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username").(string) // Get uploader/invoker from context
		var docReq struct {
			ID             string   `json:"id"`
			Name           string   `json:"name"`
			Hash           string   `json:"hash"`
			ApproverEmails []string `json:"approvers"`
			ValidDecisions []string `json:"validDecisions"`
			ResetApprovals bool     `json:"resetApprovals"`
		}

		if err := json.NewDecoder(r.Body).Decode(&docReq); err != nil {
			sendJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ensure ValidDecisions is not nil, which marshals to 'null'.
		// The chaincode schema expects an array '[]'.
		if docReq.ValidDecisions == nil {
			docReq.ValidDecisions = []string{}
		}

		// Check if document already exists in the database (for new submissions)
		var existingID string
		err := database.DB.QueryRow("SELECT doc_id FROM documents WHERE doc_id = $1", docReq.ID).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existing document: %v", err)
			sendJSONError(w, "Server error checking document", http.StatusInternalServerError)
			return
		}
		// If it exists, it's an update. If not, it's a new document.
		isUpdate := existingID != ""

		if !isUpdate {
			// Get uploader_id from database
			uploaderUser, err := database.GetUserByUsername(username)
			if err != nil || uploaderUser == nil {
				sendJSONError(w, "Uploader not found", http.StatusBadRequest)
				return
			}

			// Insert document into PostgreSQL for the first time
			var documentDBID int
			docPath := path.Join("../../uploads", docReq.Name)
			insertDocSQL := `INSERT INTO documents (doc_id, doc_name, doc_path, hash, uploader_username, approved_count, rejected_count, pending_count) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
			err = database.DB.QueryRow(insertDocSQL, docReq.ID, docReq.Name, docPath, docReq.Hash, username, 0, 0, len(docReq.ApproverEmails)).Scan(&documentDBID)
			if err != nil {
				log.Printf("Error inserting document into DB: %v", err)
				sendJSONError(w, "Server error creating document", http.StatusInternalServerError)
				return
			}
			log.Printf("[UPLOAD] Document saved to DB. doc_id: %s, doc_name: %s, hash: %s, docPath: %s", docReq.ID, docReq.Name, docReq.Hash, docPath)

			// Insert document shares for new documents
			for _, approverEmail := range docReq.ApproverEmails {
				approverUser, err := database.GetUserByEmail(approverEmail)
				if err != nil || approverUser == nil {
					sendJSONError(w, fmt.Sprintf("Approver with email %s not found", approverEmail), http.StatusBadRequest)
					return
				}
				insertShareSQL := `INSERT INTO document_shares (document_id, receiver_username, status) VALUES ($1, $2, $3)`
				_, err = database.DB.Exec(insertShareSQL, documentDBID, approverUser.Username, "PENDING")
				if err != nil {
					log.Printf("Error inserting document share into DB: %v", err)
					sendJSONError(w, "Server error submitting document shares", http.StatusInternalServerError)
					return
				}
				notificationTitle := "New Document for Approval"
				notificationBody := fmt.Sprintf("You have received a new document '%s' from %s for approval.", docReq.Name, username)
				sendFCMNotification(approverEmail, notificationTitle, notificationBody)
			}
		}

		// Prepare data for chaincode
		var approverUsernames []string
		for _, approverEmail := range docReq.ApproverEmails {
			approverUser, err := database.GetUserByEmail(approverEmail)
			if err != nil || approverUser == nil {
				sendJSONError(w, fmt.Sprintf("Approver with email %s not found", approverEmail), http.StatusBadRequest)
				return
			}
			approverUsernames = append(approverUsernames, approverUser.Username)
		}

		approversJSON, err := json.Marshal(approverUsernames)
		if err != nil {
			sendJSONError(w, "Failed to marshal approvers", http.StatusInternalServerError)
			return
		}

		validDecisionsJSON, err := json.Marshal(docReq.ValidDecisions)
		if err != nil {
			sendJSONError(w, "Failed to marshal valid decisions", http.StatusInternalServerError)
			return
		}

		resetApprovalsStr := strconv.FormatBool(docReq.ResetApprovals)

		// Submit to Fabric chaincode
		log.Printf("[SUBMIT_DOC] Submitting document to chaincode with data: ID: %s, Hash: %s, Uploader: %s, Approvers: %s, ValidDecisions: %s, ResetApprovals: %s", docReq.ID, docReq.Hash, username, string(approversJSON), string(validDecisionsJSON), resetApprovalsStr)
		_, err = contract.SubmitTransaction("SubmitDocument", docReq.ID, docReq.Hash, username, string(approversJSON), string(validDecisionsJSON), username, resetApprovalsStr)
		if err != nil {
			log.Printf("Error submitting transaction to Fabric: %v", err)
			sendJSONError(w, fmt.Sprintf("Failed to submit transaction to Fabric: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[UPLOAD] Document submitted to Fabric. doc_id: %s, hash: %s", docReq.ID, docReq.Hash)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Document submitted successfully", "doc_id": docReq.ID})
	}
}

		func approveDocumentHandler(contract *client.Contract) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        username := r.Context().Value("username").(string) // approver
        vars := mux.Vars(r)
        id := vars["id"] // doc_id

        var approval struct {
            Decision string `json:"decision"` // "APPROVED" or "REJECTED"
            Comment  string `json:"comment"`  // New field for comments
        }

        if err := json.NewDecoder(r.Body).Decode(&approval); err != nil {
            sendJSONError(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        // Submit to Fabric with the new comment field
        _, err := contract.SubmitTransaction("ApproveDocument", id, username, approval.Decision, approval.Comment)
        if err != nil {
            log.Printf("Failed to submit ApproveDocument transaction. docID: %s, user: %s, decision: %s. Error: %+v", id, username, approval.Decision, err)
            sendJSONError(w, fmt.Sprintf("Failed to submit transaction: %v", err), http.StatusInternalServerError)
            return
        }

        // After successful approval, query the chaincode for the latest status to sync DB
        time.Sleep(2 * time.Second)
        chaincodeResult, err := contract.EvaluateTransaction("QueryDocumentStatus", id)
        if err != nil {
            log.Printf("Error querying document status after approval for doc %s: %v", id, err)
            // If query fails, we can't sync, but the approval went through.
            // The old code continued, so we will too. The state will be synced by a later read.
        } else {
            var statusResp struct {
                ApprovedCount int               `json:"ApprovedCount"`
                RejectedCount int               `json:"RejectedCount"`
                PendingCount  int               `json:"PendingCount"`
                ApprovalsMap  map[string]string `json:"ApprovalsMap"` // Get the full map
            }
            if err := json.Unmarshal(chaincodeResult, &statusResp); err != nil {
                log.Printf("Error unmarshaling status after approval for doc %s: %v", id, err)
            } else {
                // Update the aggregate counts in the documents table
                updateCountsSQL := `
                    UPDATE documents
                    SET approved_count = $1, rejected_count = $2, pending_count = $3
                    WHERE doc_id = $4`
                _, err := database.DB.Exec(updateCountsSQL, statusResp.ApprovedCount, statusResp.RejectedCount, statusResp.PendingCount, id)
                if err != nil {
                    log.Printf("Error updating document counts in DB for doc %s: %v", id, err)
                } else {
                    log.Printf("Successfully updated document counts in DB for doc %s", id)
                }

                // Get document DB ID for updating shares
                var documentDBID int
                err = database.DB.QueryRow("SELECT id FROM documents WHERE doc_id = $1", id).Scan(&documentDBID)
                if err != nil {
                    log.Printf("Error finding document with doc_id %s to update shares: %v", id, err)
                } else {
                    // Sync all receiver statuses from the chaincode's response
                    for receiver, status := range statusResp.ApprovalsMap {
                        updateShareSQL := `UPDATE document_shares SET status = $1 WHERE document_id = $2 AND receiver_username = $3`
                        _, err := database.DB.Exec(updateShareSQL, status, documentDBID, receiver)
                        if err != nil {
                            // Log error but continue, to try and update as many as possible
                            log.Printf("Error updating document share for receiver %s on doc %s: %v", receiver, id, err)
                        }
                    }
                    log.Printf("Successfully synchronized document_shares for doc %s from ApprovalsMap", id)
                }
            }
        }

        // The section for updating document_shares is now handled by the sync logic above.
        // We still need to get info for the notification.
        var documentDBID int
        var uploaderUsername, docName string
        err = database.DB.QueryRow("SELECT id, uploader_username, doc_name FROM documents WHERE doc_id = $1", id).Scan(&documentDBID, &uploaderUsername, &docName)
        if err != nil {
            log.Printf("Error finding document with doc_id %s: %v", id, err)
            sendJSONError(w, "Failed to find document in database", http.StatusInternalServerError)
            return
        }

        notificationType := "document_approved"
        if approval.Decision == "REJECTED" {
            notificationType = "document_rejected"
        }

        insertNotificationSQL := `
            INSERT INTO sender_notifications (document_id, uploader_username, approver_username, status, doc_name, notification_type)
            VALUES ($1, $2, $3, $4, $5, $6)`
        _, err = database.DB.Exec(insertNotificationSQL, documentDBID, uploaderUsername, username, strings.ToUpper(approval.Decision), docName, notificationType)
        if err != nil {
            log.Printf("Error inserting sender notification: %v", err)
        }

        // Send notifications (simplified for clarity)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "message": fmt.Sprintf("Document %s %s by %s", id, approval.Decision, username),
        })
    }
}

func addEditorHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invokerID := r.Context().Value("username").(string)
		vars := mux.Vars(r)
		docID := vars["id"]

		var req struct {
			NewEditor string `json:"newEditor"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		_, err := contract.SubmitTransaction("AddEditor", docID, req.NewEditor, invokerID)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to add editor: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Editor added successfully"})
	}
}

func updateApproversHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invokerID := r.Context().Value("username").(string)
		vars := mux.Vars(r)
		docID := vars["id"]

		var req struct {
			NewApprovers []string `json:"newApprovers"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		newApproversJSON, err := json.Marshal(req.NewApprovers)
		if err != nil {
			sendJSONError(w, "Failed to marshal new approvers", http.StatusInternalServerError)
			return
		}

		_, err = contract.SubmitTransaction("UpdateDocumentApprovers", docID, string(newApproversJSON), invokerID)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to update approvers: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Approvers updated successfully"})
	}
}

func getDocumentByHashHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["hash"]

		result, err := contract.EvaluateTransaction("GetDocumentIdByHash", hash)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to get document by hash: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}


func deleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	// Delete from documents table
	_, err := database.DB.Exec("DELETE FROM documents WHERE doc_id = $1", documentID)
	if err != nil {
		log.Printf("Error deleting document: %v", err)
		sendJSONError(w, "Server error deleting document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Document deleted successfully"})
}

func getDocumentHistoryHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		docID := vars["docId"]

		// Query the chaincode for document history
		result, err := contract.EvaluateTransaction("GetHistory", docID)
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to get document history from Fabric: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}

// Claims defines the JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		FCMToken string `json:"fcm_token"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		sendJSONError(w, "Username, email, password, and role are required", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingUser, err := database.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("Error checking existing user: %v", err)
		sendJSONError(w, "Server error during registration", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		sendJSONError(w, "User with this username already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		sendJSONError(w, "Server error during registration", http.StatusInternalServerError)
		return
	}

	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		FCMToken: sql.NullString{String: req.FCMToken, Valid: req.FCMToken != ""},
		Role:     req.Role,
	}

	if err := database.CreateUser(user); err != nil {
		log.Printf("Error creating user in DB: %v", err)
		sendJSONError(w, "Server error during registration", http.StatusInternalServerError)
		return
	}

	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("Error signing token: %v", err)
		sendJSONError(w, "Server error during login", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "User registered successfully", "token": tokenString, "user": map[string]string{"id": req.Username, "email": req.Email, "role": req.Role}})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		FCMToken string `json:"fcm_token"`
	}

	// Read the body for logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		sendJSONError(w, "can't read body", http.StatusBadRequest)
		return
	}
	// Restore the body so it can be read again
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Printf("[LOGIN] Received raw request body: %s", string(bodyBytes))

	log.Println("[LOGIN] Attempting to decode JSON body...")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[LOGIN-ERROR] Failed to decode JSON: %v", err)
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Println("[LOGIN] JSON body decoded successfully.")

	if req.Username == "" || req.Password == "" {
		log.Println("[LOGIN-ERROR] Username or password missing.")
		sendJSONError(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	log.Println("[LOGIN] Username and password present.")

	log.Printf("[LOGIN] Attempting to get user by identifier: %s", req.Username)
    var user *database.User

    // Check if the login identifier is an email or a username
    if strings.Contains(req.Username, "@") {
        log.Printf("[LOGIN] Identifier contains '@', treating as email.")
        user, err = database.GetUserByEmail(req.Username)
    } else {
        log.Printf("[LOGIN] Identifier does not contain '@', treating as username.")
        user, err = database.GetUserByUsername(req.Username)
    }

	if err != nil {
		log.Printf("[LOGIN-ERROR] Error getting user from DB: %v", err)
		sendJSONError(w, "Server error during login", http.StatusInternalServerError)
		return
	}
	if user == nil {
		log.Printf("[LOGIN-ERROR] User with identifier '%s' not found.", req.Username)
		sendJSONError(w, "Invalid credentials", http.StatusBadRequest)
		return
	}
	log.Printf("[LOGIN] User %s found in database.", req.Username)

	log.Println("[LOGIN] Comparing hashed password...")
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("[LOGIN-ERROR] Password comparison failed for user %s: %v", req.Username, err)
		sendJSONError(w, "Invalid credentials", http.StatusBadRequest)
		return
	}
	log.Println("[LOGIN] Password comparison successful.")

	// Update FCM token if provided
	if req.FCMToken != "" {
		log.Printf("[LOGIN] FCM token provided. Attempting to update for user %s...", req.Username)
		nullFCMToken := sql.NullString{String: req.FCMToken, Valid: true}
		err = database.UpdateUserFCMToken(req.Username, nullFCMToken)
		if err != nil {
			log.Printf("[LOGIN-ERROR] Error updating FCM token for user %s: %v", req.Username, err)
			// Do not block login for FCM token update failure
		} else {
			log.Printf("[LOGIN] FCM token updated successfully for user %s.", req.Username)
		}
	} else {
		log.Println("[LOGIN] No FCM token provided.")
	}

	log.Println("[LOGIN] Generating JWT token...")
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("[LOGIN-ERROR] Error signing token: %v", err)
		sendJSONError(w, "Server error during login", http.StatusInternalServerError)
		return
	}
	log.Println("[LOGIN] JWT token generated successfully.")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Logged in successfully", "token": tokenString, "user": map[string]string{"id": user.Username, "email": user.Email, "role": user.Role}})
	log.Println("[LOGIN] Login successful. Response sent.")
}

func authMeHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value("username").(string)
	if !ok || username == "" {
		sendJSONError(w, "Unauthorized: No valid token provided", http.StatusUnauthorized)
		return
	}

	user, err := database.GetUserByUsername(username)
	if err != nil || user == nil {
		log.Printf("[AUTH_ME_ERROR] User %s not found in DB after token validation: %v", username, err)
		sendJSONError(w, "Authentication failed: User not found", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": user.Username, "email": user.Email, "role": user.Role})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Backend and database are running"})
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT id, username, email FROM users")
	if err != nil {
		log.Printf("Error querying users: %v", err)
		sendJSONError(w, "Server error fetching users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []database.User
	for rows.Next() {
		var user database.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
			log.Printf("Error scanning user row: %v", err)
			sendJSONError(w, "Server error fetching users", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}



func getReceivedDocumentsWithDelayHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Add a 2-second delay
		getReceivedDocumentsHandler(contract)(w, r)
	}
}

func getReceivedDocumentsHandler(contract *client.Contract) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Initialize tracing
        log.Println("[TRACE] Entering getReceivedDocumentsHandler")
        defer log.Println("[TRACE] Exiting getReceivedDocumentsHandler")

        username := r.Context().Value("username").(string) // Get username from context
        log.Printf("[DEBUG] Request for username: %q", username) // %q shows quotes to reveal hidden chars

        if username == "" {
            log.Println("[ERROR] Empty username received")
            sendJSONError(w, "Username is required", http.StatusBadRequest)
            return
        }

        // Verify database connection
        if err := database.DB.Ping(); err != nil {
            log.Printf("[DB ERROR] Connection failed: %v", err)
            sendJSONError(w, "Database unavailable", http.StatusServiceUnavailable)
            return
        }

        query := `
            SELECT d.id, d.doc_id, d.doc_name, ds.status, 
                   d.upload_time, d.doc_path, d.hash,
                   d.uploader_username as sender_username
            FROM documents d
            JOIN document_shares ds ON d.id = ds.document_id
            WHERE ds.receiver_username = $1
        `
        log.Printf(`"[SQL] Query: %s
[SQL] Param: %q"`, query, username)

        rows, err := database.DB.Query(query, username)
        if err != nil {
            log.Printf("[QUERY ERROR] %v", err)
            sendJSONError(w, "Database query failed", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        // Get column names
        columns, err := rows.Columns()
        if err != nil {
            log.Printf("[COLUMNS ERROR] %v", err)
            sendJSONError(w, "Database error", http.StatusInternalServerError)
            return
        }
        log.Printf("[DEBUG] Columns: %v", columns)

        var documents []map[string]interface{}
        count := 0

        for rows.Next() {
            count++
            // Dynamic scanning
            values := make([]interface{}, len(columns))
            valuePtrs := make([]interface{}, len(columns))
            for i := range columns {
                valuePtrs[i] = &values[i]
            }

            if err := rows.Scan(valuePtrs...); err != nil {
                log.Printf("[SCAN ERROR] Row %d: %v", count, err)
                continue
            }

            doc := make(map[string]interface{})
            for i, col := range columns {
                val := values[i]
                b, ok := val.([]byte)
                if ok {
                    doc[col] = string(b)
                } else {
                    doc[col] = val
                }
            }
            log.Printf("[DEBUG] Row %d: %+v", count, doc)
            documents = append(documents, doc)
        }

        log.Printf("[RESULT] Found %d documents for %q", len(documents), username)

        if len(documents) == 0 {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode([]map[string]interface{}{})
            return
        }

        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(documents); err != nil {
            log.Printf("[ENCODE ERROR] %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte(`"{"error":"response encoding failed"}"`))
        }
    }
}


func getUploadedDocumentsHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username").(string) // Get username from context
		log.Printf("[UPLOADED_DOCS] Received request for uploaded documents. Username: %s", username)

		if username == "" {
			sendJSONError(w, "Username is required", http.StatusBadRequest)
			log.Println("[UPLOADED_DOCS-FAIL] Username missing in request")
			return
		}

		// In your backend handler
		query := ` SELECT d.id, 
        d.doc_id, 
        d.doc_name, 
        d.upload_time,
        d.doc_path,
        d.hash,
        d.uploader_username,
		d.approved_count,
		d.rejected_count,
		d.pending_count
    FROM documents d
    WHERE d.uploader_username = $1
`

		log.Printf("[UPLOADED_DOCS] Executing DB query: %s with username: %s", query, username)
		rows, err := database.DB.Query(query, username)
		if err != nil {
			log.Printf("[UPLOADED_DOCS-FAIL] Error querying uploaded documents from DB: %v", err)
			sendJSONError(w, "Server error fetching uploaded documents", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type UploadedDoc struct {
			ID         int             `json:"id"`
			DocID      string          `json:"doc_id"`
			DocName    string          `json:"doc_name"`
			UploadTime time.Time       `json:"upload_time"`
			DocPath    string          `json:"doc_path"`
			Hash       string          `json:"hash"`
			Uploader   string          `json:"uploader_username"`
			ApprovedCount int            `json:"approved_count"`
			RejectedCount int            `json:"rejected_count"`
			PendingCount  int            `json:"pending_count"`
			ApprovalsMap  map[string]string `json:"approvals_map"` // Added for individual statuses
		}

		var documents []UploadedDoc
		for rows.Next() {
			var doc UploadedDoc
			if err := rows.Scan(
				&doc.ID,
				&doc.DocID,
				&doc.DocName,
				&doc.UploadTime,
				&doc.DocPath,
				&doc.Hash,
				&doc.Uploader,
				&doc.ApprovedCount,
				&doc.RejectedCount,
				&doc.PendingCount,
			); err != nil {
				log.Printf("[UPLOADED_DOCS-FAIL] Error scanning uploaded document row: %v", err)
				continue
			}

			// Query chaincode for the latest status and approver map
			chaincodeResult, err := contract.EvaluateTransaction("QueryDocumentStatus", doc.DocID)
			if err != nil {
                if strings.Contains(err.Error(), "not found") {
                    log.Printf("[UPLOADED_DOCS-SKIP] Document %s found in DB but not on chain. Skipping.", doc.DocID)
                    continue // Skip this inconsistent record
                }
				log.Printf("[UPLOADED_DOCS-WARN] Chaincode error for document %s: %v. Counts and map may be stale.", doc.DocID, err)
			} else {
				var statusResp struct {
					ApprovedCount int               `json:"ApprovedCount"`
					RejectedCount int               `json:"RejectedCount"`
					PendingCount  int               `json:"PendingCount"`
					ApprovalsMap  map[string]string `json:"ApprovalsMap"`
				}
				if err := json.Unmarshal(chaincodeResult, &statusResp); err != nil {
					log.Printf("[UPLOADED_DOCS-FAIL] Error unmarshaling status for doc %s: %v.", doc.DocID, err)
				} else {
					doc.ApprovedCount = statusResp.ApprovedCount
					doc.RejectedCount = statusResp.RejectedCount
					doc.PendingCount = statusResp.PendingCount
					doc.ApprovalsMap = statusResp.ApprovalsMap
				}
			}
			log.Printf("[UPLOADED_DOCS] Final counts and map for %s: Approved=%d, Rejected=%d, Pending=%d, ApprovalsMap=%+v", doc.DocID, doc.ApprovedCount, doc.RejectedCount, doc.PendingCount, doc.ApprovalsMap)

			documents = append(documents, doc)
		}

		// If no documents are found, explicitly return an empty JSON array
		if len(documents) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]UploadedDoc{})
			return
		}

		log.Printf("[UPLOADED_DOCS] Returning %d uploaded documents for user: %s. Documents: %+v (New Logic Applied)", len(documents), username, documents)
		// Log the JSON representation of documents for detailed inspection
		jsonOutput, err := json.MarshalIndent(documents, "", "  ")
		if err != nil {
			log.Printf("[UPLOADED_DOCS-FAIL] Error marshaling documents to JSON: %v", err)
		} else {
			log.Printf("[UPLOADED_DOCS] JSON output to frontend: %s", string(jsonOutput))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(documents)
	}
}

func searchDocumentHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		queryParam := vars["docName"] // Renamed to queryParam as it can be docName or uploader
		log.Printf("[SEARCH] Received search request for query: %s", queryParam)

		// Search in the database for documents matching doc_name or uploader_username
		// Using ILIKE for case-insensitive search and % for partial matches
		rows, err := database.DB.Query("SELECT doc_id, doc_name, uploader_username FROM documents WHERE doc_name ILIKE $1 OR uploader_username ILIKE $1", "%"+queryParam+"%")
		if err != nil {
			log.Printf("[SEARCH-FAIL] Database error for query: %s: %v", queryParam, err)
			sendJSONError(w, "Failed to search for documents in the database", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var documents []map[string]string
		for rows.Next() {
			var docID, docName, uploaderUsername string
			if err := rows.Scan(&docID, &docName, &uploaderUsername); err != nil {
				log.Printf("[SEARCH-FAIL] Error scanning document row: %v", err)
				continue
			}
			documents = append(documents, map[string]string{
				"doc_id": docID,
				"doc_name": docName,
				"uploader_username": uploaderUsername,
			})
		}

		if len(documents) == 0 {
			log.Printf("[SEARCH-FAIL] No documents found for query: %s", queryParam)
			sendJSONError(w, fmt.Sprintf("No documents found for %q", queryParam), http.StatusNotFound)
			return
		}

		log.Printf("[SEARCH-SUCCESS] Found %d documents for query: %s", len(documents), queryParam)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(documents)
	}
}

func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["query"]
	log.Printf("[AUTOCOMPLETE] Received autocomplete request for query: %s", query)

	rows, err := database.DB.Query("SELECT doc_name FROM documents WHERE doc_name ILIKE $1 LIMIT 10", query+"%")
	if err != nil {
		log.Printf("[AUTOCOMPLETE-FAIL] Database error for query: %s: %v", query, err)
		sendJSONError(w, "Failed to fetch autocomplete suggestions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var suggestion string
		if err := rows.Scan(&suggestion); err != nil {
			log.Printf("[AUTOCOMPLETE-FAIL] Error scanning suggestion row: %v", err)
			continue
		}
		suggestions = append(suggestions, suggestion)
	}

	log.Printf("[AUTOCOMPLETE-SUCCESS] Found %d suggestions for query: %s", len(suggestions), query)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestions)
}

func searchSuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	log.Printf("[SEARCH_SUGGESTIONS] Received request for query: %s", query)

	if query == "" {
		sendJSONError(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	suggestions := struct {
		DocumentNames []string `json:"documentNames"`
		UploaderNames []string `json:"uploaderNames"`
		ReceiverNames []string `json:"receiverNames"`
	}{
		DocumentNames: []string{},
		UploaderNames: []string{},
		ReceiverNames: []string{},
	}

	// Search for document names with uploader
	docRows, err := database.DB.Query("SELECT DISTINCT doc_name, uploader_username FROM documents WHERE doc_name ILIKE $1 LIMIT 5", "%"+query+"%")
	if err != nil {
		log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error querying document names: %v", err)
	} else {
		for docRows.Next() {
			var docName, uploaderName string
			if err := docRows.Scan(&docName, &uploaderName); err != nil {
				log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error scanning document name and uploader: %v", err)
				continue
			}
			// Format the string for display
			formattedSuggestion := fmt.Sprintf("Document: %s (Uploader: %s)", docName, uploaderName)
			suggestions.DocumentNames = append(suggestions.DocumentNames, formattedSuggestion)
		}
		docRows.Close()
	}

	// Search for uploader names
	uploaderRows, err := database.DB.Query("SELECT DISTINCT uploader_username FROM documents WHERE uploader_username ILIKE $1 LIMIT 5", "%"+query+"%")
	if err != nil {
		log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error querying uploader names: %v", err)
	} else {
		for uploaderRows.Next() {
			var uploaderName string
			if err := uploaderRows.Scan(&uploaderName); err != nil {
				log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error scanning uploader name: %v", err)
				continue
			}
			suggestions.UploaderNames = append(suggestions.UploaderNames, uploaderName)
		}
		uploaderRows.Close()
	}

	// Search for receiver names
	receiverRows, err := database.DB.Query("SELECT DISTINCT receiver_username FROM document_shares WHERE receiver_username ILIKE $1 LIMIT 5", "%"+query+"%")
	if err != nil {
		log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error querying receiver names: %v", err)
	} else {
		for receiverRows.Next() {
			var receiverName string
			if err := receiverRows.Scan(&receiverName); err != nil {
				log.Printf("[SEARCH_SUGGESTIONS-FAIL] Error scanning receiver name: %v", err)
				continue
			}
			suggestions.ReceiverNames = append(suggestions.ReceiverNames, receiverName)
		}
		receiverRows.Close()
	}

	log.Printf("[SEARCH_SUGGESTIONS-SUCCESS] Found suggestions: %+v", suggestions)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestions)
}

func getDocumentsByUploaderHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username").(string) // Get username from context
		log.Printf("[DOCS_BY_UPLOADER] Received request for documents uploaded by: %s", username)

		if username == "" {
			sendJSONError(w, "Username is required", http.StatusBadRequest)
			log.Println("[DOCS_BY_UPLOADER-FAIL] Username missing in request")
			return
		}

		query := `
			SELECT 
				d.id, 
				d.doc_id, 
				d.doc_name, 
				d.upload_time,
				d.doc_path,
				d.hash,
				d.uploader_username,
				d.approved_count,
				d.rejected_count,
				d.pending_count
			FROM documents d
			WHERE d.uploader_username = $1
		`

		log.Printf("[DOCS_BY_UPLOADER] Executing DB query: %s with username: %s", query, username)
		rows, err := database.DB.Query(query, username)
		if err != nil {
			log.Printf("[DOCS_BY_UPLOADER-FAIL] Error querying documents by uploader from DB: %v", err)
			sendJSONError(w, "Server error fetching documents by uploader", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type UploadedDoc struct {
			ID         int             `json:"id"`
			DocID      string          `json:"doc_id"`
			DocName    string          `json:"doc_name"`
			UploadTime time.Time       `json:"upload_time"`
			DocPath    string          `json:"doc_path"`
			Hash       string          `json:"hash"`
			Uploader   string          `json:"uploader_username"`
			ApprovedCount int            `json:"approved_count"`
			RejectedCount int            `json:"rejected_count"`
			PendingCount  int            `json:"pending_count"`
			ApprovalsMap  map[string]string `json:"approvals_map"` // Added for individual statuses
		}

		var documents []UploadedDoc
		for rows.Next() {
			var doc UploadedDoc
			if err := rows.Scan(
				&doc.ID,
				&doc.DocID,
				&doc.DocName,
				&doc.UploadTime,
				&doc.DocPath,
				&doc.Hash,
				&doc.Uploader,
				&doc.ApprovedCount,
				&doc.RejectedCount,
				&doc.PendingCount,
			); err != nil {
				log.Printf("[DOCS_BY_UPLOADER-FAIL] Error scanning document row: %v", err)
				continue
			}

			// Query chaincode for the latest status and approver map
			chaincodeResult, err := contract.EvaluateTransaction("QueryDocumentStatus", doc.DocID)
			if err != nil {
				log.Printf("[DOCS_BY_UPLOADER-FAIL] Chaincode error for document %s: %v. Counts and map may be stale.", doc.DocID, err)
			} else {
				var statusResp struct {
					ApprovedCount int               `json:"ApprovedCount"`
					RejectedCount int               `json:"RejectedCount"`
					PendingCount  int               `json:"PendingCount"`
					ApprovalsMap  map[string]string `json:"ApprovalsMap"`
				}
				if err := json.Unmarshal(chaincodeResult, &statusResp); err != nil {
					log.Printf("[DOCS_BY_UPLOADER-FAIL] Error unmarshaling status for doc %s: %v.", doc.DocID, err)
				} else {
					doc.ApprovedCount = statusResp.ApprovedCount
					doc.RejectedCount = statusResp.RejectedCount
					doc.PendingCount = statusResp.PendingCount
					doc.ApprovalsMap = statusResp.ApprovalsMap
				}
			}
			log.Printf("[DOCS_BY_UPLOADER] Final counts and map for %s: Approved=%d, Rejected=%d, Pending=%d, ApprovalsMap=%+v", doc.DocID, doc.ApprovedCount, doc.RejectedCount, doc.PendingCount, doc.ApprovalsMap)

			documents = append(documents, doc)
		}

		log.Printf("[DOCS_BY_UPLOADER] Returning %d documents for uploader: %s. Documents: %+v", len(documents), username, documents)
		jsonOutput, err := json.MarshalIndent(documents, "", "  ")
		if err != nil {
			log.Printf("[DOCS_BY_UPLOADER-FAIL] Error marshaling documents to JSON: %v", err)
		} else {
			log.Printf("[DOCS_BY_UPLOADER] JSON output to frontend: %s", string(jsonOutput))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(documents)
	}
}

func newGrpcConnection() *grpc.ClientConn {


	certPEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		panic(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	creds := credentials.NewClientTLSFromCert(pool, gatewayPeer)

	conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return conn
}

func newIdentity() *identity.X509Identity {
	pem, err := os.ReadFile(firstFile(certPath))
	if err != nil {
		panic(err)
	}
	cert, err := identity.CertificateFromPEM(pem)
	if err != nil {
		panic(err)
	}
	id, err := identity.NewX509Identity(mspID, cert)
	if err != nil {
		panic(err)
	}
	return id
}

func newSign() identity.Sign {
	pem, err := os.ReadFile(firstFile(keyPath))
	if err != nil {
		panic(err)
	}
	pk, err := identity.PrivateKeyFromPEM(pem)
	if err != nil {
		panic(err)
	}
	sign, err := identity.NewPrivateKeySign(pk)
	if err != nil {
		panic(err)
	}
	return sign
}

func firstFile(dir string) string {
	f, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	names, err := f.Readdirnames(1)
	if err != nil {
		panic(err)
	}
	return path.Join(dir, names[0])
}

func uploadDocumentHandler(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error retrieving the file"})
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadsDir := "../../uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadsDir, 0755)
		if err != nil {
			log.Printf("Failed to create uploads directory: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Server error creating upload directory"})
			return
		}
	}

	// Create a new file in the uploads directory
	dst, err := os.Create(fmt.Sprintf("%s/%s", uploadsDir, handler.Filename))
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server error creating file"})
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Failed to copy file: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server error saving file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}

func getNotificationsHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username").(string) // Get username from context
		if username == "" {
			sendJSONError(w, "Username is required", http.StatusBadRequest)
			return
		}

		query := `
            SELECT
                ds.id,
                d.doc_id,
                d.doc_name,
                d.uploader_username AS sender_username,
                ds.receiver_username AS approver_username, -- Add this line
                ds.status,
                ds.created_at,
                ds.viewed,
                CASE
                    WHEN ds.status = 'PENDING' THEN 'new_document'
                    ELSE 'status_change'
                END AS type
            FROM document_shares ds
            JOIN documents d ON ds.document_id = d.id
            WHERE ds.receiver_username = $1
            ORDER BY ds.created_at DESC
        `

		rows, err := database.DB.Query(query, username)
		if err != nil {
			log.Printf("[NOTIFICATIONS-FAIL] Error querying notifications from DB: %v", err)
			sendJSONError(w, "Server error fetching notifications", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Notification struct {
			ID             int       `json:"id"`
			DocID          string    `json:"doc_id"`
			DocName        string    `json:"doc_name"`
			SenderUsername string    `json:"sender_username"`
			ApproverUsername sql.NullString `json:"approver_username"`
			Status         string    `json:"status"`
			CreatedAt      time.Time `json:"created_at"`
			Type           string    `json:"type"`
			Viewed         bool      `json:"viewed"`
		}

		var notifications []Notification
		for rows.Next() {
			var n Notification
			if err := rows.Scan(&n.ID, &n.DocID, &n.DocName, &n.SenderUsername, &n.ApproverUsername, &n.Status, &n.CreatedAt, &n.Viewed, &n.Type); err != nil {
				log.Printf("[NOTIFICATIONS-FAIL] Error scanning notification row: %v", err)
				continue
			}
			notifications = append(notifications, n)
		}

		if len(notifications) == 0 {
			notifications = make([]Notification, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}

func getSenderNotificationsHandler(contract *client.Contract) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username").(string) // Get username from context
		if username == "" {
			sendJSONError(w, "Username is required", http.StatusBadRequest)
			return
		}

		query := `
            SELECT
                sn.id,
				d.doc_id,
                d.doc_name,
                sn.approver_username,
                sn.status,
                sn.created_at,
                sn.notification_type,
				sn.viewed
            FROM sender_notifications sn
            JOIN documents d ON sn.document_id = d.id
            WHERE sn.uploader_username = $1
            ORDER BY sn.created_at DESC
        `

		rows, err := database.DB.Query(query, username)
		if err != nil {
			log.Printf("[SENDER-NOTIFICATIONS-FAIL] Error querying sender notifications from DB: %v", err)
			sendJSONError(w, "Server error fetching sender notifications", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type SenderNotification struct {
			ID               int       `json:"id"`
			DocID            string    `json:"doc_id"`
			DocName          string    `json:"doc_name"`
			ApproverUsername sql.NullString    `json:"approver_username"`
			Status           string    `json:"status"`
			CreatedAt        time.Time `json:"created_at"`
			NotificationType string    `json:"notification_type"`
			Viewed           bool      `json:"viewed"`
		}

		var notifications []SenderNotification
		for rows.Next() {
			var n SenderNotification
			if err := rows.Scan(&n.ID, &n.DocID, &n.DocName, &n.ApproverUsername, &n.Status, &n.CreatedAt, &n.NotificationType, &n.Viewed); err != nil {
				log.Printf("[SENDER-NOTIFICATIONS-FAIL] Error scanning sender notification row: %v", err)
				continue
			}
			notifications = append(notifications, n)
		}

		if len(notifications) == 0 {
			notifications = make([]SenderNotification, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}

func getUnreadNotificationCountHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string) // Get username from context
	if username == "" {
		sendJSONError(w, "Username is required", http.StatusBadRequest)
		return
	}

	var count int
	query := `SELECT COUNT(*) FROM document_shares WHERE receiver_username = $1 AND viewed = FALSE`
	err := database.DB.QueryRow(query, username).Scan(&count)
	if err != nil {
		log.Printf("[UNREAD-COUNT-FAIL] Error querying unread notification count from DB: %v", err)
		sendJSONError(w, "Server error fetching unread notification count", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})

	
}

func markNotificationAsReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["id"]

	query := `UPDATE document_shares SET viewed = TRUE WHERE id = $1`
	_, err := database.DB.Exec(query, notificationID)
	if err != nil {
		log.Printf("[MARK-AS-READ-FAIL] Error updating notification status in DB: %v", err)
		sendJSONError(w, "Server error marking notification as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Notification marked as read successfully"})
}

func markSenderNotificationAsReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["id"]

	query := `UPDATE sender_notifications SET viewed = TRUE WHERE id = $1`
	_, err := database.DB.Exec(query, notificationID)
	if err != nil {
		log.Printf("[MARK-SENDER-AS-READ-FAIL] Error updating sender notification status in DB: %v", err)
		sendJSONError(w, "Server error marking sender notification as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Sender notification marked as read successfully"})
}

func getUnreadSenderNotificationCountHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string) // Get username from context
	if username == "" {
		sendJSONError(w, "Username is required", http.StatusBadRequest)
		return
	}

	var count int
	query := `SELECT COUNT(*) FROM sender_notifications WHERE uploader_username = $1 AND viewed = FALSE`
	err := database.DB.QueryRow(query, username).Scan(&count)
	if err != nil {
		log.Printf("[UNREAD-SENDER-COUNT-FAIL] Error querying unread sender notification count from DB: %v", err)
		sendJSONError(w, "Server error fetching unread sender notification count", http.StatusInternalServerError)
		return
		}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// GetUnreadTrainingApplicationCountHandler gets the count of unread training applications for a coordinator
func GetUnreadTrainingApplicationCountHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	user, err := database.GetUserByUsername(username)
	if err != nil || user == nil {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if the user is a training coordinator
	if !user.IsTrainingCoordinator {
		// If not a coordinator, they have no unread training applications
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"count": 0})
		return
	}

	coordinatorEmail := user.Email // Get coordinator email from authenticated user's email
	if coordinatorEmail == "" {
		sendJSONError(w, "Coordinator email is required", http.StatusBadRequest)
		return
	}

	var count int
	query := `SELECT COUNT(*) FROM training_applications WHERE coordinator_email = $1 AND viewed = FALSE`
	err = database.DB.QueryRow(query, coordinatorEmail).Scan(&count)
	if err != nil {
		log.Printf("[UNREAD-TRAINING-COUNT-FAIL] Error querying unread training application count from DB: %v", err)
		sendJSONError(w, "Server error fetching unread training application count", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}


// MarkTrainingNotificationAsReadHandler marks a training application notification as read
func MarkTrainingNotificationAsReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationIDStr := vars["id"]
	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		sendJSONError(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	query := `UPDATE training_applications SET viewed = TRUE WHERE id = $1`
	_, err = database.DB.Exec(query, notificationID)
	if err != nil {
		log.Printf("[MARK-TRAINING-AS-READ-FAIL] Error updating training notification status in DB: %v", err)
		sendJSONError(w, "Server error marking training notification as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Training notification marked as read successfully"})
}

func getDocumentStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["documentId"]
	username := r.Context().Value("username").(string) // Get username from context

	if username == "" {
		sendJSONError(w, "Username is required", http.StatusBadRequest)
		return
	}

	var status string
	var isReceiver bool

	// Check if the user is a receiver for this document
	query := `SELECT status FROM document_shares WHERE document_id = (SELECT id FROM documents WHERE doc_id = $1) AND receiver_username = $2`
	err := database.DB.QueryRow(query, documentID, username).Scan(&status)

	if err == sql.ErrNoRows {
		// User is not a receiver for this document
		isReceiver = false
		status = "N/A"
	} else if err != nil {
		log.Printf("Error querying document share status: %v", err)
		sendJSONError(w, "Server error fetching document status", http.StatusInternalServerError)
		return
	} else {
		isReceiver = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"isReceiver": isReceiver,
	})
}

func serveDocumentHandler(uploadsDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := path.Join(uploadsDir, r.URL.Path)
		
		if strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
			w.Header().Set("Content-Type", "application/pdf")
		} else if strings.HasSuffix(strings.ToLower(filePath), ".docx") {
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		}
		http.ServeFile(w, r, filePath)
	})	
}

// sendFCMNotification sends a Firebase Cloud Message to a user's device.
func sendFCMNotification(email, title, body string) {
	user, err := database.GetUserByEmail(email)
	if err != nil || user == nil || !user.FCMToken.Valid {
		log.Printf("Could not send FCM notification to %s: user not found or FCM token missing/invalid. Error: %v", email, err)
		return
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: user.FCMToken.String,
	}

	response, err := fcmClient.Send(context.Background(), message)
	if err != nil {
		log.Printf("Failed to send FCM message to %s: %v", email, err)
		return
	}
	log.Printf("Successfully sent FCM message to %s: %s", email, response)
}