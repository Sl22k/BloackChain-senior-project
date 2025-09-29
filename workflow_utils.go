package chaincode

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// generateApproverKey creates consistent approver keys for workflow tracking
func (s *SmartContract) generateApproverKey(approver string, stageNumber int, totalStages int) string {
	if totalStages == 1 {
		return approver
	}
	return fmt.Sprintf("#%d %s", stageNumber, approver)
}

// handleStageAdvancement processes stage completion and advancement with proper notifications
func (s *SmartContract) handleStageAdvancement(ctx contractapi.TransactionContextInterface, doc *Document, currentStage *WorkflowStage, approver string) error {
	// Count approved decisions in current stage
	approvedCount := 0
	for _, stageDecision := range currentStage.StageApprovals {
		if stageDecision.Status == Approved {
			approvedCount++
		}
	}

	// Check if stage requirements are met (handle RequiredCount = 0 as "ALL")
	requiredCount := currentStage.RequiredCount
	if requiredCount <= 0 {
		requiredCount = len(currentStage.Approvers) // 0 means all approvers needed
	}

	if approvedCount >= requiredCount {
		// Mark current stage as completed
		stageCompleted := false
		for _, completedStage := range doc.Workflow.CompletedStages {
			if completedStage == doc.Workflow.CurrentStage {
				stageCompleted = true
				break
			}
		}
		if !stageCompleted {
			doc.Workflow.CompletedStages = append(doc.Workflow.CompletedStages, doc.Workflow.CurrentStage)
		}

		// Handle stage advancement based on AutoAdvance flag
		if doc.Workflow.CurrentStage < len(doc.Workflow.Stages) {
			if currentStage.AutoAdvance {
				// Auto-advance to next stage and notify next stage approvers
				doc.Workflow.CurrentStage++

				// Emit notification for next stage approvers
				nextStage := &doc.Workflow.Stages[doc.Workflow.CurrentStage-1]
				details := map[string]interface{}{
					"stage":         doc.Workflow.CurrentStage,
					"stageName":     nextStage.StageName,
					"approvers":     nextStage.Approvers,
					"requiredCount": nextStage.RequiredCount,
					"autoAdvanced":  true,
					"previousStage": doc.Workflow.CurrentStage - 1,
				}

				if err := s.emitEvent(ctx, "STAGE_READY_FOR_APPROVAL", "system",
					fmt.Sprintf("Document %s advanced to stage %d - %s", doc.ID, doc.Workflow.CurrentStage, nextStage.StageName),
					doc.ID, doc.LatestVersion, details); err != nil {
					return err
				}
			} else {
				// Manual advance required - notify uploader/privileged users
				details := map[string]interface{}{
					"stage":               doc.Workflow.CurrentStage,
					"stageName":           currentStage.StageName,
					"nextStage":           doc.Workflow.CurrentStage + 1,
					"nextStageName":       doc.Workflow.Stages[doc.Workflow.CurrentStage].StageName,
					"approvedCount":       approvedCount,
					"requiredCount":       requiredCount,
					"uploader":            doc.Uploader,
					"manualAdvanceNeeded": true,
				}

				if err := s.emitEvent(ctx, "STAGE_COMPLETED_MANUAL_ADVANCE", doc.Uploader,
					fmt.Sprintf("Stage %d completed for document %s - manual advance needed", doc.Workflow.CurrentStage, doc.ID),
					doc.ID, doc.LatestVersion, details); err != nil {
					return err
				}
			}
		} else {
			// Final stage completed - document fully approved
			details := map[string]interface{}{
				"allStagesCompleted": true,
				"totalStages":        len(doc.Workflow.Stages),
				"finalApprover":      approver,
			}

			if err := s.emitEvent(ctx, "DOCUMENT_FULLY_APPROVED", approver,
				fmt.Sprintf("Document %s fully approved - all stages completed", doc.ID),
				doc.ID, doc.LatestVersion, details); err != nil {
				return err
			}
		}
	}

	return nil
}

