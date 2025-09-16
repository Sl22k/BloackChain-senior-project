package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ConfigureDocumentWorkflow sets up multi-stage workflow for a document
func (s *SmartContract) ConfigureDocumentWorkflow(ctx contractapi.TransactionContextInterface, id, invokerId, stagesJSON string) error {
	if id == "" || invokerId == "" || stagesJSON == "" {
		return fmt.Errorf("required parameters cannot be empty")
	}

	// Parse stages JSON
	var stages []WorkflowStage
	if err := json.Unmarshal([]byte(stagesJSON), &stages); err != nil {
		return fmt.Errorf("invalid stages JSON: %v", err)
	}

	// Validate stages structure
	if len(stages) == 0 {
		return fmt.Errorf("at least one stage is required")
	}

	for i, stage := range stages {
		if len(stage.Approvers) == 0 {
			return fmt.Errorf("stage %d must have at least one approver", i+1)
		}
		if stage.StageNumber != i+1 {
			return fmt.Errorf("stage numbers must be sequential starting from 1")
		}
	}

	timestamp := time.Now().Format(time.RFC3339)

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Check authorization (only uploader or privileged users can configure workflow)
	if doc.Uploader != invokerId && !s.hasPrivilegedAccess(invokerId, &doc) {
		return fmt.Errorf("user %s is not authorized to configure workflow for document %s", invokerId, id)
	}

	// Initialize workflow stages with approval tracking
	initializedStages := s.initializeStageApprovals(stages, timestamp)

	// Configure document workflow
	doc.Workflow = WorkflowConfig{
		Enabled:         true,
		CurrentStage:    1,
		Stages:          initializedStages,
		CompletedStages: []int{},
		StageHistory:    []StageTransition{},
	}

	updatedData, err := json.Marshal(&doc)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, updatedData)
	if err != nil {
		return err
	}

	description := fmt.Sprintf("Multi-stage workflow configured for document %s with %d stages", doc.ID, len(stages))
	details := map[string]interface{}{
		"totalStages":  len(stages),
		"currentStage": 1,
		"configured":   true,
	}

	return s.emitEvent(ctx, "WorkflowConfigured", invokerId, description, doc.ID, doc.LatestVersion, details)
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

// ReturnDocumentToStage returns a document to a previous stage for re-approval
func (s *SmartContract) ReturnDocumentToStage(ctx contractapi.TransactionContextInterface, id, invokerId string, targetStage int, comment string) error {
	if id == "" || invokerId == "" {
		return fmt.Errorf("document ID and invoker ID cannot be empty")
	}

	if targetStage < 1 {
		return fmt.Errorf("target stage must be 1 or greater")
	}

	timestamp := time.Now().Format(time.RFC3339)

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Check if workflow is enabled
	if !doc.Workflow.Enabled {
		return fmt.Errorf("workflow is not enabled for document %s", id)
	}

	// Validate target stage
	if targetStage > len(doc.Workflow.Stages) {
		return fmt.Errorf("target stage %d does not exist (max: %d)", targetStage, len(doc.Workflow.Stages))
	}

	// Check authorization - only privileged users can return documents to previous stages
	if !s.hasPrivilegedAccess(invokerId, &doc) {
		return fmt.Errorf("user %s does not have authorization to return document to previous stage", invokerId)
	}

	// Reset workflow using reusable utility function
	s.resetWorkflowFromStage(&doc, targetStage)

	updatedData, err := json.Marshal(&doc)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, updatedData)
	if err != nil {
		return err
	}

	description := fmt.Sprintf("Document %s returned to stage %d by %s", id, targetStage, invokerId)
	details := map[string]interface{}{
		"targetStage":   targetStage,
		"returnedBy":    invokerId,
		"returnComment": comment,
		"timestamp":     timestamp,
	}

	return s.emitEvent(ctx, "DocumentReturnedToStage", invokerId, description, doc.ID, doc.LatestVersion, details)
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