package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type ApproverStatus string

const (
	Pending  ApproverStatus = "PENDING"
	Approved ApproverStatus = "APPROVED"
	Rejected ApproverStatus = "REJECTED"
)

type Decision struct {
	Status  ApproverStatus `json:"Status"`
	Comment string         `json:"Comment"`
}

type DocumentVersion struct {
	Version   int    `json:"Version"`
	Hash      string `json:"Hash"`
	Submitter string `json:"Submitter"`
	Timestamp int64  `json:"Timestamp"`
}

type Document struct {
	ID                string              `json:"ID"`
	LatestVersion     int                 `json:"LatestVersion"`
	Versions          []DocumentVersion   `json:"Versions"`
	Uploader          string              `json:"Uploader"`
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"`
	ValidDecisions    []string            `json:"ValidDecisions"`
	ApprovedCount     int                 `json:"ApprovedCount"`
	RejectedCount     int                 `json:"RejectedCount"`
	PendingCount      int                 `json:"PendingCount"`
}

// StatusResponse gives document status summary
type StatusResponse struct {
	ID                string              `json:"ID"`
	LatestVersion     int                 `json:"LatestVersion"`
	Versions          []DocumentVersion   `json:"Versions"`
	Uploader          string              `json:"Uploader"`
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"`
	DecisionCounts    map[string]int      `json:"DecisionCounts"`
	ValidDecisions    []string            `json:"ValidDecisions"`
	ApprovedCount     int                 `json:"ApprovedCount"`
	RejectedCount     int                 `json:"RejectedCount"`
	PendingCount      int                 `json:"PendingCount"`
}

type SmartContract struct {
	contractapi.Contract
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// No initial documents in the ledger
	return nil
}

func (s *SmartContract) SubmitDocument(ctx contractapi.TransactionContextInterface, id, hash, uploader, approversJSON, validDecisionsJSON, invokerId string, resetApprovals bool) error {
	exists, err := s.DocumentExists(ctx, id)
	if err != nil {
		return err
	}

	var doc Document
	if exists {
		// Document exists, so we are creating a new version
		data, err := ctx.GetStub().GetState(id)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return err
		}

		// Security Check 1: Verify the submitter is an editor (REMOVED)
		/*
		isEditor := false
		for _, editor := range doc.Editors {
			if editor == invokerId {
				isEditor = true
				break
			}
		}
		if !isEditor {
			return fmt.Errorf("uploader %s is not an editor for document %s", invokerId, id)
		}
		*/

		// Security Check 2: Verify the hash is different from the latest version
		if doc.Versions[doc.LatestVersion-1].Hash == hash {
			return fmt.Errorf("new version hash is the same as the latest version hash")
		}

		doc.LatestVersion++
		if resetApprovals {
			// Reset approvals for the new version
			for approver := range doc.ApprovalsMap {
				doc.ApprovalsMap[approver] = Decision{Status: Pending, Comment: ""}
			}
		}
	} else {
		// This is a new document
		var approvers []string
		if err := json.Unmarshal([]byte(approversJSON), &approvers); err != nil {
			return fmt.Errorf("invalid approvers JSON: %v", err)
		}
		var validDecisions []string
		if err := json.Unmarshal([]byte(validDecisionsJSON), &validDecisions); err != nil {
			return fmt.Errorf("invalid validDecisions JSON: %v", err)
		}
		if validDecisions == nil {
			validDecisions = []string{}
		}

		approvalsMap := make(map[string]Decision)
		for _, approver := range approvers {
			approvalsMap[approver] = Decision{Status: Pending, Comment: ""}
		}

		doc = Document{
			ID:                id,
			LatestVersion:     1,
			Uploader:          uploader,
			PrivilegedEditors: []string{uploader}, // Initialize with the uploader as a privileged editor
			Editors:           []string{uploader}, // The original uploader is the first editor
			ValidDecisions:    validDecisions,
			ApprovalsMap:      approvalsMap,
			ApprovedCount:     0,
			RejectedCount:     0,
			PendingCount:      len(approvers),
		}
	}

	// Add the new version to the history
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	newVersion := DocumentVersion{
		Version:   doc.LatestVersion,
		Hash:      hash,
		Submitter: uploader,
		Timestamp: timestamp.GetSeconds(),
	}
	doc.Versions = append(doc.Versions, newVersion)

	// Create the hash-to-ID index
	hashKey := fmt.Sprintf("hash->%s", hash)
	if err := ctx.GetStub().PutState(hashKey, []byte(id)); err != nil {
		return fmt.Errorf("failed to create hash-to-ID index: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, data)
	if err != nil {
		return err
	}

	// Add event emission for SubmitDocument
	eventPayload := map[string]interface{}{
		"documentID": doc.ID,
		"version":    doc.LatestVersion,
		"eventType":  "DocumentSubmitted",
		"submitter":  uploader,
		"timestamp":  newVersion.Timestamp,
		"details": map[string]string{
			"hash": hash,
		},
	}

	eventBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload for SubmitDocument: %v", err)
	}

	err = ctx.GetStub().SetEvent("DocumentUpdated", eventBytes)
	if err != nil {
		return fmt.Errorf("failed to set DocumentUpdated event for SubmitDocument: %v", err)
	}

	return nil
}

