package chaincode

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// generateApproverKey creates consistent approver keys for workflow tracking
func (s *SmartContract) generateApproverKey(approver string, stageNumber int, totalStages int) string {
	if totalStages == 1 {
		return approver
	}
	return fmt.Sprintf("#%d %s", stageNumber, approver)
}

// handleStageAdvancement processes stage completion and advancement with proper notifications
func (s *SmartContract) handleStageAdvancement(ctx contractapi.TransactionContextInterface, doc *Document, currentStage *WorkflowStage, approver, timestamp string) error {
	// Count approved decisions in current stage
	approvedCount := 0
	for _, stageDecision := range currentStage.StageApprovals {
		if stageDecision.Status == Approved {
			approvedCount++
		}
	}

	// Check if stage requirements are met (handle RequiredCount = 0 as "ALL")
	requiredCount := currentStage.RequiredCount
	if requiredCount <= 0 {
		requiredCount = len(currentStage.Approvers) // 0 means all approvers needed
	}

	if approvedCount >= requiredCount {
		// Mark current stage as completed
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

		// Handle stage advancement based on AutoAdvance flag
		if doc.Workflow.CurrentStage < len(doc.Workflow.Stages) {
			if currentStage.AutoAdvance {
				// Auto-advance to next stage and notify next stage approvers
				doc.Workflow.CurrentStage++

				// Emit notification for next stage approvers
				nextStage := &doc.Workflow.Stages[doc.Workflow.CurrentStage-1]
				details := map[string]interface{}{
					"stage":           doc.Workflow.CurrentStage,
					"stageName":       nextStage.StageName,
					"approvers":       nextStage.Approvers,
					"requiredCount":   nextStage.RequiredCount,
					"autoAdvanced":    true,
					"previousStage":   doc.Workflow.CurrentStage - 1,
				}

				if err := s.emitEvent(ctx, "STAGE_READY_FOR_APPROVAL", "system",
					fmt.Sprintf("Document %s advanced to stage %d - %s", doc.ID, doc.Workflow.CurrentStage, nextStage.StageName),
					doc.ID, doc.LatestVersion, details); err != nil {
					return err
				}
			} else {
				// Manual advance required - notify uploader/privileged users
				details := map[string]interface{}{
					"stage":              doc.Workflow.CurrentStage,
					"stageName":          currentStage.StageName,
					"nextStage":          doc.Workflow.CurrentStage + 1,
					"nextStageName":      doc.Workflow.Stages[doc.Workflow.CurrentStage].StageName,
					"approvedCount":      approvedCount,
					"requiredCount":      requiredCount,
					"uploader":           doc.Uploader,
					"manualAdvanceNeeded": true,
				}

				if err := s.emitEvent(ctx, "STAGE_COMPLETED_MANUAL_ADVANCE", doc.Uploader,
					fmt.Sprintf("Stage %d completed for document %s - manual advance needed", doc.Workflow.CurrentStage, doc.ID),
					doc.ID, doc.LatestVersion, details); err != nil {
					return err
				}
			}
		} else {
			// Final stage completed - document fully approved
			details := map[string]interface{}{
				"allStagesCompleted": true,
				"totalStages":        len(doc.Workflow.Stages),
				"finalApprover":      approver,
			}

			if err := s.emitEvent(ctx, "DOCUMENT_FULLY_APPROVED", approver,
				fmt.Sprintf("Document %s fully approved - all stages completed", doc.ID),
				doc.ID, doc.LatestVersion, details); err != nil {
				return err
			}
		}
	}

	return nil
}

// resetWorkflowFromStage resets workflow stages from target stage onwards (reusable reset logic)
func (s *SmartContract) resetWorkflowFromStage(doc *Document, targetStage int) {
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
}

// ensureWorkflowArraysInitialized ensures all workflow arrays are properly initialized
func (s *SmartContract) ensureWorkflowArraysInitialized(workflow *WorkflowConfig) {
	if workflow.Stages == nil {
		workflow.Stages = []WorkflowStage{}
	}
	if workflow.CompletedStages == nil {
		workflow.CompletedStages = []int{}
	}
	if workflow.StageHistory == nil {
		workflow.StageHistory = []StageTransition{}
	}

	// Initialize stage approvals for each stage
	for i := range workflow.Stages {
		if workflow.Stages[i].StageApprovals == nil {
			workflow.Stages[i].StageApprovals = make(map[string]Decision)
		}
	}
}

// ensureDocumentArraysInitialized ensures all document arrays are properly initialized
func (s *SmartContract) ensureDocumentArraysInitialized(doc *Document) {
	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}
	if doc.Versions == nil {
		doc.Versions = []DocumentVersion{}
	}

	// Initialize workflow arrays
	s.ensureWorkflowArraysInitialized(&doc.Workflow)
}

// initializeStageApprovals initializes stage approvals with pending status
func (s *SmartContract) initializeStageApprovals(stages []WorkflowStage, timestamp string) []WorkflowStage {
	initializedStages := make([]WorkflowStage, len(stages))

	for i, stage := range stages {
		initializedStages[i] = WorkflowStage{
			StageNumber:    stage.StageNumber,
			StageName:      stage.StageName,
			Approvers:      stage.Approvers,
			RequiredCount:  stage.RequiredCount,
			AutoAdvance:    stage.AutoAdvance,
			StageApprovals: make(map[string]Decision),
		}

		// Initialize each approver with pending status
		for _, approver := range stage.Approvers {
			initializedStages[i].StageApprovals[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return initializedStages
}

// createInitialApprovalsMap creates the document-level approvals map for workflow tracking
func (s *SmartContract) createInitialApprovalsMap(stages []WorkflowStage, timestamp string) map[string]Decision {
	approvalsMap := make(map[string]Decision)

	for _, stage := range stages {
		for _, approver := range stage.Approvers {
			var key string
			if len(stages) == 1 {
				key = approver
			} else {
				key = fmt.Sprintf("#%d %s", stage.StageNumber, approver)
			}

			approvalsMap[key] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return approvalsMap
}

// extractAllApprovers extracts a unique list of all approvers across all stages
func (s *SmartContract) extractAllApprovers(stages []WorkflowStage) []string {
	approversMap := make(map[string]bool)

	for _, stage := range stages {
		for _, approver := range stage.Approvers {
			approversMap[approver] = true
		}
	}

	var approvers []string
	for approver := range approversMap {
		approvers = append(approvers, approver)
	}
	return approvers
}

// isDocumentFullyApproved checks if document is in fully approved state
func (s *SmartContract) isDocumentFullyApproved(doc Document) bool {
	if !doc.Workflow.Enabled {
		return false
	}

	// Check if all stages are completed
	totalStages := len(doc.Workflow.Stages)
	if len(doc.Workflow.CompletedStages) == totalStages {
		return true
	}

	return false
}