// resetWorkflowFromStage resets workflow stages from target stage onwards (reusable reset logic)
func (s *SmartContract) resetWorkflowFromStage(doc *Document, targetStage int) {
	// Reset stages from target stage onwards
	for i := targetStage - 1; i < len(doc.Workflow.Stages); i++ {
		// Reset stage approvals to PENDING (don't delete approvers entirely)
		if doc.Workflow.Stages[i].StageApprovals == nil {
			doc.Workflow.Stages[i].StageApprovals = make(map[string]Decision)
		}

		// Reset existing approvals to PENDING instead of deleting them
		for approver := range doc.Workflow.Stages[i].StageApprovals {
			doc.Workflow.Stages[i].StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: time.Now().Format(time.RFC3339),
			}
		}

		// Add any missing approvers from the stage configuration as PENDING
		for _, approver := range doc.Workflow.Stages[i].Approvers {
			if _, exists := doc.Workflow.Stages[i].StageApprovals[approver]; !exists {
				doc.Workflow.Stages[i].StageApprovals[approver] = Decision{
					Status:    Pending,
					Comment:   "",
					Timestamp: time.Now().Format(time.RFC3339),
				}
			}
		}
	}

	// Set current stage
	doc.Workflow.CurrentStage = targetStage

	// Remove completed stages from target stage onwards
	newCompletedStages := []int{}
	for _, stage := range doc.Workflow.CompletedStages {
		if stage < targetStage {
			newCompletedStages = append(newCompletedStages, stage)
		}
	}
	doc.Workflow.CompletedStages = newCompletedStages
}

// ensureWorkflowArraysInitialized ensures all workflow arrays are properly initialized
func (s *SmartContract) ensureWorkflowArraysInitialized(workflow *WorkflowConfig) {
	if workflow.Stages == nil {
		workflow.Stages = []WorkflowStage{}
	}
	if workflow.CompletedStages == nil {
		workflow.CompletedStages = []int{}
	}
	if workflow.StageHistory == nil {
		workflow.StageHistory = []StageTransition{}
	}

	// Initialize stage approvals for each stage
	for i := range workflow.Stages {
		if workflow.Stages[i].StageApprovals == nil {
			workflow.Stages[i].StageApprovals = make(map[string]Decision)
		}
	}
}

// ensureDocumentArraysInitialized ensures all document arrays are properly initialized
func (s *SmartContract) ensureDocumentArraysInitialized(doc *Document) {
	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}
	if doc.Versions == nil {
		doc.Versions = []DocumentVersion{}
	}
	// FIX: Initialize editor arrays to prevent null values in Fabric
	if doc.Editors == nil {
		doc.Editors = []string{}
	}
	if doc.PrivilegedEditors == nil {
		doc.PrivilegedEditors = []string{}
	}

	// Initialize workflow arrays
	s.ensureWorkflowArraysInitialized(&doc.Workflow)

	// Initialize deadline arrays to prevent null values
	if doc.DeadlineConfig.StageDeadlines == nil {
		doc.DeadlineConfig.StageDeadlines = make(map[string]string)
	}
	if doc.DeadlineConfig.EscalationTargets == nil {
		doc.DeadlineConfig.EscalationTargets = []string{}
	}
	if doc.DeadlineConfig.AutomationHooks == nil {
		doc.DeadlineConfig.AutomationHooks = []AutomationHook{}
	}
	if doc.DeadlineStatus.StageStatuses == nil {
		doc.DeadlineStatus.StageStatuses = make(map[string]StageDeadlineStatus)
	}
	if doc.DeadlineStatus.WarningsTriggered == nil {
		doc.DeadlineStatus.WarningsTriggered = []DeadlineWarning{}
	}
	if doc.DeadlineStatus.BreachesRecorded == nil {
		doc.DeadlineStatus.BreachesRecorded = []DeadlineBreach{}
	}
}

// ============================================================================
// ENHANCED ROLLBACK UTILITIES
// ============================================================================

