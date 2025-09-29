package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ConfigureDocumentWorkflow provides comprehensive workflow reconfiguration capabilities
func (s *SmartContract) ConfigureDocumentWorkflow(ctx contractapi.TransactionContextInterface, id, invokerId, configurationJSON string) (*WorkflowConfigurationResponse, error) {
	if id == "" || invokerId == "" || configurationJSON == "" {
		return nil, fmt.Errorf("required parameters cannot be empty")
	}

	// Parse configuration request JSON
	var request WorkflowConfigurationRequest
	if err := json.Unmarshal([]byte(configurationJSON), &request); err != nil {
		return nil, fmt.Errorf("invalid configuration JSON: %v", err)
	}

	// Validate configuration request
	if request.ConfigurationReason == "" {
		return nil, fmt.Errorf("configuration reason is required")
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Check authorization (only uploader or privileged users can configure workflow)
	if doc.Uploader != invokerId && !s.hasPrivilegedAccess(invokerId, &doc) {
		return nil, fmt.Errorf("user %s is not authorized to configure workflow for document %s", invokerId, id)
	}

	// Validate workflow is enabled (don't allow configuration of disabled workflows)
	if !doc.Workflow.Enabled {
		return nil, fmt.Errorf("workflow is not enabled for document %s - use SubmitDocument to create initial workflow", id)
	}

	// Validate stage operations
	if err := s.validateConfigurationRequest(&doc.Workflow, request); err != nil {
		return nil, fmt.Errorf("invalid configuration request: %v", err)
	}

	// CRITICAL FIX: Preserve current workflow state BEFORE reconfiguration
	// Store current workflow state with actual approvals for version history
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}
	preservedWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)

	// Perform comprehensive workflow reconfiguration
	response, err := s.performWorkflowReconfiguration(&doc, request, invokerId)
	if err != nil {
		return nil, fmt.Errorf("failed to reconfigure workflow: %v", err)
	}

	// FIXED: Increment version for workflow configuration changes
	if response.NewStageCount != response.PreviousStageCount || len(response.StagesModified) > 0 || len(response.DeadlineChanges) > 0 {
		// CRITICAL FIX: Store current workflow state BEFORE calling performWorkflowReconfiguration
		// This was moved here from after the function call to preserve actual approval states
		if doc.VersionWorkflows == nil {
			doc.VersionWorkflows = make(map[string]*WorkflowConfig)
		}

		// Store the preserved workflow state (with actual approvals) for current version
		doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &preservedWorkflowState

		// Get previous version hash
		previousHash := "INITIAL"
		if len(doc.Versions) > 0 {
			previousHash = doc.Versions[len(doc.Versions)-1].Hash
		}

		// Determine change type and approvers changed flag
		approversChanged := len(response.StagesAdded) > 0 || len(response.StagesRemoved) > 0
		// Note: For stage modifications, we need to check if actual approver lists changed
		// vs just stage info (name/description/RequiredCount/AutoAdvance)
		// Currently treating all stage modifications as approver changes for safety
		// TODO: Implement detailed change detection to distinguish stage_info vs approvers
		for _, stageNum := range response.StagesModified {
			approversChanged = true // For now, assume any stage modification includes approver changes
			_ = stageNum // Avoid unused variable
			break
		}

		// Determine ChangeType based on what actually changed:
		// - "approvers": When approver lists change, stages added/removed
		// - "stage_info": When only stage name/description/RequiredCount/AutoAdvance/deadlines changed
		changeType := "stage_info"
		if approversChanged {
			changeType = "approvers"
		}

		// FIXED: If only deadline changes occurred (no approver/stage structure changes), it's stage_info
		if !approversChanged && len(response.StagesModified) == 0 && len(response.DeadlineChanges) > 0 {
			changeType = "stage_info"
			approversChanged = false // Deadline changes don't affect approvers
		}

		// Increment version
		doc.LatestVersion++

		// Create version entry for workflow reconfiguration
		newVersion := DocumentVersion{
			Version:          doc.LatestVersion,
			Hash:            previousHash,
			Submitter:       invokerId,
			Timestamp:       timestamp,
			ValidDecisions:   doc.ValidDecisions,
			ApproversChanged: approversChanged,
			DecisionsChanged: false, // Workflow reconfiguration doesn't change valid decisions
			ContentChanged:  false,
			ChangeType:      changeType,
			WorkflowSnapshot: s.createWorkflowSnapshot(doc.Workflow, timestamp),
		}

		// Copy current ApprovalsMap to version
		newVersion.ApprovalsMap = make(map[string]Decision)
		for k, v := range doc.ApprovalsMap {
			newVersion.ApprovalsMap[k] = v
		}

		// Add to version history
		doc.Versions = append(doc.Versions, newVersion)

		// Store new version's workflow state
		newVersionWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)
		doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &newVersionWorkflowState
	}

	// Update document timestamp
	doc.LastModifiedTimestamp = timestamp

	// Ensure all arrays are initialized (Fabric null-safety)
	s.ensureDocumentArraysInitialized(&doc)

	// Save updated document
	updatedData, err := json.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %v", err)
	}

	err = ctx.GetStub().PutState(id, updatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %v", err)
	}

	// Emit comprehensive event
	description := fmt.Sprintf("Workflow reconfigured for document %s: %s", id, request.ConfigurationReason)
	details := map[string]interface{}{
		"previousStageCount":    response.PreviousStageCount,
		"newStageCount":         response.NewStageCount,
		"stagesAdded":          response.StagesAdded,
		"stagesRemoved":        response.StagesRemoved,
		"stagesModified":       response.StagesModified,
		"approvalsReset":       response.ApprovalsReset,
		"currentStageReset":    response.CurrentStageReset,
		"completedStagesCleared": response.CompletedStagesCleared,
		"deadlineChanges":      len(response.DeadlineChanges),
		"configurationReason":  request.ConfigurationReason,
		"configuratedBy":       invokerId,
		"timestamp":            timestamp,
	}

	// Emit event (but don't fail configuration if event fails)
	s.emitEvent(ctx, "WorkflowReconfigured", invokerId, description, doc.ID, doc.LatestVersion, details)

	return response, nil
}

