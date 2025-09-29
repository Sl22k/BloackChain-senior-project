package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SetStageDeadlines sets deadlines for specific workflow stages
func (s *SmartContract) SetStageDeadlines(ctx contractapi.TransactionContextInterface, documentID, stageDeadlinesJSON, invokerId string) error {
	// Parse stage deadlines JSON: {"1": 24, "3": 48}
	var stageDeadlines map[string]int
	if err := json.Unmarshal([]byte(stageDeadlinesJSON), &stageDeadlines); err != nil {
		return fmt.Errorf("invalid stageDeadlines JSON: %v", err)
	}

	if len(stageDeadlines) == 0 {
		return fmt.Errorf("at least one stage deadline must be specified")
	}

	// Load document and validate authorization
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("only document owner or privileged editors can set deadlines")
	}

	// Validate all stage numbers exist in workflow
	for stageNumStr := range stageDeadlines {
		stageNum, err := strconv.Atoi(stageNumStr)
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", stageNumStr)
		}
		if stageNum < 1 || stageNum > len(doc.Workflow.Stages) {
			return fmt.Errorf("stage %d does not exist (valid range: 1-%d)", stageNum, len(doc.Workflow.Stages))
		}
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Initialize deadline maps if needed
	if doc.DeadlineConfig.StageDeadlines == nil {
		doc.DeadlineConfig.StageDeadlines = make(map[string]string)
	}
	if doc.DeadlineStatus.StageStatuses == nil {
		doc.DeadlineStatus.StageStatuses = make(map[string]StageDeadlineStatus)
	}

	// DETERMINISM FIX: Process stages in sorted order to ensure consistent results across peers
	var stageKeys []string
	for stageNumStr := range stageDeadlines {
		stageKeys = append(stageKeys, stageNumStr)
	}
	sort.Strings(stageKeys)

	// Set deadlines for specified stages in deterministic order
	var updatedStages []string
	for _, stageNumStr := range stageKeys {
		deadlineHours := stageDeadlines[stageNumStr]
		if deadlineHours <= 0 {
			return fmt.Errorf("deadline hours must be positive for stage %s", stageNumStr)
		}

		stageKey := fmt.Sprintf("stage_%s", stageNumStr)
		deadlineRFC3339 := calculateDeadlineFromHours(deadlineHours)

		// Update stage deadline
		doc.DeadlineConfig.StageDeadlines[stageKey] = deadlineRFC3339

		// Update stage status
		stageNum, _ := strconv.Atoi(stageNumStr)
		hoursRemaining := int64(deadlineHours)
		status := "ACTIVE"
		if hoursRemaining <= 24 {
			status = "WARNING"
		}

		doc.DeadlineStatus.StageStatuses[stageKey] = StageDeadlineStatus{
			StageNumber:   stageNum,
			Deadline:      deadlineRFC3339,
			Status:        status,
			TimeRemaining: hoursRemaining,
		}

		updatedStages = append(updatedStages, stageNumStr)
	}

	// Enable deadline system if any deadline is set
	doc.DeadlineConfig.Enabled = len(doc.DeadlineConfig.StageDeadlines) > 0

	// Update overall status based on current stage
	currentStageKey := fmt.Sprintf("stage_%d", doc.Workflow.CurrentStage)
	if currentStageStatus, exists := doc.DeadlineStatus.StageStatuses[currentStageKey]; exists {
		doc.DeadlineStatus.CurrentStageStatus = currentStageStatus.Status
		doc.DeadlineStatus.CurrentStageDeadline = currentStageStatus.Deadline
		doc.DeadlineStatus.TimeRemaining = currentStageStatus.TimeRemaining
	} else {
		doc.DeadlineStatus.CurrentStageStatus = "ACTIVE"
		doc.DeadlineStatus.CurrentStageDeadline = ""
		doc.DeadlineStatus.TimeRemaining = 0
	}

	doc.LastModifiedTimestamp = timestamp
	s.ensureDocumentArraysInitialized(doc)

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Emit event
	description := fmt.Sprintf("Stage deadlines updated for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"updatedStages": updatedStages,
		"stageDeadlines": stageDeadlines,
	}

	return s.emitEvent(ctx, "StageDeadlinesUpdated", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// GetDeadlineStatus returns current deadline status for a document
func (s *SmartContract) GetDeadlineStatus(ctx contractapi.TransactionContextInterface, id string) (DeadlineStatus, error) {
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return DeadlineStatus{}, err
	}

	if !doc.DeadlineConfig.Enabled || len(doc.DeadlineConfig.StageDeadlines) == 0 {
		return DeadlineStatus{
			DocumentId:           id,
			CurrentStageStatus:   "NONE",
			CurrentStageDeadline: "",
			TimeRemaining:        0,
			StageStatuses:        make(map[string]StageDeadlineStatus),
			WarningsTriggered:    []DeadlineWarning{},
			BreachesRecorded:     []DeadlineBreach{},
		}, nil
	}

	// Get current stage deadline
	currentStageKey := fmt.Sprintf("stage_%d", doc.Workflow.CurrentStage)
	var currentStageDeadline string
	var currentStageHours int64
	var overallStatus string

	if deadline, exists := doc.DeadlineConfig.StageDeadlines[currentStageKey]; exists && deadline != "" {
		currentStageDeadline = deadline
		deadlineTime, err := time.Parse(time.RFC3339, deadline)
		if err == nil {
			currentStageHours = int64(time.Until(deadlineTime).Hours())
			if currentStageHours <= 0 {
				overallStatus = "OVERDUE"
			} else if currentStageHours <= 24 {
				overallStatus = "WARNING"
			} else {
				overallStatus = "ACTIVE"
			}
		} else {
			overallStatus = "INVALID"
		}
	} else {
		// Current stage has no deadline
		overallStatus = "NONE"
		currentStageHours = 0
	}

	// Build result with current stage info
	result := DeadlineStatus{
		DocumentId:           id,
		CurrentStageStatus:   overallStatus,
		CurrentStageDeadline: currentStageDeadline,
		TimeRemaining:        currentStageHours,
		StageStatuses:        make(map[string]StageDeadlineStatus),
		WarningsTriggered:    []DeadlineWarning{},
		BreachesRecorded:     []DeadlineBreach{},
	}

	// Build comprehensive audit trail - show ALL stage deadlines with their statuses
	for stageKey, deadline := range doc.DeadlineConfig.StageDeadlines {
		if deadline != "" {
			deadlineTime, err := time.Parse(time.RFC3339, deadline)
			var status string
			var hoursRemaining int64

			if err == nil {
				hoursRemaining = int64(time.Until(deadlineTime).Hours())
				if hoursRemaining <= 0 {
					status = "OVERDUE"
				} else if hoursRemaining <= 24 {
					status = "WARNING"
				} else {
					status = "ACTIVE"
				}
			} else {
				status = "INVALID"
				hoursRemaining = 0
			}

			// Extract stage number from stage key (e.g., "stage_1" -> 1)
			var stageNumber int
			fmt.Sscanf(stageKey, "stage_%d", &stageNumber)

			// Mark completed stages appropriately
			if doc.Workflow.Enabled {
				for _, completedStage := range doc.Workflow.CompletedStages {
					if completedStage == stageNumber {
						status = "COMPLETED"
						break
					}
				}
			}

			result.StageStatuses[stageKey] = StageDeadlineStatus{
				StageNumber:   stageNumber,
				Deadline:      deadline,
				Status:        status,
				TimeRemaining: hoursRemaining,
			}
		}
	}

	// Copy arrays from stored status (ensuring they're never nil)
	if doc.DeadlineStatus.WarningsTriggered != nil {
		result.WarningsTriggered = doc.DeadlineStatus.WarningsTriggered
	}
	if doc.DeadlineStatus.BreachesRecorded != nil {
		result.BreachesRecorded = doc.DeadlineStatus.BreachesRecorded
	}

	return result, nil
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

		// Check stage deadlines instead of empty DocumentDeadline
		documentAdded := false
		for _, stageDeadline := range doc.DeadlineConfig.StageDeadlines {
			if stageDeadline == "" {
				continue
			}

			inRange, hoursRemaining, err := isDeadlineInRange(stageDeadline, hoursAheadInt)
			if err != nil {
				continue
			}

			if inRange && !documentAdded {
				status, _, _ := calculateDeadlineStatus(stageDeadline, doc.DeadlineConfig.WarningThreshold)
				doc.DeadlineStatus = createDefaultDeadlineStatus(doc.ID, status, stageDeadline, int64(hoursRemaining))
				doc.DeadlineStatus = s.ensureDeadlineArraysInitialized(doc.DeadlineStatus)
				upcomingDeadlineDocs = append(upcomingDeadlineDocs, doc)
				documentAdded = true // Avoid adding same document multiple times
				break
			}
		}
	}
	return upcomingDeadlineDocs, nil
}