func (s *SmartContract) ApproveDocument(ctx contractapi.TransactionContextInterface, id, approver, decision, comment string) error {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}

	// Check if the approver is authorized
	decisionInfo, ok := doc.ApprovalsMap[approver]
	if !ok {
		return fmt.Errorf("%s not authorized to decide", approver)
	}

	// Check if the approver has already made a decision
	if decisionInfo.Status != Pending {
		return fmt.Errorf("%s already decided", approver)
	}

	// Validate the decision against the document's valid decisions
	validDecision := false
	for _, d := range doc.ValidDecisions {
		if decision == d {
			validDecision = true
			break
		}
	}
	if !validDecision {
		return fmt.Errorf("invalid decision '%s' for this document", decision)
	}

	oldStatus := decisionInfo.Status // Store old status before updating
	decisionInfo.Status = ApproverStatus(decision)
	decisionInfo.Comment = comment
	doc.ApprovalsMap[approver] = decisionInfo

	// Update counts based on the decision
	if oldStatus == Pending { // If it was pending, decrement pending count
		doc.PendingCount--
	}
	if ApproverStatus(decision) == Approved {
		doc.ApprovedCount++
	} else if ApproverStatus(decision) == Rejected {
		doc.RejectedCount++
	}

	updated, _ := json.Marshal(doc)

	err = ctx.GetStub().PutState(id, updated)
	if err != nil {
		return err
	}

	// Add event emission for ApproveDocument
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	eventPayload := map[string]interface{}{
		"documentID": doc.ID,
		"version":    doc.LatestVersion,
		"eventType":  "ApprovalMade",
		"approver":   approver,
		"decision":   decision,
		"comment":    comment,
		"timestamp":  timestamp.GetSeconds(),
	}

	eventBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload for ApproveDocument: %v", err)
	}

	err = ctx.GetStub().SetEvent("DocumentUpdated", eventBytes)
	if err != nil {
		return fmt.Errorf("failed to set DocumentUpdated event for ApproveDocument: %v", err)
	}

	return nil
}

func (s *SmartContract) QueryDocumentStatus(ctx contractapi.TransactionContextInterface, id string) (*StatusResponse, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}

	resp := &StatusResponse{
		ID:                doc.ID,
		LatestVersion:     doc.LatestVersion,
		Versions:          doc.Versions,
		Uploader:          doc.Uploader,
		PrivilegedEditors: doc.PrivilegedEditors,
		Editors:           doc.Editors,
		ApprovalsMap:      doc.ApprovalsMap,
		ValidDecisions:    doc.ValidDecisions,
		ApprovedCount:     doc.ApprovedCount,
		RejectedCount:     doc.RejectedCount,
		PendingCount:      doc.PendingCount,
		DecisionCounts:    make(map[string]int),
	}

	return resp, nil
}