// validateConfigurationRequest validates the workflow configuration request
func (s *SmartContract) validateConfigurationRequest(currentWorkflow *WorkflowConfig, request WorkflowConfigurationRequest) error {
	// Validate stage operations
	for stageNumStr, operation := range request.StageOperations {
		stageNum, err := strconv.Atoi(stageNumStr)
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", stageNumStr)
		}

		if stageNum < 1 || stageNum > len(currentWorkflow.Stages) {
			return fmt.Errorf("stage number %d out of range (valid: 1-%d)", stageNum, len(currentWorkflow.Stages))
		}

		if operation.Action == "update" {
			if operation.UpdatedStage == nil {
				return fmt.Errorf("updated stage configuration required for update operation on stage %d", stageNum)
			}

			// Validate updated stage
			if len(operation.UpdatedStage.Approvers) == 0 {
				return fmt.Errorf("stage %d must have at least one approver", stageNum)
			}

			if operation.UpdatedStage.StageName == "" {
				return fmt.Errorf("stage name is required for stage %d", stageNum)
			}
		}
	}

	// Validate stages to remove
	for _, stageNum := range request.StagesToRemove {
		if stageNum < 1 || stageNum > len(currentWorkflow.Stages) {
			return fmt.Errorf("cannot remove stage %d - out of range (valid: 1-%d)", stageNum, len(currentWorkflow.Stages))
		}
	}

	// Validate new stages
	for i, newStage := range request.NewStages {
		if len(newStage.Approvers) == 0 {
			return fmt.Errorf("new stage %d must have at least one approver", i+1)
		}

		if newStage.StageName == "" {
			return fmt.Errorf("stage name is required for new stage %d", i+1)
		}
	}

	// Ensure at least one stage will remain after operations
	finalStageCount := len(currentWorkflow.Stages) - len(request.StagesToRemove) + len(request.NewStages)
	if finalStageCount < 1 {
		return fmt.Errorf("configuration would result in zero stages - at least one stage is required")
	}

	// Validate deadline updates
	for stageNumStr, hours := range request.DeadlineUpdates {
		if hours <= 0 {
			return fmt.Errorf("deadline hours must be positive for stage %s", stageNumStr)
		}

		stageNum, err := strconv.Atoi(stageNumStr)
		if err != nil {
			return fmt.Errorf("invalid stage number for deadline: %s", stageNumStr)
		}

		// Check if this stage will exist after reconfiguration
		// This is a simplified check - more complex validation would require simulating the operations
		if stageNum < 1 || stageNum > finalStageCount {
			return fmt.Errorf("cannot set deadline for stage %d - will not exist after reconfiguration", stageNum)
		}
	}

	return nil
}

