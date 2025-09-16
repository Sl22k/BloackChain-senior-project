package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitNewVersion creates a new document version with unified modification approach
// Supports content changes, approver list changes, and valid decisions changes in single operation
func (s *SmartContract) SubmitNewVersion(ctx contractapi.TransactionContextInterface, documentID, newStateJSON string) error {
	// Parse the new desired state
	var newState NewVersionState
	if err := json.Unmarshal([]byte(newStateJSON), &newState); err != nil {
		return fmt.Errorf("invalid newState JSON: %v", err)
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Essential data integrity - load document
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Detect what actually changed
	changes := s.detectChanges(*doc, newState)

	// Business rule validations
	if changes.NoChangesDetected {
		return fmt.Errorf("no changes detected - version not created")
	}

	if s.isDocumentFullyApproved(*doc) {
		return fmt.Errorf("cannot modify fully approved document")
	}

	// Multi-stage security: only allow "keep approvals" in stage 1
	if doc.Workflow.CurrentStage > 1 && !newState.ResetApprovals {
		return fmt.Errorf("can only keep approvals when document is in stage 1 (current stage: %d)", doc.Workflow.CurrentStage)
	}

	// Apply changes and create new version
	if err := s.applyVersionChanges(doc, newState, changes, timestamp); err != nil {
		return fmt.Errorf("failed to apply version changes: %v", err)
	}

	// Save updated document
	updatedData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal updated document: %v", err)
	}

	if err := ctx.GetStub().PutState(documentID, updatedData); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Create hash-to-ID index if content changed
	if changes.ContentChanged && newState.ContentHash != "" {
		hashKey := fmt.Sprintf("hash->%s", newState.ContentHash)
		if err := ctx.GetStub().PutState(hashKey, []byte(documentID)); err != nil {
			return fmt.Errorf("failed to create hash index: %v", err)
		}
	}

	// Emit event with change details
	details := map[string]interface{}{
		"contentChanged":   changes.ContentChanged,
		"approversAdded":   changes.ApproversAdded,
		"approversRemoved": changes.ApproversRemoved,
		"decisionsAdded":   changes.DecisionsAdded,
		"decisionsRemoved": changes.DecisionsRemoved,
		"resetApprovals":   newState.ResetApprovals,
		"changeReason":     newState.ChangeReason,
	}

	description := fmt.Sprintf("Version %d created: %s", doc.LatestVersion, newState.ChangeReason)
	return s.emitEvent(ctx, "NewVersionCreated", "", description, doc.ID, doc.LatestVersion, details)
}

// applyVersionChanges applies all detected changes to create the new version
func (s *SmartContract) applyVersionChanges(doc *Document, newState NewVersionState, changes DetectedChanges, timestamp string) error {
	// Preserve current workflow state for version history
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}
	
	currentWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)
	doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &currentWorkflowState

	// Update document metadata
	doc.LatestVersion++
	doc.LastModifiedTimestamp = timestamp

	// Apply content changes
	contentHash := newState.ContentHash
	if !changes.ContentChanged {
		// Use existing hash if content didn't change
		if len(doc.Versions) > 0 {
			contentHash = doc.Versions[len(doc.Versions)-1].Hash
		}
	}

	// Apply valid decisions changes
	if len(changes.DecisionsAdded) > 0 || len(changes.DecisionsRemoved) > 0 {
		doc.ValidDecisions = newState.ValidDecisions
	}

	// Apply approver changes by rebuilding workflow stages
	if len(changes.ApproversAdded) > 0 || len(changes.ApproversRemoved) > 0 {
		if err := s.updateWorkflowStages(doc, newState.ApproversList, timestamp); err != nil {
			return fmt.Errorf("failed to update workflow stages: %v", err)
		}
		
		// Rebuild ApprovalsMap with new approvers
		doc.ApprovalsMap = s.createInitialApprovalsMap(doc.Workflow.Stages, timestamp)
		
		// If not resetting approvals, preserve existing approvals for unchanged approvers
		if !newState.ResetApprovals && len(doc.Versions) > 0 {
			lastVersionApprovals := doc.Versions[len(doc.Versions)-1].ApprovalsMap
			for key, decision := range lastVersionApprovals {
				if _, exists := doc.ApprovalsMap[key]; exists {
					// Approver still exists, keep their previous decision
					doc.ApprovalsMap[key] = decision
				}
			}
		}
	}

	// Handle approval reset logic
	if newState.ResetApprovals {
		// Reset workflow to initial state
		doc.Workflow.CurrentStage = 1
		doc.Workflow.CompletedStages = []int{}
		
		// Reset all stage approvals
		doc.Workflow.Stages = s.initializeStageApprovals(doc.Workflow.Stages, timestamp)
		
		// Reset ApprovalsMap to all pending
		doc.ApprovalsMap = s.createInitialApprovalsMap(doc.Workflow.Stages, timestamp)
		
		// Reset stage history
		doc.Workflow.StageHistory = []StageTransition{{
			FromStage:  0,
			ToStage:    1,
			Transition: "version_reset",
			Actor:      "",
			Timestamp:  timestamp,
			Comment:    fmt.Sprintf("Approvals reset for version %d: %s", doc.LatestVersion, newState.ChangeReason),
		}}
	}

	// Create new version entry with proper approval snapshot
	newVersionApprovals := s.copyApprovalsMap(doc.ApprovalsMap)

	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             contentHash,
		Submitter:        "", // Will be set by caller context
		Timestamp:        timestamp,
		ApprovalsMap:     newVersionApprovals,
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: len(changes.ApproversAdded) > 0 || len(changes.ApproversRemoved) > 0,
		DecisionsChanged: len(changes.DecisionsAdded) > 0 || len(changes.DecisionsRemoved) > 0,
		WorkflowSnapshot: s.createWorkflowSnapshot(doc.Workflow, timestamp),
	}

	doc.Versions = append(doc.Versions, newVersion)

	// Store new workflow state
	workflowCopy := s.deepCopyWorkflowConfig(doc.Workflow)
	doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &workflowCopy

	// No synchronization functions needed - data is maintained correctly during operations

	return nil
}

// updateWorkflowStages rebuilds workflow stages with new approvers list
func (s *SmartContract) updateWorkflowStages(doc *Document, newApproversList []string, timestamp string) error {
	if len(doc.Workflow.Stages) == 0 {
		return fmt.Errorf("cannot update approvers - document has no workflow stages")
	}

	// For single stage documents, simply update the approvers list
	if len(doc.Workflow.Stages) == 1 {
		stage := &doc.Workflow.Stages[0]
		stage.Approvers = newApproversList
		
		// Reinitialize stage approvals for new approvers
		stage.StageApprovals = make(map[string]Decision)
		for _, approver := range newApproversList {
			stage.StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
		return nil
	}

	// For multi-stage documents, this is more complex
	// For now, we'll update all stages with the same approvers list
	// TODO: In future, frontend could specify approvers per stage
	for i := range doc.Workflow.Stages {
		stage := &doc.Workflow.Stages[i]
		stage.Approvers = newApproversList
		
		// Reinitialize stage approvals
		stage.StageApprovals = make(map[string]Decision)
		for _, approver := range newApproversList {
			stage.StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return nil
}