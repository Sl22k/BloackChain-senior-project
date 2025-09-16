package chaincode

import (
	"fmt"
)

// ensureDeadlineArraysInitialized ensures deadline status arrays are initialized
func (s *SmartContract) ensureDeadlineArraysInitialized(status DeadlineStatus) DeadlineStatus {
	if status.StageStatuses == nil {
		status.StageStatuses = make(map[string]StageDeadlineStatus)
	}
	if status.WarningsTriggered == nil {
		status.WarningsTriggered = []DeadlineWarning{}
	}
	if status.BreachesRecorded == nil {
		status.BreachesRecorded = []DeadlineBreach{}
	}
	return status
}

// deepCopyWorkflowConfig creates a deep copy of WorkflowConfig
func (s *SmartContract) deepCopyWorkflowConfig(src WorkflowConfig) WorkflowConfig {
	dst := WorkflowConfig{
		Enabled:         src.Enabled,
		CurrentStage:    src.CurrentStage,
		CompletedStages: make([]int, len(src.CompletedStages)),
		StageHistory:    make([]StageTransition, len(src.StageHistory)),
		Stages:          make([]WorkflowStage, len(src.Stages)),
	}

	// Copy completed stages
	copy(dst.CompletedStages, src.CompletedStages)

	// Copy stage history
	copy(dst.StageHistory, src.StageHistory)

	// Deep copy stages
	for i, stage := range src.Stages {
		dst.Stages[i] = WorkflowStage{
			StageNumber:    stage.StageNumber,
			StageName:      stage.StageName,
			Approvers:      make([]string, len(stage.Approvers)),
			RequiredCount:  stage.RequiredCount,
			AutoAdvance:    stage.AutoAdvance,
			StageApprovals: s.copyApprovalsMap(stage.StageApprovals),
		}
		copy(dst.Stages[i].Approvers, stage.Approvers)
	}

	return dst
}

// copyApprovalsMap creates a deep copy of approvals map for version snapshots
func (s *SmartContract) copyApprovalsMap(source map[string]Decision) map[string]Decision {
	if source == nil {
		return make(map[string]Decision)
	}

	result := make(map[string]Decision)
	for key, decision := range source {
		result[key] = Decision{
			Status:    decision.Status,
			Comment:   decision.Comment,
			Timestamp: decision.Timestamp,
		}
	}
	return result
}

// createWorkflowSnapshot creates a snapshot of current workflow state for version history
func (s *SmartContract) createWorkflowSnapshot(workflow WorkflowConfig, timestamp string) *WorkflowSnapshot {
	snapshot := &WorkflowSnapshot{
		Enabled:             workflow.Enabled,
		CurrentStage:        workflow.CurrentStage,
		CompletedStages:     make([]int, len(workflow.CompletedStages)),
		TotalStages:         len(workflow.Stages),
		CompletionTimestamp: timestamp,
	}

	// Copy completed stages
	copy(snapshot.CompletedStages, workflow.CompletedStages)

	return snapshot
}

// calculateStageProgress calculates progress for a specific workflow stage
func (s *SmartContract) calculateStageProgress(stage WorkflowStage, isCurrentStage, isCompleted bool) StageProgressSummary {
	approvedCount := 0
	rejectedCount := 0
	pendingCount := 0

	// Count decisions by status
	for _, decision := range stage.StageApprovals {
		switch decision.Status {
		case Approved:
			approvedCount++
		case Rejected:
			rejectedCount++
		case Pending:
			pendingCount++
		}
	}

	// Determine required count (0 = all approvers needed)
	requiredCount := stage.RequiredCount
	if requiredCount <= 0 {
		requiredCount = len(stage.Approvers)
	}

	// Calculate status
	status := "pending"
	if isCompleted {
		status = "completed"
	} else if rejectedCount > 0 {
		status = "rejected"
	} else if approvedCount >= requiredCount {
		status = "approved"
	} else if isCurrentStage {
		status = "in_progress"
	}

	// Calculate completion percentage
	completionPercent := 0.0
	if requiredCount > 0 {
		completionPercent = float64(approvedCount) / float64(requiredCount) * 100
		if completionPercent > 100 {
			completionPercent = 100
		}
	}

	// Get pending approvers (initialize as empty array to avoid null in JSON)
	pendingApprovers := make([]string, 0)
	for _, approver := range stage.Approvers {
		if decision, exists := stage.StageApprovals[approver]; exists && decision.Status == Pending {
			pendingApprovers = append(pendingApprovers, approver)
		}
	}

	return StageProgressSummary{
		StageNumber:       stage.StageNumber,
		StageName:         stage.StageName,
		Status:            status,
		RequiredCount:     requiredCount,
		ApprovedCount:     approvedCount,
		RejectedCount:     rejectedCount,
		PendingCount:      pendingCount,
		CompletionPercent: completionPercent,
		Approvers:         stage.Approvers,
		PendingApprovers:  pendingApprovers,
	}
}