// GetWorkflowStatus returns current workflow status for a document
func (s *SmartContract) GetWorkflowStatus(ctx contractapi.TransactionContextInterface, id string) (*WorkflowConfig, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	return &doc.Workflow, nil
}

// ReturnDocumentToStage returns a document to a previous stage with enhanced deadline handling
func (s *SmartContract) ReturnDocumentToStage(ctx contractapi.TransactionContextInterface, id, invokerId string, targetStage int, comment, deadlineOptionsJSON string) (*ReturnToStageResponse, error) {
	if id == "" || invokerId == "" {
		return nil, fmt.Errorf("document ID and invoker ID cannot be empty")
	}

	if targetStage < 1 {
		return nil, fmt.Errorf("target stage must be 1 or greater")
	}

	timestamp := time.Now().Format(time.RFC3339)

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Check if workflow is enabled
	if !doc.Workflow.Enabled {
		return nil, fmt.Errorf("workflow is not enabled for document %s", id)
	}

	// Validate target stage
	if targetStage > len(doc.Workflow.Stages) {
		return nil, fmt.Errorf("target stage %d does not exist (max: %d)", targetStage, len(doc.Workflow.Stages))
	}

	// Check authorization - only privileged users can return documents to previous stages
	if !s.hasPrivilegedAccess(invokerId, &doc) {
		return nil, fmt.Errorf("user %s does not have authorization to return document to previous stage", invokerId)
	}

	// ENHANCED: Validate deadline conflicts
	deadlineConflicts, err := s.validateRollbackDeadlines(&doc, targetStage)
	if err != nil {
		return nil, fmt.Errorf("failed to validate deadlines: %v", err)
	}

	// Initialize deadline changes map (never nil for Fabric compatibility)
	deadlineChanges := make(map[string]string)

	// Handle deadline conflicts if any exist
	if len(deadlineConflicts) > 0 {
		if deadlineOptionsJSON == "" {
			// No resolution provided - return conflicts for user to resolve
			response := &ReturnToStageResponse{
				Success:           false,
				DeadlineConflicts: deadlineConflicts,
				DeadlineChanges:   deadlineChanges,
			}
			return response, fmt.Errorf("deadline conflicts detected - resolution required")
		}

		// Parse deadline resolution options
		var deadlineOptions DeadlineRollbackOptions
		if err := json.Unmarshal([]byte(deadlineOptionsJSON), &deadlineOptions); err != nil {
			return nil, fmt.Errorf("invalid deadline options JSON: %v", err)
		}

		// Apply deadline conflict resolution
		deadlineChanges, err = s.resolveDeadlineConflicts(&doc, deadlineOptions, deadlineConflicts)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve deadline conflicts: %v", err)
		}
	}

	// Perform enhanced rollback with comprehensive tracking
	response := s.performEnhancedRollback(&doc, targetStage, comment, invokerId)
	response.DeadlineConflicts = deadlineConflicts
	response.DeadlineChanges = deadlineChanges

	// Update document timestamp
	doc.LastModifiedTimestamp = timestamp

	// Ensure all arrays are initialized (Fabric null-safety)
	s.ensureDocumentArraysInitialized(&doc)

	// Save updated document
	updatedData, err := json.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %v", err)
	}

	err = ctx.GetStub().PutState(id, updatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %v", err)
	}

	// Emit enhanced event
	description := fmt.Sprintf("Document %s returned to stage %d by %s", id, targetStage, invokerId)
	details := map[string]interface{}{
		"targetStage":        targetStage,
		"returnedBy":         invokerId,
		"returnComment":      comment,
		"timestamp":          timestamp,
		"resetStages":        response.ResetStages,
		"approvalsCleared":   response.ApprovalsCleared,
		"deadlineConflicts":  len(deadlineConflicts),
		"deadlineChanges":    len(deadlineChanges),
	}

	// Emit event (but don't fail rollback if event fails)
	s.emitEvent(ctx, "DocumentReturnedToStage", invokerId, description, doc.ID, doc.LatestVersion, details)

	return &response, nil
}

