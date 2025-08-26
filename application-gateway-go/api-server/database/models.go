package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID                   int    `json:"id"`
	Username             string `json:"username"`
	Email                string `json:"email"`
	Password             string `json:"-"` // Stored hashed, never expose
	FCMToken             sql.NullString `json:"fcm_token"`
	Role                 string `json:"role"`
	IsTrainingCoordinator bool   `json:"is_training_coordinator"`
}

// Updated Document struct to match the new database schema
type Document struct {
	ID             int       `json:"id"`
	DocID          string    `json:"doc_id"`
	DocName        string    `json:"doc_name"`
	UploadTime     time.Time `json:"upload_time"`
	DocPath        string    `json:"doc_path"`
	Hash           string    `json:"hash"`
	SenderUsername string    `json:"sender_username"`
	ApprovedCount  int       `json:"approved_count"`
	RejectedCount  int       `json:"rejected_count"`
	PendingCount   int       `json:"pending_count"`
	ApprovalsMap  map[string]string `json:"approvals_map"` // Added for individual statuses
}

// New struct for training applications
type TrainingApplication struct {
	ID               int       `json:"id"`
	StudentName      string    `json:"studentName"`
	StudentID        string    `json:"studentId"`
	CPR              string    `json:"cpr"`
	Nationality      string    `json:"nationality"`
	Telephone        string    `json:"telephone"`
	Email            string    `json:"email"`
	CourseCode       string    `json:"courseCode"`
	CoordinatorEmail string    `json:"coordinatorEmail"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	Viewed           bool      `json:"viewed"`
}

var DB *sql.DB

// Updated InitDB to reflect the new database schema and handle ENUM creation correctly
func InitDB() {
	connStr := "postgresql://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable"
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL!")

	// This table remains unchanged
	createUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		fcm_token TEXT,
		role VARCHAR(50) NOT NULL DEFAULT 'student',
		is_training_coordinator BOOLEAN DEFAULT FALSE
	);
	`
	_, err = DB.Exec(createUsersTableSQL)
	if err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}

	// Add role column if it doesn't exist
	alterUsersTableSQL := `
	ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) NOT NULL DEFAULT 'student';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS is_training_coordinator BOOLEAN DEFAULT FALSE;
	`
	_, err = DB.Exec(alterUsersTableSQL)
	if err != nil {
		log.Fatalf("Error altering users table: %v", err)
	}
	log.Println("Users table checked/created successfully.")

	// Updated documents table schema
	createDocumentsTableSQL := `
	CREATE TABLE IF NOT EXISTS documents (
		id SERIAL PRIMARY KEY,
		doc_id TEXT UNIQUE NOT NULL,
		doc_name TEXT,
		doc_path TEXT,
		hash TEXT,
		uploader_username TEXT REFERENCES users(username),
		upload_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		approved_count INTEGER DEFAULT 0,
		rejected_count INTEGER DEFAULT 0,
		pending_count INTEGER DEFAULT 0
	);
	`
	_, err = DB.Exec(createDocumentsTableSQL)
	if err != nil {
		log.Fatalf("Error creating documents table: %v", err)
	}
	log.Println("Documents table checked/created successfully.")

	// Create ENUM type for document_shares status, handling if it already exists
	createApproverStatusTypeSQL := `CREATE TYPE approver_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED')`
	_, err = DB.Exec(createApproverStatusTypeSQL)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42710" { // 42710 is duplicate_object
			log.Println("Approver status ENUM type already exists, skipping creation.")
		} else {
			log.Fatalf("Error creating approver_status type: %v", err)
		}
	} else {
		log.Println("Approver status ENUM type created successfully.")
	}

	// Updated document_shares table schema
	createDocumentSharesTableSQL := `
	CREATE TABLE IF NOT EXISTS document_shares (
		id SERIAL PRIMARY KEY,
		document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
		receiver_username TEXT REFERENCES users(username),
		status approver_status DEFAULT 'PENDING',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(document_id, receiver_username)
	);
	`
	_, err = DB.Exec(createDocumentSharesTableSQL)
	if err != nil {
		log.Fatalf("Error creating document_shares table: %v", err)
	}
	log.Println("Document shares table checked/created successfully.")

	// Add the 'viewed' and 'comment' columns to document_shares if they don't exist
	alterDocumentSharesTableSQL := `
	ALTER TABLE document_shares ADD COLUMN IF NOT EXISTS viewed BOOLEAN DEFAULT FALSE;
	ALTER TABLE document_shares ADD COLUMN IF NOT EXISTS comment TEXT;
	`
	_, err = DB.Exec(alterDocumentSharesTableSQL)
	if err != nil {
		log.Fatalf("Error altering document_shares table: %v", err)
	}
	log.Println("Document shares table altered successfully.")

	// New sender_notifications table
	createSenderNotificationsTableSQL := `
	CREATE TABLE IF NOT EXISTS sender_notifications (
		id SERIAL PRIMARY KEY,
		document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
		uploader_username TEXT REFERENCES users(username),
		approver_username TEXT REFERENCES users(username),
		status approver_status,
		doc_name TEXT,
		notification_type TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		viewed BOOLEAN DEFAULT FALSE
	);
	`
	_, err = DB.Exec(createSenderNotificationsTableSQL)
	if err != nil {
		log.Fatalf("Error creating sender_notifications table: %v", err)
	}
	log.Println("Sender notifications table checked/created successfully.")

	// Add the 'viewed' column to sender_notifications if it doesn't exist
	alterSenderNotificationsTableSQL := `
	ALTER TABLE sender_notifications ADD COLUMN IF NOT EXISTS viewed BOOLEAN DEFAULT FALSE;
	`
	_, err = DB.Exec(alterSenderNotificationsTableSQL)
	if err != nil {
		log.Fatalf("Error altering sender_notifications table: %v", err)
	}
	log.Println("Sender notifications table altered successfully.")

    // --- Add new training_applications table ---
	createTrainingApplicationsTableSQL := `
	CREATE TABLE IF NOT EXISTS training_applications (
		id SERIAL PRIMARY KEY,
		student_name TEXT NOT NULL,
		student_id TEXT NOT NULL,
		cpr TEXT NOT NULL,
		nationality TEXT NOT NULL,
		telephone TEXT NOT NULL,
		email TEXT NOT NULL,
		course_code TEXT NOT NULL,
		coordinator_email TEXT NOT NULL,
		status VARCHAR(50) DEFAULT 'pending',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		viewed BOOLEAN DEFAULT FALSE
	);
	`
	_, err = DB.Exec(createTrainingApplicationsTableSQL)
	if err != nil {
		log.Fatalf("Error creating training_applications table: %v", err)
	}
	log.Println("Training applications table checked/created successfully.")

	// Add the 'viewed' column to training_applications if it doesn't exist
	alterTrainingApplicationsTableSQL := `
	ALTER TABLE training_applications ADD COLUMN IF NOT EXISTS viewed BOOLEAN DEFAULT FALSE;
	`
	_, err = DB.Exec(alterTrainingApplicationsTableSQL)
	if err != nil {
		log.Fatalf("Error altering training_applications table: %v", err)
	}
	log.Println("Training applications table altered successfully.")
}

