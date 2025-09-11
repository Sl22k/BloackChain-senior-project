package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// NEW: Workflow history audit functions - FIX FOR BUG #2

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
				CompletionDate:  version.WorkflowSnapshot.CompletionDate,
				CompletionTime:  version.WorkflowSnapshot.CompletionTime,
			}
			
			// SCHEMA FIX: Ensure completion fields are never empty for schema validation
			if snapshot.CompletionDate == "" {
				snapshot.CompletionDate = "Not Completed"
			}
			if snapshot.CompletionTime == "" {
				snapshot.CompletionTime = "Not Completed"
			}
			
			response.VersionSnapshots[versionKey] = snapshot
		} else {
			// Create empty snapshot for schema compliance
			response.VersionSnapshots[versionKey] = &WorkflowSnapshot{
				Enabled:         false,
				CurrentStage:    0,
				CompletedStages: []int{},
				TotalStages:     0,
				CompletionDate:  "Not Completed",
				CompletionTime:  "Not Completed",
			}
		}
	}
	
	return response, nil
}

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

// NEW: Enhanced defensive initialization function (private helper)
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
	}
}