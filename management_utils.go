package chaincode

import (
	"fmt"
	"sort"
)

// addToUserList adds a user to a list if not already present
func addToUserList(user string, list []string) []string {
	for _, listUser := range list {
		if listUser == user {
			return list
		}
	}
	return append(list, user)
}

// removeFromUserList removes a user from a list
func removeFromUserList(userToRemove string, list []string) []string {
	var newList []string
	for _, user := range list {
		if user != userToRemove {
			newList = append(newList, user)
		}
	}
	return newList
}

// validateEditorManagement checks common editor management rules
func (s *SmartContract) validateEditorManagement(invokerId string, doc *Document) error {
	if !s.hasPrivilegedAccess(invokerId, doc) {
		return fmt.Errorf("user %s does not have privileged access to document %s", invokerId, doc.ID)
	}
	return nil
}

// validatePrivilegedEditorManagement checks privileged editor rules (stricter)
func (s *SmartContract) validatePrivilegedEditorManagement(invokerId string, doc *Document, isRemoval bool) error {
	if isRemoval {
		// Only document owner can remove privileged editors
		if invokerId != doc.Uploader {
			return fmt.Errorf("only the document owner %s can remove privileged editors", doc.Uploader)
		}
	} else {
		// Anyone with privileged access can add privileged editors
		if !s.hasPrivilegedAccess(invokerId, doc) {
			return fmt.Errorf("user %s does not have privileged access to document %s", invokerId, doc.ID)
		}
	}
	return nil
}

// createDeterministicApproversList creates a sorted list of approvers for consensus
func createDeterministicApproversList(approvalsMap map[string]Decision) []string {
	approvers := make([]string, 0, len(approvalsMap))
	for approver := range approvalsMap {
		approvers = append(approvers, approver)
	}
	sort.Strings(approvers) // CRITICAL: Ensure deterministic order for consensus
	return approvers
}

// rebuildApprovalsMap rebuilds the approvals map preserving existing decisions
func rebuildApprovalsMap(oldApprovalsMap map[string]Decision, newApprovers []string, timestamp string) map[string]Decision {
	updatedApprovalsMap := make(map[string]Decision)

	for _, approver := range newApprovers {
		if existingDecision, exists := oldApprovalsMap[approver]; exists {
			// Keep existing decision
			updatedApprovalsMap[approver] = existingDecision
		} else {
			// New approver - set to pending
			updatedApprovalsMap[approver] = Decision{
				Status:    Pending,
				Comment:   "",
				Timestamp: timestamp,
			}
		}
	}

	return updatedApprovalsMap
}

// synchronizeWorkflowApprovers updates workflow stage approvers to match document approvers
func synchronizeWorkflowApprovers(doc *Document, newApprovers []string) {
	if !doc.Workflow.Enabled || len(doc.Workflow.Stages) == 0 {
		return
	}

	if len(doc.Workflow.Stages) == 1 {
		// Single-stage workflow: update the only stage's approvers
		doc.Workflow.Stages[0].Approvers = newApprovers
	} else {
		// Multi-stage workflow: update current stage's approvers
		currentStageIndex := doc.Workflow.CurrentStage - 1
		if currentStageIndex >= 0 && currentStageIndex < len(doc.Workflow.Stages) {
			doc.Workflow.Stages[currentStageIndex].Approvers = newApprovers
		}
	}
}