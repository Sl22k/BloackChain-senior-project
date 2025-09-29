package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// GetDocumentWorkflowHistory returns workflow history for a document
func (s *SmartContract) GetDocumentWorkflowHistory(ctx contractapi.TransactionContextInterface, id string) (*WorkflowHistoryResponse, error) {
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
	
	// Initialize response
	response := &WorkflowHistoryResponse{
		DocumentID:       doc.ID,
		VersionCount:     doc.LatestVersion,
		CurrentVersion:   doc.LatestVersion,
		VersionSnapshots: make(map[string]*WorkflowSnapshot),
		VersionWorkflows: doc.VersionWorkflows,
		CurrentWorkflow:  &doc.Workflow,
	}
	
	// Initialize VersionWorkflows if nil
	if response.VersionWorkflows == nil {
		response.VersionWorkflows = make(map[string]*WorkflowConfig)
	}
	
	// Collect workflow snapshots from version history with schema compliance
	for _, version := range doc.Versions {
		versionKey := fmt.Sprintf("%d", version.Version)
		
		if version.WorkflowSnapshot != nil {
			// Create a copy and ensure schema compliance
			snapshot := &WorkflowSnapshot{
				Enabled:         version.WorkflowSnapshot.Enabled,
				CurrentStage:    version.WorkflowSnapshot.CurrentStage,
				CompletedStages: append([]int{}, version.WorkflowSnapshot.CompletedStages...),
				TotalStages:     version.WorkflowSnapshot.TotalStages,
				CompletionTimestamp: version.WorkflowSnapshot.CompletionTimestamp,
			}
			
			// SCHEMA FIX: Ensure completion fields are never empty for schema validation
			if snapshot.CompletionTimestamp == "" {
				snapshot.CompletionTimestamp = "Not Completed"
			}
			
			response.VersionSnapshots[versionKey] = snapshot
		} else {
			// Create empty snapshot for schema compliance
			response.VersionSnapshots[versionKey] = &WorkflowSnapshot{
				Enabled:         false,
				CurrentStage:    0,
				CompletedStages: []int{},
				TotalStages:     0,
				CompletionTimestamp: "Not Completed",
			}
		}
	}
	
	return response, nil
}

// GetVersionWorkflowState returns workflow state for a specific version
func (s *SmartContract) GetVersionWorkflowState(ctx contractapi.TransactionContextInterface, id string, version int) (*WorkflowConfig, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}
	if version <= 0 {
		return nil, fmt.Errorf("version must be positive, got %d", version)
	}
	
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}
	
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	
	// Check if this is the current version
	if version == doc.LatestVersion {
		return &doc.Workflow, nil
	}
	
	// Check in version workflows
	if doc.VersionWorkflows != nil {
		versionKey := fmt.Sprintf("%d", version)
		if workflowConfig, exists := doc.VersionWorkflows[versionKey]; exists {
			return workflowConfig, nil
		}
	}
	
	return nil, fmt.Errorf("workflow state for version %d not found", version)
}

// ensureAllWorkflowFieldsInitialized ensures all workflow fields are properly initialized
func (s *SmartContract) ensureAllWorkflowFieldsInitialized(doc *Document) {
	// Initialize VersionWorkflows if nil
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}
	
	// Ensure workflow arrays are initialized
	s.ensureWorkflowArraysInitialized(&doc.Workflow)
	
	// Initialize stage approvals for all workflow stages
	if doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		for i := range doc.Workflow.Stages {
			if doc.Workflow.Stages[i].StageApprovals == nil {
				doc.Workflow.Stages[i].StageApprovals = make(map[string]Decision)
			}
		}
	}
	
	// Initialize compatibility ApprovalsMap if nil
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}
	
	// Initialize version workflow snapshots
	for i := range doc.Versions {
		if doc.Versions[i].ApprovalsMap == nil {
			doc.Versions[i].ApprovalsMap = make(map[string]Decision)
		}
		// Ensure WorkflowSnapshot is initialized
		if doc.Versions[i].WorkflowSnapshot == nil {
			// BUG FIX: Use version-specific workflow state instead of current workflow
			versionKey := fmt.Sprintf("%d", doc.Versions[i].Version)
			timestamp := time.Now().Format(time.RFC3339)

			// Try to get the workflow state for this specific version
			if doc.VersionWorkflows != nil && doc.VersionWorkflows[versionKey] != nil {
				// Use the historical workflow state for this version
				doc.Versions[i].WorkflowSnapshot = s.createWorkflowSnapshot(*doc.VersionWorkflows[versionKey], timestamp)
			} else {
				// Fallback: Use current workflow state (maintains backward compatibility)
				doc.Versions[i].WorkflowSnapshot = s.createWorkflowSnapshot(doc.Workflow, timestamp)
			}
		}
		// Ensure CompletionTimestamp is never nil
		if doc.Versions[i].WorkflowSnapshot.CompletionTimestamp == "" {
			// Keep empty string, don't set to nil
		}
	}
	
	// Initialize DeadlineConfig - ensure empty values instead of nil
	if doc.DeadlineConfig.StageDeadlines == nil {
		doc.DeadlineConfig.StageDeadlines = make(map[string]string)
	}
	if doc.DeadlineConfig.EscalationTargets == nil {
		doc.DeadlineConfig.EscalationTargets = []string{}
	}
	if doc.DeadlineConfig.AutomationHooks == nil {
		doc.DeadlineConfig.AutomationHooks = []AutomationHook{}
	}
	
	// Initialize DeadlineStatus - ensure empty values instead of nil
	if doc.DeadlineStatus.DocumentId == "" && doc.ID != "" {
		doc.DeadlineStatus.DocumentId = doc.ID
	}
	if doc.DeadlineStatus.CurrentStageStatus == "" {
		doc.DeadlineStatus.CurrentStageStatus = "NONE"
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