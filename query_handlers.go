package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// QueryDocumentStatus returns detailed status information for a document
func (s *SmartContract) QueryDocumentStatus(ctx contractapi.TransactionContextInterface, id string) (*StatusResponse, error) {
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

	// Ensure all arrays are initialized (minimal defensive check)
	s.ensureDocumentArraysInitialized(&doc)
	// No need for ensureAllWorkflowFieldsInitialized - data should be correct from creation/updates

	// Calculate decision counts
	decisionCounts := make(map[string]int)
	approvedCount := 0
	rejectedCount := 0
	pendingCount := 0

	for _, decision := range doc.ApprovalsMap {
		switch decision.Status {
		case Approved:
			approvedCount++
			decisionCounts["APPROVED"]++
		case Rejected:
			rejectedCount++
			decisionCounts["REJECTED"]++
		case Pending:
			pendingCount++
			decisionCounts["PENDING"]++
		}
	}

	// Ensure WorkflowSnapshot exists in all versions (schema requirement)
	for i := range doc.Versions {
		if doc.Versions[i].WorkflowSnapshot == nil {
			// Create a proper WorkflowSnapshot for this version
			doc.Versions[i].WorkflowSnapshot = s.createWorkflowSnapshot(doc.Workflow, doc.Versions[i].Timestamp)
		}
	}

	// Get current stage info (required field - initialize even for simple docs)
	var currentStageInfo *WorkflowStage
	if doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 && doc.Workflow.CurrentStage > 0 && doc.Workflow.CurrentStage <= len(doc.Workflow.Stages) {
		currentStageInfo = &doc.Workflow.Stages[doc.Workflow.CurrentStage-1]
	} else {
		// For simple documents, create a default stage info
		currentStageInfo = &WorkflowStage{
			StageNumber:    1,
			StageName:      "Simple Approval",
			RequiredCount:  0,
			Approvers:      []string{},
			StageApprovals: make(map[string]Decision),
			AutoAdvance:    false,
			Description:    "Single-stage approval process",
		}
		// Copy approvers from ApprovalsMap for simple docs
		for approver := range doc.ApprovalsMap {
			currentStageInfo.Approvers = append(currentStageInfo.Approvers, approver)
			currentStageInfo.StageApprovals[approver] = doc.ApprovalsMap[approver]
		}
	}

	// Calculate WorkflowProgress (required field)
	var workflowProgress *WorkflowProgressOverview
	if doc.Workflow.Enabled {
		workflowProgress = s.calculateWorkflowProgress(doc.Workflow)
	} else {
		// Create progress info for simple documents
		totalApprovers := len(doc.ApprovalsMap)
		overallProgress := float64(0)
		if totalApprovers > 0 {
			overallProgress = float64(approvedCount) / float64(totalApprovers) * 100
		}
		
		statusSummary := "Simple Approval"
		if approvedCount == totalApprovers && totalApprovers > 0 {
			statusSummary = "Completed"
		} else if rejectedCount > 0 {
			statusSummary = "Rejected"  
		} else {
			statusSummary = "Pending Approval"
		}
		
		workflowProgress = &WorkflowProgressOverview{
			CurrentStageNumber:   1,
			CurrentStageName:     "Simple Approval",
			CurrentStageProgress: overallProgress,
			OverallProgress:      overallProgress,
			CompletedStages:      0,
			TotalStages:          1,
			StatusSummary:        statusSummary,
			NextAction:           "Waiting for approvals",
			StageDetails:         []StageProgressSummary{},
		}
	}

	response := &StatusResponse{
		ID:                doc.ID,
		Title:             doc.Title,
		Description:       doc.Description,
		LatestVersion:     doc.LatestVersion,
		// Versions field removed - use GetDocumentHistory() for version history
		Uploader:          doc.Uploader,
		PrivilegedEditors: doc.PrivilegedEditors,
		Editors:           doc.Editors,
		ApprovalsMap:      doc.ApprovalsMap,
		DecisionCounts:    decisionCounts,
		ApprovedCount:     approvedCount,
		RejectedCount:     rejectedCount,
		PendingCount:      pendingCount,
		ValidDecisions:    doc.ValidDecisions,
		Workflow:          doc.Workflow,
		CurrentStageInfo:  currentStageInfo,
		WorkflowProgress:  workflowProgress,
		// VersionWorkflows field removed - use GetDocumentHistory() for version workflows
		DeadlineConfig:    doc.DeadlineConfig,
		DeadlineStatus:    doc.DeadlineStatus,
		CreatedTimestamp:      doc.CreatedTimestamp,
		LastModifiedTimestamp: doc.LastModifiedTimestamp,
	}

	return response, nil
}

// GetDocumentHistory returns complete version history for a document
func (s *SmartContract) GetDocumentHistory(ctx contractapi.TransactionContextInterface, id string) ([]DocumentVersion, error) {
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	// Return complete version history
	return doc.Versions, nil
}

// GetDocumentVersion returns a specific version of a document
func (s *SmartContract) GetDocumentVersion(ctx contractapi.TransactionContextInterface, id string, version int) (*DocumentVersion, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid version")
	}

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	// Find the requested version
	for _, v := range doc.Versions {
		if v.Version == version {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("version %d not found for document %s", version, id)
}

// GetDocumentVersionWorkflow returns the workflow configuration for a specific version
func (s *SmartContract) GetDocumentVersionWorkflow(ctx contractapi.TransactionContextInterface, id string, version int) (*WorkflowConfig, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid version")
	}

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
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

	return nil, fmt.Errorf("workflow configuration for version %d not found", version)
}