// validateRollbackDeadlines checks for deadline conflicts when rolling back to a target stage
func (s *SmartContract) validateRollbackDeadlines(doc *Document, targetStage int) ([]DeadlineConflict, error) {
	// Initialize conflicts slice (never nil for Fabric compatibility)
	conflicts := []DeadlineConflict{}

	if !doc.DeadlineConfig.Enabled || doc.DeadlineConfig.StageDeadlines == nil {
		return conflicts, nil // No deadlines = no conflicts
	}

	// Check all stages from target onwards for deadline conflicts
	for i := targetStage; i <= len(doc.Workflow.Stages); i++ {
		stageKey := fmt.Sprintf("stage_%d", i)
		deadline, exists := doc.DeadlineConfig.StageDeadlines[stageKey]

		if !exists || deadline == "" {
			continue // No deadline for this stage
		}

		deadlineTime, err := time.Parse(time.RFC3339, deadline)
		if err != nil {
			continue // Invalid deadline format, skip
		}

		hoursOverdue := int(-time.Until(deadlineTime).Hours())
		if hoursOverdue > 0 {
			// Deadline has passed - this is a conflict
			status := "OVERDUE"
			if hoursOverdue <= 24 {
				status = "WARNING"
			}

			conflicts = append(conflicts, DeadlineConflict{
				StageNumber:  i,
				Deadline:     deadline,
				HoursOverdue: hoursOverdue,
				Status:       status,
			})
		}
	}

	return conflicts, nil
}

// resolveDeadlineConflicts applies user's chosen resolution to deadline conflicts
func (s *SmartContract) resolveDeadlineConflicts(doc *Document, options DeadlineRollbackOptions, conflicts []DeadlineConflict) (map[string]string, error) {
	// Initialize changes map (never nil for Fabric compatibility)
	changes := make(map[string]string)

	if len(conflicts) == 0 {
		return changes, nil // No conflicts to resolve
	}

	timestamp := time.Now()

	if options.RemoveConflicts {
		// Remove all conflicting deadlines
		for _, conflict := range conflicts {
			stageKey := fmt.Sprintf("stage_%d", conflict.StageNumber)
			if doc.DeadlineConfig.StageDeadlines != nil {
				delete(doc.DeadlineConfig.StageDeadlines, stageKey)
			}
			if doc.DeadlineStatus.StageStatuses != nil {
				delete(doc.DeadlineStatus.StageStatuses, stageKey)
			}
			changes[stageKey] = "REMOVED"
		}
	} else if len(options.NewDeadlines) > 0 {
		// Apply new deadlines for specified stages
		for stageNumStr, hours := range options.NewDeadlines {
			if hours <= 0 {
				return nil, fmt.Errorf("deadline hours must be positive for stage %s", stageNumStr)
			}

			stageKey := fmt.Sprintf("stage_%s", stageNumStr)
			newDeadline := timestamp.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)

			// Initialize maps if nil (Fabric null-safety)
			if doc.DeadlineConfig.StageDeadlines == nil {
				doc.DeadlineConfig.StageDeadlines = make(map[string]string)
			}
			if doc.DeadlineStatus.StageStatuses == nil {
				doc.DeadlineStatus.StageStatuses = make(map[string]StageDeadlineStatus)
			}

			doc.DeadlineConfig.StageDeadlines[stageKey] = newDeadline

			stageNum, _ := strconv.Atoi(stageNumStr)
			doc.DeadlineStatus.StageStatuses[stageKey] = StageDeadlineStatus{
				StageNumber:   stageNum,
				Deadline:      newDeadline,
				Status:        "ACTIVE",
				TimeRemaining: int64(hours),
			}

			changes[stageKey] = newDeadline
		}
	} else {
		return nil, fmt.Errorf("no valid conflict resolution provided")
	}

	// Update deadline system enabled status
	doc.DeadlineConfig.Enabled = len(doc.DeadlineConfig.StageDeadlines) > 0

	return changes, nil
}