// calculateWorkflowProgress calculates overall workflow progress
func (s *SmartContract) calculateWorkflowProgress(workflow WorkflowConfig) *WorkflowProgressOverview {
	if !workflow.Enabled || len(workflow.Stages) == 0 {
		return &WorkflowProgressOverview{
			CurrentStageNumber:   0,
			CurrentStageName:     "Not configured",
			CurrentStageProgress: 0,
			OverallProgress:      0,
			CompletedStages:      0,
			TotalStages:          0,
			StatusSummary:        "not_configured",
			NextAction:           "Configure workflow to enable progress tracking",
			StageDetails:         []StageProgressSummary{},
		}
	}

	var stageDetails []StageProgressSummary
	completedStagesMap := make(map[int]bool)
	for _, stageNum := range workflow.CompletedStages {
		completedStagesMap[stageNum] = true
	}

	for i, stage := range workflow.Stages {
		isCurrentStage := (i + 1) == workflow.CurrentStage
		isCompleted := completedStagesMap[stage.StageNumber]

		stageSummary := s.calculateStageProgress(stage, isCurrentStage, isCompleted)
		stageDetails = append(stageDetails, stageSummary)
	}

	// Calculate overall status and next action
	statusSummary := "in_progress"
	var nextAction string
	var currentStageName string
	var currentStageProgress float64

	if len(workflow.CompletedStages) == len(workflow.Stages) {
		statusSummary = "completed"
		nextAction = "All stages completed"
		currentStageName = "Completed"
		currentStageProgress = 100
	} else if workflow.CurrentStage <= len(workflow.Stages) {
		currentStage := workflow.Stages[workflow.CurrentStage-1]
		stageSummary := stageDetails[workflow.CurrentStage-1]
		currentStageName = currentStage.StageName
		currentStageProgress = stageSummary.CompletionPercent

		if stageSummary.RejectedCount > 0 {
			statusSummary = "rejected"
			nextAction = fmt.Sprintf("Document rejected in %s", currentStage.StageName)
		} else if stageSummary.ApprovedCount >= stageSummary.RequiredCount {
			if currentStage.AutoAdvance {
				nextAction = "Stage completed - auto-advancing"
			} else {
				nextAction = "Stage completed - manual advance needed"
			}
		} else {
			nextAction = fmt.Sprintf("Waiting for %d approvers in %s", stageSummary.PendingCount, currentStage.StageName)
		}
	}

	// Calculate overall progress percentage
	overallProgress := float64(len(workflow.CompletedStages)) / float64(len(workflow.Stages)) * 100

	return &WorkflowProgressOverview{
		CurrentStageNumber:   workflow.CurrentStage,
		CurrentStageName:     currentStageName,
		CurrentStageProgress: currentStageProgress,
		OverallProgress:      overallProgress,
		CompletedStages:      len(workflow.CompletedStages),
		TotalStages:          len(workflow.Stages),
		StatusSummary:        statusSummary,
		NextAction:           nextAction,
		StageDetails:         stageDetails,
	}
}