// UpdateCoordinatorStatusFromJSON reads coordinator emails from a JSON file
// and updates the is_training_coordinator status in the database for matching users.
func UpdateCoordinatorStatusFromJSON(jsonFilePath string) error {
	log.Printf("Attempting to update coordinator status from JSON file: %s", jsonFilePath)

	// Read the JSON file
	data, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to read coordinator JSON file: %w", err)
	}

	// Define a map to unmarshal the JSON into
	var coordinators map[string]struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(data, &coordinators); err != nil {
		return fmt.Errorf("failed to unmarshal coordinator JSON: %w", err)
	}

	// Collect unique coordinator emails
	uniqueCoordinatorEmails := make(map[string]bool)
	for _, coord := range coordinators {
		if coord.Email != "-" { // Skip placeholder emails
			uniqueCoordinatorEmails[strings.ToLower(coord.Email)] = true
		}
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error

	// First, set all users' is_training_coordinator to FALSE
	// This ensures that if an email is removed from coordinators.json,
	// their status is correctly revoked.
	_, err = tx.Exec("UPDATE users SET is_training_coordinator = FALSE")
	if err != nil {
		return fmt.Errorf("failed to reset all is_training_coordinator statuses: %w", err)
	}
	log.Println("Reset all users' is_training_coordinator status to FALSE.")

	// Then, set is_training_coordinator to TRUE and role to 'coordinator' for matching emails
	for email := range uniqueCoordinatorEmails {
		_, err := tx.Exec("UPDATE users SET is_training_coordinator = TRUE, role = 'coordinator' WHERE LOWER(email) = $1", email)
		if err != nil {
			return fmt.Errorf("failed to update is_training_coordinator for %s: %w", email, err)
		}
		log.Printf("Updated user with email %s: set is_training_coordinator to TRUE and role to 'coordinator'.", email)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Successfully updated is_training_coordinator status for all relevant users.")
	return nil
}

func CreateUser(user *User) error {
	query := "INSERT INTO users (username, email, password, fcm_token, role) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	err := DB.QueryRow(query, user.Username, user.Email, user.Password, user.FCMToken, user.Role).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	query := "SELECT id, username, email, password, fcm_token, role, is_training_coordinator FROM users WHERE username = $1"
	err := DB.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FCMToken, &user.Role, &user.IsTrainingCoordinator)
	if err == sql.ErrNoRows {
		return nil, nil // User not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func GetUserByEmail(email string) (*User, error) {
	user := &User{}
	// Use LOWER() for case-insensitive search and trim space for robustness
	query := "SELECT id, username, email, password, fcm_token, role, is_training_coordinator FROM users WHERE LOWER(email) = LOWER(TRIM($1))"
	err := DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FCMToken, &user.Role, &user.IsTrainingCoordinator)
	if err == sql.ErrNoRows {
		return nil, nil // User not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func UpdateUserFCMToken(username string, fcmToken sql.NullString) error {
	query := "UPDATE users SET fcm_token = $1 WHERE username = $2"
	_, err := DB.Exec(query, fcmToken, username)
	if err != nil {
		return fmt.Errorf("failed to update FCM token for user %s: %w", username, err)
	}
	return nil
}

// --- New function to create a training application ---
func CreateTrainingApplication(app *TrainingApplication) error {
	query := `
	INSERT INTO training_applications 
	(student_name, student_id, cpr, nationality, telephone, email, course_code, coordinator_email, status, viewed)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`

	err := DB.QueryRow(query, 
		app.StudentName, app.StudentID, app.CPR, app.Nationality, app.Telephone, app.Email, app.CourseCode, app.CoordinatorEmail, "pending", false,
	).Scan(&app.ID, &app.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create training application: %w", err)
	}
	return nil
}

// New function to get training applications by coordinator email
func GetTrainingApplicationsByCoordinatorEmail(coordinatorEmail string) ([]TrainingApplication, error) {
	query := `
	SELECT id, student_name, student_id, cpr, nationality, telephone, email, course_code, coordinator_email, status, created_at, viewed
	FROM training_applications
	WHERE coordinator_email = $1
	ORDER BY created_at DESC
	`

	rows, err := DB.Query(query, coordinatorEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get training applications: %w", err)
	}
	defer rows.Close()

	var applications []TrainingApplication
	for rows.Next() {
		var app TrainingApplication
		err := rows.Scan(
			&app.ID, &app.StudentName, &app.StudentID, &app.CPR, &app.Nationality, &app.Telephone, &app.Email, &app.CourseCode, &app.CoordinatorEmail, &app.Status, &app.CreatedAt, &app.Viewed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan training application row: %w", err)
		}
		applications = append(applications, app)
	}

	return applications, nil
}

// New function to delete a training application
func DeleteTrainingApplication(id int) error {
	query := `DELETE FROM training_applications WHERE id = $1`
	_, err := DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete training application: %w", err)
	}
	return nil
}