// performEnhancedRollback executes the rollback with comprehensive tracking
func (s *SmartContract) performEnhancedRollback(doc *Document, targetStage int, comment, invokerId string) ReturnToStageResponse {
	timestamp := time.Now().Format(time.RFC3339)

	// Initialize response with null-safe arrays
	response := ReturnToStageResponse{
		Success:              true,
		ReturnedToStage:      targetStage,
		ResetStages:          []int{},
		RemovedFromCompleted: []int{},
		ApprovalsCleared:     0,
		NewCurrentStage:      targetStage,
		DeadlineConflicts:    []DeadlineConflict{},
		DeadlineChanges:      make(map[string]string),
	}

	previousStage := doc.Workflow.CurrentStage

	// Track which stages will be reset
	for i := targetStage; i <= len(doc.Workflow.Stages); i++ {
		response.ResetStages = append(response.ResetStages, i)
	}

	// Track which stages will be removed from completed
	newCompletedStages := []int{}
	for _, stage := range doc.Workflow.CompletedStages {
		if stage < targetStage {
			newCompletedStages = append(newCompletedStages, stage)
		} else {
			response.RemovedFromCompleted = append(response.RemovedFromCompleted, stage)
		}
	}

	// Count approvals that will be cleared
	for i := targetStage - 1; i < len(doc.Workflow.Stages); i++ {
		if doc.Workflow.Stages[i].StageApprovals != nil {
			response.ApprovalsCleared += len(doc.Workflow.Stages[i].StageApprovals)
		}
	}

	// Perform the actual rollback using existing function
	s.resetWorkflowFromStage(doc, targetStage)

	// CRITICAL: Sync ApprovalsMap with reset StageApprovals (Fabric null-safety)
	s.syncApprovalsMapFromStages(doc)

	// Add StageHistory entry for audit trail
	historyEntry := StageTransition{
		FromStage:  previousStage,
		ToStage:    targetStage,
		Transition: "returned",
		Actor:      invokerId,
		Timestamp:  timestamp,
		Comment:    comment,
	}

	// Initialize StageHistory if nil (Fabric null-safety)
	if doc.Workflow.StageHistory == nil {
		doc.Workflow.StageHistory = []StageTransition{}
	}
	doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, historyEntry)
	response.StageHistoryAdded = historyEntry

	return response
}

// syncApprovalsMapFromStages rebuilds ApprovalsMap from all StageApprovals with composite keys (CRITICAL FIX)
func (s *SmartContract) syncApprovalsMapFromStages(doc *Document) {
	// Initialize ApprovalsMap if nil (Fabric null-safety)
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}

	// Clear existing ApprovalsMap
	for key := range doc.ApprovalsMap {
		delete(doc.ApprovalsMap, key)
	}

	// Rebuild from all stages with proper composite keys
	for _, stage := range doc.Workflow.Stages {
		if stage.StageApprovals != nil {
			for approver, decision := range stage.StageApprovals {
				// Use composite key format: #stageNumber approver
				compositeKey := s.generateApproverKey(approver, stage.StageNumber, len(doc.Workflow.Stages))
				doc.ApprovalsMap[compositeKey] = decision
			}
		}
	}
}