func (s *SmartContract) AddEditor(ctx contractapi.TransactionContextInterface, id, newEditor, invokerId string) error {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Security Check: In a production environment, you would want to ensure that only an existing editor
	// can add a new one. This requires a robust way to map the transaction submitter's certificate
	// identity (e.g., 'isadmin') to an application-level identity (e.g., 'yousif@uni.edu').
	// This often involves an external identity management system or custom attributes in the certificate.
	// For this demonstration, we are using a simulated invokerId to test the logic.
	isEditor := false
	for _, editor := range doc.Editors {
		if editor == invokerId {
			isEditor = true
			break
		}
	}
	if !isEditor {
		return fmt.Errorf("submitter %s is not an editor for document %s", invokerId, id)
	}

	// Add the new editor
	doc.Editors = append(doc.Editors, newEditor)

	updated, _ := json.Marshal(doc)
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) UpdateDocumentApprovers(ctx contractapi.TransactionContextInterface, documentID string, newApproversJSON string, invokerId string) error {
	// 1. Retrieve the document
	data, err := ctx.GetStub().GetState(documentID)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", documentID)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// 2. Security Check: Verify invokerId is a Privileged Editor for this document
	isPrivilegedEditor := false
	for _, editor := range doc.PrivilegedEditors {
		if editor == invokerId {
			isPrivilegedEditor = true
			break
		}
	}
	if !isPrivilegedEditor {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, documentID)
	}

	// 3. Parse new approvers
	var newApprovers []string
	if err := json.Unmarshal([]byte(newApproversJSON), &newApprovers); err != nil {
		return fmt.Errorf("invalid newApprovers JSON: %v", err)
	}

	// 4. Update ApprovalsMap based on newApprovers
	//    - Add new approvers with PENDING status
	//    - Remove approvers no longer in the list (their past decisions remain recorded in history)
	updatedApprovalsMap := make(map[string]Decision)
	for _, approver := range newApprovers {
		if existingDecision, ok := doc.ApprovalsMap[approver]; ok {
			// Carry over existing decision if approver was already there
			updatedApprovalsMap[approver] = existingDecision
		} else {
			// New approver, set to PENDING
			updatedApprovalsMap[approver] = Decision{Status: Pending, Comment: ""}
		}
	}
	doc.ApprovalsMap = updatedApprovalsMap

	// 5. Persist the updated document
	updatedDocBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(documentID, updatedDocBytes)
	if err != nil {
		return err
	}

	// Add event emission for UpdateDocumentApprovers
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	eventPayload := map[string]interface{}{
		"documentID":   doc.ID,
		"version":      doc.LatestVersion,
		"eventType":    "ApproversUpdated",
		"invoker":      invokerId,
		"newApprovers": newApprovers,
		"timestamp":    timestamp.GetSeconds(),
	}

	eventBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload for UpdateDocumentApprovers: %v", err)
	}

	err = ctx.GetStub().SetEvent("DocumentUpdated", eventBytes)
	if err != nil {
		return fmt.Errorf("failed to set DocumentUpdated event for UpdateDocumentApprovers: %v", err)
	}

	return nil
}

func (s *SmartContract) GetDocumentIdByHash(ctx contractapi.TransactionContextInterface, hash string) (string, error) {
	hashKey := fmt.Sprintf("hash->%s", hash)
	idBytes, err := ctx.GetStub().GetState(hashKey)
	if err != nil || idBytes == nil {
		return "", fmt.Errorf("no document found with hash %s", hash)
	}
	return string(idBytes), nil
}

func (s *SmartContract) DocumentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

// GetAllDocuments returns all documents on the ledger
func (s *SmartContract) GetAllDocuments(ctx contractapi.TransactionContextInterface) ([]*Document, error) {
	iterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	var documents []*Document
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		// Skip the hash-to-ID index entries
		if len(queryResponse.Key) > 5 && string(queryResponse.Key[:5]) == "hash->" {
			continue
		}

		var document Document
		err = json.Unmarshal(queryResponse.Value, &document)
		if err != nil {
			// This might happen if there are other non-document entries
			// For now, we'll just skip them
			continue
		}
		if document.ValidDecisions == nil {
			document.ValidDecisions = []string{}
		}
		documents = append(documents, &document)
	}

	return documents, nil
}

// GetHistory returns the modification history of a document
func (s *SmartContract) GetHistory(ctx contractapi.TransactionContextInterface, documentID string) ([]map[string]interface{}, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(documentID)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var history []map[string]interface{}
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var record map[string]interface{}
		if err := json.Unmarshal(response.Value, &record); err != nil {
			return nil, err
		}

		historyEntry := map[string]interface{}{
			"TxId":      response.TxId,
			"Timestamp": time.Unix(response.Timestamp.GetSeconds(), int64(response.Timestamp.GetNanos())).Format(time.RFC3339),
			"IsDelete":  response.IsDelete,
			"Value":     record,
		}
		history = append(history, historyEntry)
	}

	return history, nil
}


