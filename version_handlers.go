package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SubmitNewVersion creates a new document version with unified modification approach
// Supports content changes, approver list changes, and valid decisions changes in single operation
func (s *SmartContract) SubmitNewVersion(ctx contractapi.TransactionContextInterface, documentID, submitter, newStateJSON string) error {
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

	// Essential security - validate authorization
	if err := s.validateAuthorization(submitter, doc, "edit"); err != nil {
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
	if err := s.applyVersionChanges(doc, newState, changes, timestamp, submitter); err != nil {
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
func (s *SmartContract) applyVersionChanges(doc *Document, newState NewVersionState, changes DetectedChanges, timestamp, submitter string) error {
	// Preserve current workflow state for version history (BEFORE modifications)
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}

	// Save the CURRENT version's state before we modify it (for previousStage reference)
	previousVersionWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)

	// Update document metadata
	doc.LatestVersion++
	doc.LastModifiedTimestamp = timestamp

	// Store the previous version's workflow under the PREVIOUS version number
	if doc.LatestVersion > 1 {
		doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion-1)] = &previousVersionWorkflowState
	}

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

	// Handle approval reset logic and add StageHistory entry FIRST
	previousStage := previousVersionWorkflowState.CurrentStage

	if newState.ResetApprovals {
		// Reset workflow to initial state
		doc.Workflow.CurrentStage = 1
		doc.Workflow.CompletedStages = []int{}

		// Add version reset entry to stage history
		doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, StageTransition{
			FromStage:  previousStage,
			ToStage:    1,
			Transition: "version_reset",
			Actor:      submitter,
			Timestamp:  timestamp,
			Comment:    fmt.Sprintf("Approvals reset for version %d: %s", doc.LatestVersion, newState.ChangeReason),
		})
	} else {
		// Add version update entry to stage history (preserving approvals)
		doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, StageTransition{
			FromStage:  previousStage,
			ToStage:    doc.Workflow.CurrentStage,
			Transition: "version_update",
			Actor:      submitter,
			Timestamp:  timestamp,
			Comment:    fmt.Sprintf("Version %d updated: %s", doc.LatestVersion, newState.ChangeReason),
		})
	}

	// Apply stage updates using unified model (AFTER reset logic)
	if len(newState.StageUpdates) > 0 {
		if err := s.updateWorkflowStages(doc, newState.StageUpdates, timestamp); err != nil {
			return fmt.Errorf("failed to update workflow stages: %v", err)
		}

		// Rebuild ApprovalsMap from all stages
		doc.ApprovalsMap = s.createInitialApprovalsMap(doc.Workflow.Stages, timestamp)

		// If not resetting approvals, preserve existing approvals for unchanged approvers
		if !newState.ResetApprovals && len(doc.Versions) > 0 {
			lastVersionApprovals := doc.Versions[len(doc.Versions)-1].ApprovalsMap

			// DETERMINISM FIX: Sort keys to ensure consistent iteration order across all peers
			var sortedKeys []string
			for key := range lastVersionApprovals {
				sortedKeys = append(sortedKeys, key)
			}
			sort.Strings(sortedKeys)

			// Iterate in deterministic order
			for _, key := range sortedKeys {
				if _, exists := doc.ApprovalsMap[key]; exists {
					// Approver still exists, keep their previous decision
					doc.ApprovalsMap[key] = lastVersionApprovals[key]
				}
			}
		}

		// SYNC FIX: When preserving approvals, sync ApprovalsMap back to StageApprovals
		if !newState.ResetApprovals {
			s.syncApprovalsMapToStageApprovals(doc)
		}
	}

	// CRITICAL FIX: Apply reset logic to stage approvals AFTER stage updates
	if newState.ResetApprovals {
		// Reset all stage approvals (but preserve updated stage configurations)
		doc.Workflow.Stages = s.initializeStageApprovals(doc.Workflow.Stages, timestamp)

		// Reset ApprovalsMap to all pending
		doc.ApprovalsMap = s.createInitialApprovalsMap(doc.Workflow.Stages, timestamp)
	}

	// Create new version entry with proper approval snapshot
	newVersionApprovals := s.copyApprovalsMap(doc.ApprovalsMap)

	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             contentHash,
		Submitter:        submitter,
		Timestamp:        timestamp,
		ApprovalsMap:     newVersionApprovals,
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: len(changes.ApproversAdded) > 0 || len(changes.ApproversRemoved) > 0,
		DecisionsChanged: len(changes.DecisionsAdded) > 0 || len(changes.DecisionsRemoved) > 0,
		ContentChanged:   changes.ContentChanged,
		ChangeType:       s.determineChangeType(changes),
		WorkflowSnapshot: s.createWorkflowSnapshot(doc.Workflow, timestamp),
	}

	doc.Versions = append(doc.Versions, newVersion)

	// Store the NEW version's workflow state (with updated StageHistory)
	newVersionWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)
	doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &newVersionWorkflowState

	return nil
}

