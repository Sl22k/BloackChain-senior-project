package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// GetDocumentIdByHash returns document ID for a given hash
func (s *SmartContract) GetDocumentIdByHash(ctx contractapi.TransactionContextInterface, hash string) (string, error) {
	if hash == "" {
		return "", fmt.Errorf("hash cannot be empty")
	}

	iterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return "", fmt.Errorf("error retrieving documents: %v", err)
	}
	defer iterator.Close()

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return "", err
		}
		var doc Document
		if json.Unmarshal(queryResponse.Value, &doc) == nil {
			for _, version := range doc.Versions {
				if version.Hash == hash {
					return doc.ID, nil
				}
			}
		}
	}
	return "", fmt.Errorf("document with hash %s not found", hash)
}

// DocumentExists checks if a document exists by ID
func (s *SmartContract) DocumentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	if err := validateNonEmptyString("document ID", id); err != nil {
		return false, err
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return data != nil, nil
}

// GetDocHistory returns the history of a document
func (s *SmartContract) GetDocHistory(ctx contractapi.TransactionContextInterface, id string) ([]map[string]interface{}, error) {
	if err := validateNonEmptyString("document ID", id); err != nil {
		return nil, err
	}

	historyIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for document %s: %v", id, err)
	}
	defer historyIterator.Close()

	var history []map[string]interface{}
	for historyIterator.HasNext() {
		modification, err := historyIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate history: %v", err)
		}

		if modification.Value != nil {
			var doc Document
			if err := json.Unmarshal(modification.Value, &doc); err != nil {
				return nil, fmt.Errorf("failed to unmarshal document: %v", err)
			}

			historyEntry := map[string]interface{}{
				"txId":        modification.TxId,
				"timestamp":   modification.Timestamp.AsTime().Format(time.RFC3339),
				"isDelete":    modification.IsDelete,
				"document":    doc,
			}
			history = append(history, historyEntry)
		}
	}

	return history, nil
}

// GetAllDocuments returns all documents in the system
func (s *SmartContract) GetAllDocuments(ctx contractapi.TransactionContextInterface) ([]*Document, error) {
	queryString := `{
		"selector": {
			"ID": {
				"$exists": true
			}
		}
	}`
	return s.getQueryResult(ctx, queryString)
}

// GetDocumentsByEditor returns documents where the user is an editor
func (s *SmartContract) GetDocumentsByEditor(ctx contractapi.TransactionContextInterface, editorId string) ([]*Document, error) {
	if err := validateNonEmptyString("editor ID", editorId); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"Editors": {
				"$in": ["%s"]
			}
		}
	}`, editorId)

	return s.getQueryResult(ctx, queryString)
}

// GetDocumentsByApprover returns documents where the user is an approver
func (s *SmartContract) GetDocumentsByApprover(ctx contractapi.TransactionContextInterface, approverId string) ([]*Document, error) {
	if err := validateNonEmptyString("approver ID", approverId); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"ApprovalsMap.%s": {
				"$exists": true
			}
		}
	}`, approverId)

	return s.getQueryResult(ctx, queryString)
}