// New function to get a training application by student ID
func GetTrainingApplicationByStudentID(studentID string) (*TrainingApplication, error) {
	query := `
	SELECT id, student_name, student_id, cpr, nationality, telephone, email, course_code, coordinator_email, status, created_at, viewed
	FROM training_applications
	WHERE student_id = $1
	`

	var app TrainingApplication
	err := DB.QueryRow(query, studentID).Scan(
		&app.ID, &app.StudentName, &app.StudentID, &app.CPR, &app.Nationality, &app.Telephone, &app.Email, &app.CourseCode, &app.CoordinatorEmail, &app.Status, &app.CreatedAt, &app.Viewed,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Application not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get training application by student ID: %w", err)
	}
	return &app, nil
}

// New function to get a training application by student email
func GetTrainingApplicationByEmail(email string) (*TrainingApplication, error) {
	query := `
	SELECT id, student_name, student_id, cpr, nationality, telephone, email, course_code, coordinator_email, status, created_at, viewed
	FROM training_applications
	WHERE email = $1
	`

	var app TrainingApplication
	err := DB.QueryRow(query, email).Scan(
		&app.ID, &app.StudentName, &app.StudentID, &app.CPR, &app.Nationality, &app.Telephone, &app.Email, &app.CourseCode, &app.CoordinatorEmail, &app.Status, &app.CreatedAt, &app.Viewed,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Application not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get training application by email: %w", err)
	}
	return &app, nil
}

func UpdateTrainingApplicationStatus(id int, status string) error {
    query := `UPDATE training_applications SET status = $1 WHERE id = $2`
    _, err := DB.Exec(query, status, id)
    if err != nil {
        return fmt.Errorf("failed to update training application status: %w", err)
    }
    return nil
}