package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ============================================================================
// PHASE 3: MULTI-STAGE WORKFLOW FUNCTIONS - CLEANED VERSION
// ============================================================================

// ConfigureDocumentWorkflow sets up multi-stage workflow for a document
func (s *SmartContract) ConfigureDocumentWorkflow(ctx contractapi.TransactionContextInterface, id, invokerId, stagesJSON string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":         id,
		"invokerId":  invokerId,
		"stagesJSON": stagesJSON,
	}); err != nil {
		return err
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

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

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
	initializedStages := s.initializeStageApprovals(stages, date, timeStr)

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
	if err := s.validateInputs(map[string]string{"id": id}); err != nil {
		return nil, err
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
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":        id,
		"invokerId": invokerId,
	}); err != nil {
		return err
	}

	if targetStage < 1 {
		return fmt.Errorf("target stage must be 1 or greater")
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

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

	// Reset stages from target stage onwards
	for i := targetStage - 1; i < len(doc.Workflow.Stages); i++ {
		// Reset stage approvals
		doc.Workflow.Stages[i].StageApprovals = make(map[string]Decision)
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
		"timestamp":     fmt.Sprintf("%s %s", date, timeStr),
	}

	return s.emitEvent(ctx, "DocumentReturnedToStage", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) hasPrivilegedAccess(invokerId string, doc *Document) bool {
	// Check if user is document uploader
	if doc.Uploader == invokerId {
		return true
	}
	
	// Additional privilege checks could be added here (admin roles, etc.)
	return false
}