// initializeStageApprovals initializes stage approvals with pending status
func (s *SmartContract) initializeStageApprovals(stages []WorkflowStage, timestamp string) []WorkflowStage {
	initializedStages := make([]WorkflowStage, len(stages))

	for i, stage := range stages {
		initializedStages[i] = WorkflowStage{
			StageNumber:    stage.StageNumber,
			StageName:      stage.StageName,
			Approvers:      stage.Approvers,
			RequiredCount:  stage.RequiredCount,
			AutoAdvance:    stage.AutoAdvance,
			StageApprovals: make(map[string]Decision),
		}

		// Initialize each approver with pending status
		for _, approver := range stage.Approvers {
			initializedStages[i].StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return initializedStages
}

// createInitialApprovalsMap creates the document-level approvals map for workflow tracking
func (s *SmartContract) createInitialApprovalsMap(stages []WorkflowStage, timestamp string) map[string]Decision {
	approvalsMap := make(map[string]Decision)

	for _, stage := range stages {
		for _, approver := range stage.Approvers {
			var key string
			if len(stages) == 1 {
				key = approver
			} else {
				key = fmt.Sprintf("#%d %s", stage.StageNumber, approver)
			}

			approvalsMap[key] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return approvalsMap
}

// extractAllApprovers extracts a unique list of all approvers across all stages
func (s *SmartContract) extractAllApprovers(stages []WorkflowStage) []string {
	approversMap := make(map[string]bool)

	for _, stage := range stages {
		for _, approver := range stage.Approvers {
			approversMap[approver] = true
		}
	}

	// DETERMINISM FIX: Sort keys to ensure consistent iteration order across all peers
	var approvers []string
	for approver := range approversMap {
		approvers = append(approvers, approver)
	}
	sort.Strings(approvers)
	return approvers
}

// isDocumentFullyApproved checks if document is in fully approved state
func (s *SmartContract) isDocumentFullyApproved(doc Document) bool {
	if !doc.Workflow.Enabled {
		return false
	}

	// Check if all stages are completed
	totalStages := len(doc.Workflow.Stages)
	return len(doc.Workflow.CompletedStages) == totalStages
}

// ============================================================================
// ENHANCED WORKFLOW CONFIGURATION UTILITIES
// ============================================================================

// detectWorkflowChanges analyzes configuration request and determines what changes will be made
func (s *SmartContract) detectWorkflowChanges(currentWorkflow *WorkflowConfig, request WorkflowConfigurationRequest) (bool, bool, error) {
	// Initialize change detection flags
	approvalAffectingChanges := false
	structuralChanges := false

	// Check for stage removals
	if len(request.StagesToRemove) > 0 {
		approvalAffectingChanges = true
		structuralChanges = true
	}

	// Check for new stages being added
	if len(request.NewStages) > 0 {
		structuralChanges = true
	}

	// Check for stage modifications
	for stageNumStr, operation := range request.StageOperations {
		if operation.Action == "update" && operation.UpdatedStage != nil {
			stageNum, err := strconv.Atoi(stageNumStr)
			if err != nil {
				return false, false, fmt.Errorf("invalid stage number: %s", stageNumStr)
			}

			if stageNum < 1 || stageNum > len(currentWorkflow.Stages) {
				return false, false, fmt.Errorf("stage number %d out of range", stageNum)
			}

			currentStage := &currentWorkflow.Stages[stageNum-1]
			newStage := operation.UpdatedStage

			// Check if approvers changed
			if !s.stringSlicesEqual(currentStage.Approvers, newStage.Approvers) {
				approvalAffectingChanges = true
			}

			// Check if required count changed
			if currentStage.RequiredCount != newStage.RequiredCount {
				approvalAffectingChanges = true
			}

			// Any modification is considered structural
			structuralChanges = true
		}
	}

	return approvalAffectingChanges, structuralChanges, nil
}

// stringSlicesEqual compares two string slices for equality (order matters)
func (s *SmartContract) stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// applyStageOperations applies the stage operations to create the new workflow configuration
// DETERMINISM FIX: Process operations in deterministic order
//
// STAGE RENUMBERING LOGIC VALIDATION:
// Example 1: Remove stage 2 from [1, 2, 3]
//   Input:  Stages [1:"Legal", 2:"Dev", 3:"Management"]
//   Remove: [2]
//   Result: [1:"Legal", 2:"Management"] (stage 3 becomes stage 2)
//
// Example 2: Remove stages 1,3 from [1, 2, 3, 4]
//   Input:  Stages [1:"Legal", 2:"Dev", 3:"QA", 4:"Management"]
//   Remove: [1, 3]
//   Result: [1:"Dev", 2:"Management"] (stages 2,4 become 1,2)
//
// Example 3: Add 2 stages to [1, 2]
//   Input:  Stages [1:"Legal", 2:"Dev"]
//   Add:    [NewStage1, NewStage2]
//   Result: [1:"Legal", 2:"Dev", 3:"NewStage1", 4:"NewStage2"]
func (s *SmartContract) applyStageOperations(currentWorkflow *WorkflowConfig, request WorkflowConfigurationRequest, timestamp string) ([]WorkflowStage, []int, []int, []int, error) {
	// Initialize tracking arrays (Fabric null-safety)
	stagesAdded := []int{}
	stagesRemoved := []int{}
	stagesModified := []int{}

	// DETERMINISM FIX: Sort all arrays to ensure consistent processing order
	sort.Ints(request.StagesToRemove)

	// Create sorted list of stage operations for deterministic processing
	var sortedOperations []struct {
		stageNum  int
		operation StageOperation
	}
	for stageNumStr, operation := range request.StageOperations {
		stageNum, err := strconv.Atoi(stageNumStr)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("invalid stage number in operations: %s", stageNumStr)
		}
		sortedOperations = append(sortedOperations, struct {
			stageNum  int
			operation StageOperation
		}{stageNum, operation})
	}
	// DETERMINISM FIX: Sort operations by stage number
	sort.Slice(sortedOperations, func(i, j int) bool {
		return sortedOperations[i].stageNum < sortedOperations[j].stageNum
	})

	// Start with current stages as base
	newStages := make([]WorkflowStage, 0, len(currentWorkflow.Stages)+len(request.NewStages))

	// DETERMINISM FIX: Process stages in sequential order (1, 2, 3, ...)
	stageSkipList := make(map[int]bool)
	for _, removeStageNum := range request.StagesToRemove {
		stageSkipList[removeStageNum] = true
		stagesRemoved = append(stagesRemoved, removeStageNum)
	}

	// Create lookup map for operations (for O(1) access)
	operationsMap := make(map[int]StageOperation)
	for _, sortedOp := range sortedOperations {
		operationsMap[sortedOp.stageNum] = sortedOp.operation
	}

	// Process existing stages in deterministic order (1, 2, 3, ...)
	for i, currentStage := range currentWorkflow.Stages {
		stageNum := i + 1 // 1-based stage numbers

		// Skip removed stages
		if stageSkipList[stageNum] {
			continue
		}

		// Check for operations on this stage
		operation, hasOperation := operationsMap[stageNum]

		var finalStage WorkflowStage

		if hasOperation && operation.Action == "update" && operation.UpdatedStage != nil {
			// Apply updates to this stage
			finalStage = *operation.UpdatedStage
			finalStage.StageNumber = stageNum // Preserve original stage number for now

			// FIXED: Preserve existing approvals if user requested, otherwise reset
			if request.KeepExistingApprovals {
				finalStage.StageApprovals = currentStage.StageApprovals
			} else {
				finalStage.StageApprovals = make(map[string]Decision)
				// Initialize all approvers with PENDING status
				for _, approver := range finalStage.Approvers {
					finalStage.StageApprovals[approver] = Decision{
						Status:    Pending,
						Comment:   "",
						Timestamp: timestamp,
					}
				}
			}

			// DETERMINISM FIX: Sort approvers to ensure consistent order
			sort.Strings(finalStage.Approvers)

			stagesModified = append(stagesModified, stageNum)
		} else {
			// Keep existing stage unchanged
			finalStage = currentStage

			// DETERMINISM FIX: Ensure approvers are sorted even for unchanged stages
			sort.Strings(finalStage.Approvers)

			// FIXED: Only reset approvals if user explicitly chose NOT to keep them
			if !request.KeepExistingApprovals {
				finalStage.StageApprovals = make(map[string]Decision)
				// Initialize all approvers with PENDING status
				for _, approver := range finalStage.Approvers {
					finalStage.StageApprovals[approver] = Decision{
						Status:    Pending,
						Comment:   "",
						Timestamp: timestamp,
					}
				}
			}
			// If KeepExistingApprovals is true, we keep the existing approvals (no action needed)
		}

		newStages = append(newStages, finalStage)
	}

	// Add new stages at the end in deterministic order
	nextStageNumber := len(newStages) + 1
	for _, newStage := range request.NewStages {
		newStage.StageNumber = nextStageNumber
		newStage.StageApprovals = make(map[string]Decision) // New stages always start with empty approvals

		// DETERMINISM FIX: Sort approvers for new stages
		sort.Strings(newStage.Approvers)

		// FIXED: Initialize StageApprovals for all approvers in new stages
		for _, approver := range newStage.Approvers {
			newStage.StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}

		newStages = append(newStages, newStage)
		stagesAdded = append(stagesAdded, nextStageNumber)
		nextStageNumber++
	}

	// Renumber all stages to be sequential (1, 2, 3, ...) and track final mappings
	finalStageMapping := make(map[int]int) // original -> final stage number
	for i := range newStages {
		oldStageNum := newStages[i].StageNumber
		newStageNum := i + 1
		newStages[i].StageNumber = newStageNum
		finalStageMapping[oldStageNum] = newStageNum
	}

	// DETERMINISM FIX: Recalculate tracking arrays based on final sequential positions
	stagesAdded = []int{}
	finalStagesModified := []int{}

	// Mark added stages (those beyond original stage count minus removed stages)
	originalCount := len(currentWorkflow.Stages)
	finalCount := len(newStages)
	removedCount := len(request.StagesToRemove)
	expectedCountAfterRemovals := originalCount - removedCount

	if finalCount > expectedCountAfterRemovals {
		for i := expectedCountAfterRemovals; i < finalCount; i++ {
			stagesAdded = append(stagesAdded, i+1) // 1-based final stage numbers
		}
	}

	// Update modified stages array to reflect final stage numbers
	for _, origStageNum := range stagesModified {
		if finalStageNum, exists := finalStageMapping[origStageNum]; exists {
			finalStagesModified = append(finalStagesModified, finalStageNum)
		}
	}
	stagesModified = finalStagesModified

	// DETERMINISM FIX: Sort all tracking arrays
	sort.Ints(stagesAdded)
	sort.Ints(stagesRemoved)
	sort.Ints(stagesModified)

	// FIXED: Don't reinitialize approvals - they're already properly set above based on KeepExistingApprovals
	// The logic above already handles approval preservation vs reset correctly

	return newStages, stagesAdded, stagesRemoved, stagesModified, nil
}

// applyDeadlineUpdates applies deadline configuration updates to the document
// DETERMINISM FIX: Process deadline updates in sorted order
func (s *SmartContract) applyDeadlineUpdates(doc *Document, request WorkflowConfigurationRequest) (map[string]string, error) {
	deadlineChanges := make(map[string]string) // Never nil for Fabric compatibility

	if len(request.DeadlineUpdates) == 0 {
		return deadlineChanges, nil
	}

	timestamp := time.Now()

	// Initialize deadline maps if nil (Fabric null-safety)
	if doc.DeadlineConfig.StageDeadlines == nil {
		doc.DeadlineConfig.StageDeadlines = make(map[string]string)
	}
	if doc.DeadlineStatus.StageStatuses == nil {
		doc.DeadlineStatus.StageStatuses = make(map[string]StageDeadlineStatus)
	}

	// DETERMINISM FIX: Sort deadline updates by stage number for consistent processing
	var sortedDeadlineUpdates []struct {
		stageNumStr string
		hours       int
	}
	for stageNumStr, hours := range request.DeadlineUpdates {
		sortedDeadlineUpdates = append(sortedDeadlineUpdates, struct {
			stageNumStr string
			hours       int
		}{stageNumStr, hours})
	}
	sort.Slice(sortedDeadlineUpdates, func(i, j int) bool {
		// Sort by stage number (convert to int for proper numerical sorting)
		stageI, _ := strconv.Atoi(sortedDeadlineUpdates[i].stageNumStr)
		stageJ, _ := strconv.Atoi(sortedDeadlineUpdates[j].stageNumStr)
		return stageI < stageJ
	})

	// Apply deadline updates in deterministic order
	for _, update := range sortedDeadlineUpdates {
		if update.hours <= 0 {
			continue // Skip invalid deadline values
		}

		stageKey := fmt.Sprintf("stage_%s", update.stageNumStr)
		newDeadline := timestamp.Add(time.Duration(update.hours) * time.Hour).Format(time.RFC3339)

		doc.DeadlineConfig.StageDeadlines[stageKey] = newDeadline

		stageNum, _ := strconv.Atoi(update.stageNumStr)
		doc.DeadlineStatus.StageStatuses[stageKey] = StageDeadlineStatus{
			StageNumber:   stageNum,
			Deadline:      newDeadline,
			Status:        "ACTIVE",
			TimeRemaining: int64(update.hours),
		}

		deadlineChanges[stageKey] = newDeadline
	}

	// Enable deadline system if any deadlines were set
	if len(doc.DeadlineConfig.StageDeadlines) > 0 {
		doc.DeadlineConfig.Enabled = true
	}

	return deadlineChanges, nil
}

// performWorkflowReconfiguration executes the complete workflow reconfiguration with comprehensive tracking
func (s *SmartContract) performWorkflowReconfiguration(doc *Document, request WorkflowConfigurationRequest, invokerId string) (*WorkflowConfigurationResponse, error) {
	timestamp := time.Now().Format(time.RFC3339)

	// Initialize response with null-safe arrays
	response := &WorkflowConfigurationResponse{
		Success:               true,
		PreviousStageCount:    len(doc.Workflow.Stages),
		StagesAdded:          []int{},
		StagesRemoved:        []int{},
		StagesModified:       []int{},
		ApprovalsReset:       false,
		CurrentStageReset:    false,
		CompletedStagesCleared: false,
		DeadlineChanges:      make(map[string]string),
	}

	// Detect what changes will be made
	approvalAffectingChanges, structuralChanges, err := s.detectWorkflowChanges(&doc.Workflow, request)
	if err != nil {
		return nil, fmt.Errorf("failed to detect workflow changes: %v", err)
	}

	_ = structuralChanges // Used in conditional logic below

	// Apply stage operations to create new workflow
	newStages, stagesAdded, stagesRemoved, stagesModified, err := s.applyStageOperations(&doc.Workflow, request, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to apply stage operations: %v", err)
	}

	// Update response tracking
	response.NewStageCount = len(newStages)
	response.StagesAdded = stagesAdded
	response.StagesRemoved = stagesRemoved
	response.StagesModified = stagesModified

	// Determine if approvals should be reset
	shouldResetApprovals := !request.KeepExistingApprovals || approvalAffectingChanges

	// Apply deadline updates
	deadlineChanges, err := s.applyDeadlineUpdates(doc, request)
	if err != nil {
		return nil, fmt.Errorf("failed to apply deadline updates: %v", err)
	}
	response.DeadlineChanges = deadlineChanges

	// Update workflow configuration
	previousCurrentStage := doc.Workflow.CurrentStage
	previousCompletedStages := make([]int, len(doc.Workflow.CompletedStages))
	copy(previousCompletedStages, doc.Workflow.CompletedStages)

	doc.Workflow.Stages = newStages

	// FIXED: Only reset current stage if removing stages or if current stage is beyond new workflow
	maxStage := len(newStages)
	if len(request.StagesToRemove) > 0 || doc.Workflow.CurrentStage > maxStage {
		// Current stage is beyond new workflow or stages were removed - reset to stage 1
		doc.Workflow.CurrentStage = 1
		doc.Workflow.CompletedStages = []int{}
		response.CurrentStageReset = true
		response.CompletedStagesCleared = true
	} else {
		// Workflow structure preserved current stage position - keep it
		// Only clear completed stages beyond current max if necessary
		newCompletedStages := []int{}
		for _, stage := range doc.Workflow.CompletedStages {
			if stage <= maxStage {
				newCompletedStages = append(newCompletedStages, stage)
			}
		}
		if len(newCompletedStages) != len(doc.Workflow.CompletedStages) {
			doc.Workflow.CompletedStages = newCompletedStages
			response.CompletedStagesCleared = true
		}
	}

	response.ApprovalsReset = shouldResetApprovals

	// CRITICAL: Rebuild ApprovalsMap from stages (Fabric null-safety)
	s.syncApprovalsMapFromStages(doc)

	// CRITICAL FIX: Update VersionWorkflows snapshot to maintain consistency
	// This ensures queries return consistent data across all version tracking systems
	versionKey := fmt.Sprintf("%d", doc.LatestVersion)
	if doc.VersionWorkflows != nil && doc.VersionWorkflows[versionKey] != nil {
		// Sync the stored workflow with current state
		updatedWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)
		doc.VersionWorkflows[versionKey] = &updatedWorkflowState

		// Also update the WorkflowSnapshot in the latest version
		if len(doc.Versions) > 0 {
			latestVersionIndex := len(doc.Versions) - 1
			if doc.Versions[latestVersionIndex].Version == doc.LatestVersion {
				doc.Versions[latestVersionIndex].WorkflowSnapshot = s.createWorkflowSnapshot(doc.Workflow, timestamp)
			}
		}
	}

	// Add comprehensive audit trail to StageHistory
	historyEntry := StageTransition{
		FromStage:  previousCurrentStage,
		ToStage:    doc.Workflow.CurrentStage,
		Transition: "workflow_reconfigured",
		Actor:      invokerId,
		Timestamp:  timestamp,
		Comment:    fmt.Sprintf("Workflow reconfigured: %s. Stages: %d->%d, Added: %v, Removed: %v, Modified: %v", request.ConfigurationReason, response.PreviousStageCount, response.NewStageCount, stagesAdded, stagesRemoved, stagesModified),
	}

	// Initialize StageHistory if nil (Fabric null-safety)
	if doc.Workflow.StageHistory == nil {
		doc.Workflow.StageHistory = []StageTransition{}
	}
	doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, historyEntry)
	response.StageHistoryAdded = historyEntry

	return response, nil
}