// updateWorkflowStages updates specific workflow stages with new configurations
func (s *SmartContract) updateWorkflowStages(doc *Document, stageUpdates map[string]StageUpdate, timestamp string) error {
	if len(stageUpdates) == 0 {
		return nil // No stage updates requested
	}

	if len(doc.Workflow.Stages) == 0 {
		return fmt.Errorf("cannot update stages - document has no workflow stages")
	}

	// DETERMINISM FIX: Sort stage keys to ensure consistent processing order across all peers
	var sortedStageKeys []string
	for stageKey := range stageUpdates {
		sortedStageKeys = append(sortedStageKeys, stageKey)
	}
	sort.Strings(sortedStageKeys)

	// Validate all stage numbers first (in deterministic order)
	for _, stageKey := range sortedStageKeys {
		stageNum, err := strconv.Atoi(stageKey)
		if err != nil {
			return fmt.Errorf("invalid stage number: %s", stageKey)
		}
		if stageNum < 1 || stageNum > len(doc.Workflow.Stages) {
			return fmt.Errorf("stage %d does not exist (valid range: 1-%d)", stageNum, len(doc.Workflow.Stages))
		}
		if len(stageUpdates[stageKey].Approvers) == 0 {
			return fmt.Errorf("stage %d cannot have empty approvers list", stageNum)
		}
	}

	// Apply updates to specified stages (in deterministic order)
	for _, stageKey := range sortedStageKeys {
		update := stageUpdates[stageKey]
		stageNum, _ := strconv.Atoi(stageKey) // Already validated above
		stageIndex := stageNum - 1
		stage := &doc.Workflow.Stages[stageIndex]

		// Update stage configuration - DETERMINISM FIX: Sort approvers to ensure consistent order across all peers
		sortedApprovers := make([]string, len(update.Approvers))
		copy(sortedApprovers, update.Approvers)
		sort.Strings(sortedApprovers)
		stage.Approvers = sortedApprovers
		stage.RequiredCount = update.RequiredCount
		stage.AutoAdvance = update.AutoAdvance

		// Reinitialize stage approvals for new approvers
		stage.StageApprovals = make(map[string]Decision)
		for _, approver := range sortedApprovers {
			stage.StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return nil
}

// determineChangeType creates compound ChangeType based on detected changes
func (s *SmartContract) determineChangeType(changes DetectedChanges) string {
	var changeComponents []string

	if changes.ContentChanged {
		changeComponents = append(changeComponents, "content")
	}
	if len(changes.ApproversAdded) > 0 || len(changes.ApproversRemoved) > 0 {
		changeComponents = append(changeComponents, "approvers")
	}
	if len(changes.DecisionsAdded) > 0 || len(changes.DecisionsRemoved) > 0 {
		changeComponents = append(changeComponents, "decisions")
	}

	if len(changeComponents) == 0 {
		return "metadata" // Fallback for edge cases
	}

	// Return compound change type like "content+approvers" or single type like "content"
	return strings.Join(changeComponents, "+")
}

// syncApprovalsMapToStageApprovals synchronizes ApprovalsMap back to StageApprovals
// This fixes the data inconsistency bug when resetApprovals=false
func (s *SmartContract) syncApprovalsMapToStageApprovals(doc *Document) {
	// Iterate through all stages
	for i := range doc.Workflow.Stages {
		stage := &doc.Workflow.Stages[i]

		// For each approver in this stage, sync their approval from ApprovalsMap
		for _, approver := range stage.Approvers {
			if approval, exists := doc.ApprovalsMap[approver]; exists {
				// Copy the approval from ApprovalsMap to StageApprovals
				if stage.StageApprovals == nil {
					stage.StageApprovals = make(map[string]Decision)
				}
				stage.StageApprovals[approver] = approval
			}
		}
	}
}