// AdvanceDocumentToNextStage manually advances document to next stage (for AutoAdvance=false scenarios)
func (s *SmartContract) AdvanceDocumentToNextStage(ctx contractapi.TransactionContextInterface, id, invokerId string) error {
	if id == "" || invokerId == "" {
		return fmt.Errorf("document ID and invoker ID cannot be empty")
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	// Check authorization - only uploader or privileged users can manually advance stages
	if doc.Uploader != invokerId && !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("user %s does not have authorization to advance document stages", invokerId)
	}

	// Check if workflow is enabled
	if !doc.Workflow.Enabled {
		return fmt.Errorf("workflow is not enabled for document %s", id)
	}

	// Check if we're at the final stage
	if doc.Workflow.CurrentStage >= len(doc.Workflow.Stages) {
		return fmt.Errorf("document is already at the final stage")
	}

	currentStageIndex := doc.Workflow.CurrentStage - 1
	currentStage := &doc.Workflow.Stages[currentStageIndex]

	// Check if current stage is actually completed
	approvedCount := 0
	for _, stageDecision := range currentStage.StageApprovals {
		if stageDecision.Status == Approved {
			approvedCount++
		}
	}

	requiredCount := currentStage.RequiredCount
	if requiredCount <= 0 {
		requiredCount = len(currentStage.Approvers)
	}

	if approvedCount < requiredCount {
		return fmt.Errorf("current stage %d is not yet completed (%d/%d approvals)", doc.Workflow.CurrentStage, approvedCount, requiredCount)
	}

	// Check if AutoAdvance is enabled (manual advance not needed)
	if currentStage.AutoAdvance {
		return fmt.Errorf("stage %d has AutoAdvance enabled - manual advance not needed", doc.Workflow.CurrentStage)
	}

	// Mark current stage as completed if not already
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

	// Advance to next stage
	doc.Workflow.CurrentStage++

	// Add stage transition to history
	transition := StageTransition{
		FromStage:  doc.Workflow.CurrentStage - 1,
		ToStage:    doc.Workflow.CurrentStage,
		Transition: "advanced",
		Actor:      invokerId,
		Timestamp:  timestamp,
		Comment:    "Manual stage advancement",
	}
	doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, transition)

	// Update document modification time
	doc.LastModifiedTimestamp = timestamp

	// Save document
	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Emit notification for next stage approvers
	nextStage := &doc.Workflow.Stages[doc.Workflow.CurrentStage-1]
	description := fmt.Sprintf("Document %s manually advanced to stage %d by %s", id, doc.Workflow.CurrentStage, invokerId)
	details := map[string]interface{}{
		"stage":         doc.Workflow.CurrentStage,
		"stageName":     nextStage.StageName,
		"approvers":     nextStage.Approvers,
		"requiredCount": nextStage.RequiredCount,
		"advancedBy":    invokerId,
		"manualAdvance": true,
		"timestamp":     timestamp,
	}

	return s.emitEvent(ctx, "STAGE_READY_FOR_APPROVAL", invokerId, description, doc.ID, doc.LatestVersion, details)
}