// GetPendingApprovals returns documents with pending approvals for a specific user
func (s *SmartContract) GetPendingApprovals(ctx contractapi.TransactionContextInterface, approverId string) ([]*Document, error) {
	if err := validateNonEmptyString("approver ID", approverId); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"ApprovalsMap.%s.Status": "PENDING"
		}
	}`, approverId)

	return s.getQueryResult(ctx, queryString)
}

// GetDocumentSummary returns a summary of a document
func (s *SmartContract) GetDocumentSummary(ctx contractapi.TransactionContextInterface, id string) (map[string]interface{}, error) {
	if err := validateNonEmptyString("document ID", id); err != nil {
		return nil, err
	}

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.createDocumentSummary(doc), nil
}


// UpdateValidDecisions updates the list of valid decisions for a document using sophisticated approach
func (s *SmartContract) UpdateValidDecisions(ctx contractapi.TransactionContextInterface, documentID, newDecisionsJSON, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Parse new decisions
	var newDecisions []string
	if err := json.Unmarshal([]byte(newDecisionsJSON), &newDecisions); err != nil {
		return fmt.Errorf("invalid newDecisions JSON: %v", err)
	}

	// Validate new decisions
	if len(newDecisions) == 0 {
		return fmt.Errorf("at least one valid decision must be provided")
	}

	// Load document and validate authorization
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	if err := s.validateAuthorization(invokerId, doc, "edit"); err != nil {
		return err
	}

	// Store old decisions for event
	oldDecisions := make([]string, len(doc.ValidDecisions))
	copy(oldDecisions, doc.ValidDecisions)

	// Update valid decisions
	doc.ValidDecisions = newDecisions

	// Create new version using the same sophisticated approach as SubmitNewVersion with proper change detection
	newState := NewVersionState{
		ContentHash:    doc.Versions[len(doc.Versions)-1].Hash, // Keep same content hash
		StageUpdates:   map[string]StageUpdate{},              // No stage updates for decision changes
		ValidDecisions: newDecisions,                          // New valid decisions
		ResetApprovals: false,                                 // Don't reset approvals for decision changes
		ChangeReason:   "Valid decisions updated",
	}

	// Detect changes properly (same as SubmitNewVersion)
	changes := s.detectChanges(*doc, newState)

	// Apply version changes with calculated changes
	if err := s.applyVersionChanges(doc, newState, changes, timestamp, invokerId); err != nil {
		return err
	}

	// Save document
	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Emit event
	description := fmt.Sprintf("Valid decisions updated for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"oldDecisions": oldDecisions,
		"newDecisions": newDecisions,
	}

	return s.emitEvent(ctx, "ValidDecisionsUpdated", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// GetDocumentsByDateRange returns documents created within a date range
func (s *SmartContract) GetDocumentsByDateRange(ctx contractapi.TransactionContextInterface, startDate, endDate string) ([]*Document, error) {
	if err := validateNonEmptyString("startDate", startDate); err != nil {
		return nil, err
	}
	if err := validateNonEmptyString("endDate", endDate); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"CreatedTimestamp": {
				"$gte": "%s",
				"$lte": "%s"
			}
		}
	}`, startDate, endDate)

	return s.getQueryResult(ctx, queryString)
}

// GetRecentDocuments returns the most recent documents with a limit
func (s *SmartContract) GetRecentDocuments(ctx contractapi.TransactionContextInterface, limit string) ([]*Document, error) {
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt <= 0 {
		return nil, fmt.Errorf("invalid limit parameter: %v", err)
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	sortDocuments(allDocs, SortNewest)
	return limitDocuments(allDocs, limitInt), nil
}

// GetDocumentsByUploaderSorted returns documents by uploader with sorting
func (s *SmartContract) GetDocumentsByUploaderSorted(ctx contractapi.TransactionContextInterface, uploader, sortOrder string) ([]*Document, error) {
	if err := validateNonEmptyString("uploader", uploader); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"Uploader": "%s"
		}
	}`, uploader)

	docs, err := s.getQueryResult(ctx, queryString)
	if err != nil {
		return nil, err
	}

	sortDocuments(docs, parseSortOrder(sortOrder))
	return docs, nil
}

// GetDocumentsWithPendingApprovals returns documents with pending approvals
func (s *SmartContract) GetDocumentsWithPendingApprovals(ctx contractapi.TransactionContextInterface) ([]*Document, error) {
	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	return filterDocumentsByPendingApprovals(allDocs), nil
}

// GetDocumentStats returns statistics about all documents
func (s *SmartContract) GetDocumentStats(ctx contractapi.TransactionContextInterface) (map[string]interface{}, error) {
	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	return s.calculateDocumentStatistics(allDocs), nil
}

// getQueryResult executes a CouchDB query and returns documents
func (s *SmartContract) getQueryResult(ctx contractapi.TransactionContextInterface, queryString string) ([]*Document, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer resultsIterator.Close()

	var documents []*Document
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var doc Document
		if err := json.Unmarshal(queryResponse.Value, &doc); err != nil {
			return nil, err
		}

		// Fix deadline status to be consistent with QueryDocumentStatus
		if deadlineStatus, err := s.GetDeadlineStatus(ctx, doc.ID); err == nil {
			doc.DeadlineStatus = deadlineStatus
		}

		documents = append(documents, &doc)
	}

	return documents, nil
}