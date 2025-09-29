package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ApproveDocument processes approval for current workflow stage
func (s *SmartContract) ApproveDocument(ctx contractapi.TransactionContextInterface, id, approver, decision, comment string) error {
	// Essential data integrity - load document
	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	// Essential security - validate authorization
	if err := s.validateAuthorization(approver, doc, "approve"); err != nil {
		return err
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Reconsideration validation - check if this is a reconsideration attempt
	if err := s.validateReconsiderationAttempt(doc, approver); err != nil {
		return err
	}

	// Essential validation - workflow stages configured
	if len(doc.Workflow.Stages) == 0 {
		return fmt.Errorf("no workflow stages")
	}

	// Essential validation - current stage index
	currentStageIndex := doc.Workflow.CurrentStage - 1
	if currentStageIndex < 0 || currentStageIndex >= len(doc.Workflow.Stages) {
		return fmt.Errorf("invalid stage")
	}

	currentStage := &doc.Workflow.Stages[currentStageIndex]

	// Check if approver is authorized for current stage
	authorized := false
	for _, stageApprover := range currentStage.Approvers {
		if stageApprover == approver {
			authorized = true
			break
		}
	}
	if !authorized {
		return fmt.Errorf("not authorized")
	}

	// Essential validation - already decided at this stage (only if NOT a valid reconsideration)
	if existingDecision, exists := currentStage.StageApprovals[approver]; exists && existingDecision.Status != Pending {
		// For multi-version documents, the validateReconsiderationAttempt() above already handled this
		// Only block if this is the first version (no reconsideration rules apply)
		if len(doc.Versions) < 2 {
			return fmt.Errorf("already decided")
		}
		// For multi-version documents, validateReconsiderationAttempt() has already validated whether
		// this reconsideration is allowed. If we reached here, it means it's allowed.
	}

	// Essential validation - decision against document's valid decisions
	validDecision := false
	for _, validDec := range doc.ValidDecisions {
		if decision == validDec {
			validDecision = true
			break
		}
	}
	if !validDecision {
		return fmt.Errorf("invalid decision")
	}

	// Enhanced reconsideration validation for decisions-only changes
	if err := s.validateReconsiderationDecision(doc, approver, decision); err != nil {
		return err
	}

	// Apply decision to current stage
	status := Pending
	switch decision {
	case "APPROVED":
		status = Approved
	case "REJECTED":
		status = Rejected
	}

	newDecision := Decision{
		Status:    status,
		Comment:   comment,
		Timestamp: timestamp,
	}

	// Initialize stage approvals map if needed
	if currentStage.StageApprovals == nil {
		currentStage.StageApprovals = make(map[string]Decision)
	}

	// Update stage approval
	currentStage.StageApprovals[approver] = newDecision
	
	// Update document-level ApprovalsMap directly (no sync functions needed)
	approverKey := s.generateApproverKey(approver, doc.Workflow.CurrentStage, len(doc.Workflow.Stages))
	doc.ApprovalsMap[approverKey] = newDecision

	// Check if stage is complete and handle advancement
	switch status {
	case Rejected:
		// Document rejected - workflow stops here
		// No need to advance stages on rejection
	case Approved:
		// Use enhanced stage advancement with proper notifications
		if err := s.handleStageAdvancement(ctx, doc, currentStage, approver); err != nil {
			return err
		}
	}

	// Update latest version's approvals directly (proper version snapshot maintenance)
	if len(doc.Versions) > 0 {
		latestVersionIndex := len(doc.Versions) - 1
		doc.Versions[latestVersionIndex].ApprovalsMap[approverKey] = newDecision
	}

	// CRITICAL FIX: Update VersionWorkflows snapshot to maintain consistency
	// This fixes the bug where version snapshots were frozen and not updated with approvals
	versionKey := fmt.Sprintf("%d", doc.LatestVersion)
	if doc.VersionWorkflows != nil && doc.VersionWorkflows[versionKey] != nil {
		versionWorkflow := doc.VersionWorkflows[versionKey]
		if currentStageIndex < len(versionWorkflow.Stages) {
			// Initialize stage approvals map if needed
			if versionWorkflow.Stages[currentStageIndex].StageApprovals == nil {
				versionWorkflow.Stages[currentStageIndex].StageApprovals = make(map[string]Decision)
			}

			// Update the version workflow snapshot with the new approval
			versionWorkflow.Stages[currentStageIndex].StageApprovals[approver] = newDecision

			// Synchronize completed stages and current stage with live workflow
			versionWorkflow.CompletedStages = make([]int, len(doc.Workflow.CompletedStages))
			copy(versionWorkflow.CompletedStages, doc.Workflow.CompletedStages)
			versionWorkflow.CurrentStage = doc.Workflow.CurrentStage

			// CRITICAL FIX: Also update the stored WorkflowSnapshot in the version
			if len(doc.Versions) > 0 {
				latestVersionIndex := len(doc.Versions) - 1
				doc.Versions[latestVersionIndex].WorkflowSnapshot = s.createWorkflowSnapshot(*versionWorkflow, timestamp)
			}
		}
	}

	// Update document modification time
	doc.LastModifiedTimestamp = timestamp

	// Store updated document
	updatedData, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(id, updatedData); err != nil {
		return err
	}

	// Emit event
	details := map[string]interface{}{
		"approver": approver,
		"decision": decision,
		"comment":  comment,
		"version":  doc.LatestVersion,
		"stage":    doc.Workflow.CurrentStage,
	}
	
	eventType := "DOCUMENT_APPROVED"
	if decision == "REJECT" {
		eventType = "DOCUMENT_REJECTED"
	}
	
	return s.emitEvent(ctx, eventType, approver, fmt.Sprintf("Document %s by %s", decision, approver), id, doc.LatestVersion, details)
}

