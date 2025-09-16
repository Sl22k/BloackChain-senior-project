package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// InitLedger initializes the ledger with empty state
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// SubmitDocument creates a document with workflow stages
// stagesJSON: [{"name":"","approvers":["alice"],"requiredApprovals":1,"autoAdvance":true}]
// Single stage: name can be empty. Multi-stage: names required
func (s *SmartContract) SubmitDocument(ctx contractapi.TransactionContextInterface, 
	id, title, description, hash, uploader, validDecisionsJSON, stagesJSON string) error {

	// Essential validation - document doesn't already exist
	if exists, err := s.DocumentExists(ctx, id); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("document exists")
	}

	// Parse valid decisions
	var validDecisions []string
	if validDecisionsJSON != "" {
		if err := json.Unmarshal([]byte(validDecisionsJSON), &validDecisions); err != nil {
			return fmt.Errorf("invalid validDecisions JSON: %v", err)
		}
	}
	if len(validDecisions) == 0 {
		validDecisions = []string{"APPROVED", "REJECTED"}
	}

	// Parse stages from frontend
	type StageInput struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		Approvers        []string `json:"approvers"`
		RequiredApprovals int     `json:"requiredApprovals"`
		AutoAdvance      bool     `json:"autoAdvance"`
	}
	
	var stageInputs []StageInput
	if err := json.Unmarshal([]byte(stagesJSON), &stageInputs); err != nil {
		return fmt.Errorf("invalid stages JSON: %v", err)
	}
	
	if len(stageInputs) == 0 {
		return fmt.Errorf("at least one stage is required")
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Create workflow stages with progressive disclosure logic
	var stages []WorkflowStage
	for i, input := range stageInputs {
		stageName := input.Name
		if stageName == "" {
			if len(stageInputs) == 1 {
				// Single stage - use default name
				stageName = "Document Approval"
			} else {
				// Multi-stage but no name provided - error
				return fmt.Errorf("stage names are required when using multiple stages")
			}
		}

		stage := WorkflowStage{
			StageNumber:    i + 1,
			StageName:      stageName,
			Description:    input.Description,
			Approvers:      input.Approvers,
			RequiredCount:  input.RequiredApprovals,
			StageApprovals: make(map[string]Decision),
			AutoAdvance:    input.AutoAdvance,
		}

		// Initialize stage approvals
		for _, approver := range input.Approvers {
			stage.StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}

		stages = append(stages, stage)
	}

	// Create unified document with workflow (always enabled)
	doc := Document{
		ID:                    id,
		Title:                 title,
		Description:           description,
		LatestVersion:         1,
		Uploader:              uploader,
		PrivilegedEditors:     []string{},
		Editors:               []string{},
		ValidDecisions:        validDecisions,
		CreatedTimestamp:      timestamp,
		LastModifiedTimestamp: timestamp,
		Workflow: WorkflowConfig{
			Enabled:         true,
			Stages:          stages,
			CurrentStage:    1,
			CompletedStages: []int{},
			StageHistory:    []StageTransition{},
		},
		VersionWorkflows: make(map[string]*WorkflowConfig),
		ApprovalsMap:     s.createInitialApprovalsMap(stages, timestamp), // Direct initialization
		DeadlineConfig: DeadlineConfig{
			StageDeadlines:    make(map[string]string),
			EscalationTargets: []string{},
			AutomationHooks:   []AutomationHook{},
		},
		DeadlineStatus: DeadlineStatus{
			DocumentId:        id,
			OverallStatus:     "NONE",
			StageStatuses:     make(map[string]StageDeadlineStatus),
			WarningsTriggered: []DeadlineWarning{},
			BreachesRecorded:  []DeadlineBreach{},
		},
	}

	// Initialize arrays and maps
	s.ensureDocumentArraysInitialized(&doc)
	s.ensureWorkflowArraysInitialized(&doc.Workflow)

	// Create initial version with copy of current approvals (proper version snapshot)
	initialVersion := DocumentVersion{
		Version:          1,
		Hash:             hash,
		Submitter:        uploader,
		Timestamp:        timestamp,
		ApprovalsMap:     s.copyApprovalsMap(doc.ApprovalsMap), // Copy for version snapshot
		ValidDecisions:   validDecisions,
		ApproversChanged: false,
		DecisionsChanged: false,
		WorkflowSnapshot: s.createWorkflowSnapshot(doc.Workflow, timestamp),
	}

	doc.Versions = []DocumentVersion{initialVersion}

	// Store workflow state for version tracking
	workflowCopy := s.deepCopyWorkflowConfig(doc.Workflow)
	doc.VersionWorkflows["1"] = &workflowCopy

	// Store document
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(id, data); err != nil {
		return err
	}

	// Emit event
	details := map[string]interface{}{
		"title":       title,
		"description": description,
		"uploader":    uploader,
		"stagesCount": len(stages),
		"isMultiStage": len(stages) > 1,
	}
	return s.emitEvent(ctx, "DOCUMENT_SUBMITTED", uploader, 
		fmt.Sprintf("Document '%s' submitted with %d stage(s)", title, len(stages)), id, 1, details)
}