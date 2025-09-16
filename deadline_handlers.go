package chaincode

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SetDocumentDeadline sets or updates deadline for a document
func (s *SmartContract) SetDocumentDeadline(ctx contractapi.TransactionContextInterface, id, invokerId, deadlineHours string) error {
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("only document owner or privileged editors can set deadlines")
	}

	hours, err := validateDeadlineHours(deadlineHours)
	if err != nil {
		return err
	}

	deadlineRFC3339 := calculateDeadlineFromHours(hours)

	doc.DeadlineConfig = createDefaultDeadlineConfig(true, deadlineRFC3339, hours)

	status, hoursRemaining, _ := calculateDeadlineStatus(deadlineRFC3339, doc.DeadlineConfig.WarningThreshold)
	doc.DeadlineStatus = createDefaultDeadlineStatus(id, status, deadlineRFC3339, hoursRemaining)

	doc.LastModifiedTimestamp = time.Now().Format(time.RFC3339)
	s.ensureDocumentArraysInitialized(doc)
	return s.saveDocument(ctx, doc)
}

// GetDeadlineStatus returns current deadline status for a document
func (s *SmartContract) GetDeadlineStatus(ctx contractapi.TransactionContextInterface, id string) (DeadlineStatus, error) {
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return DeadlineStatus{}, err
	}

	if !doc.DeadlineConfig.Enabled {
		return createDefaultDeadlineStatus(id, "no_deadlines", "", 0), nil
	}

	status, hoursRemaining, err := calculateDeadlineStatus(doc.DeadlineConfig.DocumentDeadline, doc.DeadlineConfig.WarningThreshold)
	if err != nil {
		return DeadlineStatus{}, err
	}

	result := createDefaultDeadlineStatus(id, status, doc.DeadlineConfig.DocumentDeadline, hoursRemaining)

	if doc.DeadlineStatus.WarningsTriggered != nil {
		result.WarningsTriggered = doc.DeadlineStatus.WarningsTriggered
	}
	if doc.DeadlineStatus.BreachesRecorded != nil {
		result.BreachesRecorded = doc.DeadlineStatus.BreachesRecorded
	}

	return s.ensureDeadlineArraysInitialized(result), nil
}

// GetDocumentsWithUpcomingDeadlines returns documents with deadlines approaching within specified hours
func (s *SmartContract) GetDocumentsWithUpcomingDeadlines(ctx contractapi.TransactionContextInterface, hoursAhead string) ([]*Document, error) {
	hoursAheadInt, err := strconv.Atoi(hoursAhead)
	if err != nil {
		return nil, fmt.Errorf("invalid hoursAhead parameter: %v", err)
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var upcomingDeadlineDocs []*Document

	for _, doc := range allDocs {
		if !doc.DeadlineConfig.Enabled {
			continue
		}

		inRange, hoursRemaining, err := isDeadlineInRange(doc.DeadlineConfig.DocumentDeadline, hoursAheadInt)
		if err != nil {
			continue
		}

		if inRange {
			status, _, _ := calculateDeadlineStatus(doc.DeadlineConfig.DocumentDeadline, doc.DeadlineConfig.WarningThreshold)
			doc.DeadlineStatus = createDefaultDeadlineStatus(doc.ID, status, doc.DeadlineConfig.DocumentDeadline, int64(hoursRemaining))
			doc.DeadlineStatus = s.ensureDeadlineArraysInitialized(doc.DeadlineStatus)
			upcomingDeadlineDocs = append(upcomingDeadlineDocs, doc)
		}
	}
	return upcomingDeadlineDocs, nil
}

// RemoveDocumentDeadline removes deadline configuration from a document
func (s *SmartContract) RemoveDocumentDeadline(ctx contractapi.TransactionContextInterface, id, invokerId string) error {
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("only document owner or privileged editors can remove deadlines")
	}

	doc.DeadlineConfig = createDefaultDeadlineConfig(false, "", 0)
	doc.DeadlineStatus = createDefaultDeadlineStatus(id, "not_configured", "", 0)
	doc.LastModifiedTimestamp = time.Now().Format(time.RFC3339)

	return s.saveDocument(ctx, doc)
}