// RemoveStageDeadlines removes deadlines from specific workflow stages
func (s *SmartContract) RemoveStageDeadlines(ctx contractapi.TransactionContextInterface, documentID, stageNumbersJSON, invokerId string) error {
	// Parse stage numbers JSON: ["1", "3"]
	var stageNumbers []string
	if err := json.Unmarshal([]byte(stageNumbersJSON), &stageNumbers); err != nil {
		return fmt.Errorf("invalid stageNumbers JSON: %v", err)
	}

	if len(stageNumbers) == 0 {
		return fmt.Errorf("at least one stage number must be specified")
	}

	// Load document and validate authorization
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("only document owner or privileged editors can remove deadlines")
	}

	// Validate all stage numbers exist in workflow
	for _, stageNumStr := range stageNumbers {
		stageNum, err := strconv.Atoi(stageNumStr)
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", stageNumStr)
		}
		if stageNum < 1 || stageNum > len(doc.Workflow.Stages) {
			return fmt.Errorf("stage %d does not exist (valid range: 1-%d)", stageNum, len(doc.Workflow.Stages))
		}
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Initialize deadline maps if needed
	if doc.DeadlineConfig.StageDeadlines == nil {
		doc.DeadlineConfig.StageDeadlines = make(map[string]string)
	}
	if doc.DeadlineStatus.StageStatuses == nil {
		doc.DeadlineStatus.StageStatuses = make(map[string]StageDeadlineStatus)
	}

	// Remove deadlines from specified stages
	var removedStages []string
	for _, stageNumStr := range stageNumbers {
		stageKey := fmt.Sprintf("stage_%s", stageNumStr)

		// Check if stage actually has a deadline to remove
		if _, exists := doc.DeadlineConfig.StageDeadlines[stageKey]; exists {
			delete(doc.DeadlineConfig.StageDeadlines, stageKey)
			delete(doc.DeadlineStatus.StageStatuses, stageKey)
			removedStages = append(removedStages, stageNumStr)
		}
	}

	if len(removedStages) == 0 {
		return fmt.Errorf("none of the specified stages had deadlines to remove")
	}

	// Disable deadline system if no deadlines remain
	doc.DeadlineConfig.Enabled = len(doc.DeadlineConfig.StageDeadlines) > 0

	// Update overall status based on current stage
	currentStageKey := fmt.Sprintf("stage_%d", doc.Workflow.CurrentStage)
	if currentStageStatus, exists := doc.DeadlineStatus.StageStatuses[currentStageKey]; exists {
		doc.DeadlineStatus.CurrentStageStatus = currentStageStatus.Status
		doc.DeadlineStatus.CurrentStageDeadline = currentStageStatus.Deadline
		doc.DeadlineStatus.TimeRemaining = currentStageStatus.TimeRemaining
	} else {
		// Current stage has no deadline
		doc.DeadlineStatus.CurrentStageStatus = "NONE"
		doc.DeadlineStatus.CurrentStageDeadline = ""
		doc.DeadlineStatus.TimeRemaining = 0
	}

	doc.LastModifiedTimestamp = timestamp
	s.ensureDocumentArraysInitialized(doc)

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Emit event
	description := fmt.Sprintf("Stage deadlines removed from document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"removedStages": removedStages,
	}

	return s.emitEvent(ctx, "StageDeadlinesRemoved", invokerId, description, doc.ID, doc.LatestVersion, details)
}