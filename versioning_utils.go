package chaincode

import (
	"fmt"
)

// detectChanges compares current document state with desired new state
func (s *SmartContract) detectChanges(current Document, newState NewVersionState) DetectedChanges {
	changes := DetectedChanges{}

	// Content change detection
	if newState.ContentHash != "" {
		latestVersion := current.Versions[len(current.Versions)-1]
		if newState.ContentHash != latestVersion.Hash {
			changes.ContentChanged = true
		}
	}

	// Approvers change detection
	if len(newState.ApproversList) > 0 {
		currentApprovers := s.extractAllApprovers(current.Workflow.Stages)
		changes.ApproversAdded = s.findAdded(currentApprovers, newState.ApproversList)
		changes.ApproversRemoved = s.findRemoved(currentApprovers, newState.ApproversList)
	}

	// Decisions change detection
	if len(newState.ValidDecisions) > 0 {
		changes.DecisionsAdded = s.findAdded(current.ValidDecisions, newState.ValidDecisions)
		changes.DecisionsRemoved = s.findRemoved(current.ValidDecisions, newState.ValidDecisions)
	}

	// Calculate reconsideration rules based on detected changes
	changes = s.calculateReconsiderationRules(changes)

	return changes
}

// calculateReconsiderationRules determines what reconsideration options are available
func (s *SmartContract) calculateReconsiderationRules(changes DetectedChanges) DetectedChanges {
	// Rule 1: Only approver list changed = No reconsideration
	if !changes.ContentChanged && len(changes.DecisionsAdded) == 0 && len(changes.DecisionsRemoved) == 0 &&
		(len(changes.ApproversAdded) > 0 || len(changes.ApproversRemoved) > 0) {
		changes.AllowReconsideration = false
		return changes
	}

	// Rule 2: Content changed = Full reconsideration allowed
	if changes.ContentChanged {
		changes.AllowReconsideration = true
		return changes
	}

	// Rule 3: Only decisions changed = Limited reconsideration (to new decisions only)
	if !changes.ContentChanged && (len(changes.DecisionsAdded) > 0 || len(changes.DecisionsRemoved) > 0) {
		changes.AllowReconsideration = true
		changes.NewDecisionsOnly = changes.DecisionsAdded // Can only reconsider to newly added decisions
		return changes
	}

	// Default: Allow reconsideration
	changes.AllowReconsideration = true
	return changes
}

// findAdded returns items in newList that are not in currentList
func (s *SmartContract) findAdded(currentList, newList []string) []string {
	currentMap := make(map[string]bool)
	for _, item := range currentList {
		currentMap[item] = true
	}

	var added []string
	for _, item := range newList {
		if !currentMap[item] {
			added = append(added, item)
		}
	}
	return added
}

// findRemoved returns items in currentList that are not in newList
func (s *SmartContract) findRemoved(currentList, newList []string) []string {
	newMap := make(map[string]bool)
	for _, item := range newList {
		newMap[item] = true
	}

	var removed []string
	for _, item := range currentList {
		if !newMap[item] {
			removed = append(removed, item)
		}
	}
	return removed
}

// validateReconsiderationAttempt checks if approval is a reconsideration and validates rules
func (s *SmartContract) validateReconsiderationAttempt(doc *Document, approver, currentTimestamp string) error {
	if len(doc.Versions) < 2 {
		return nil // No reconsideration rules for first version
	}

	// Get the approver's previous decision from the compatibility map
	if previousDecision, exists := doc.ApprovalsMap[approver]; exists && previousDecision.Status != Pending {
		// This is a reconsideration attempt

		// Get the last version changes to determine reconsideration rules
		lastVersion := doc.Versions[len(doc.Versions)-1]

		// Determine what changed in the last version based on flags
		contentChanged := false
		approversChanged := lastVersion.ApproversChanged
		decisionsChanged := lastVersion.DecisionsChanged

		// Check if content changed by comparing hashes between last two versions
		if len(doc.Versions) >= 2 {
			previousVersion := doc.Versions[len(doc.Versions)-2]
			contentChanged = lastVersion.Hash != previousVersion.Hash
		}

		// Apply reconsideration business rules
		if !contentChanged && !decisionsChanged && approversChanged {
			// Only approver list changed - no reconsideration allowed
			return fmt.Errorf("reconsideration not allowed - only approver list was modified in version %d", doc.LatestVersion)
		}

		if !contentChanged && decisionsChanged && !approversChanged {
			// Only decisions changed - limited reconsideration (to new decisions only)
			// Get the valid decisions from when the approver last decided
			approverLastDecisionTime := previousDecision.Timestamp

			// Find what decisions were available when approver last decided
			var availableDecisions []string
			for _, version := range doc.Versions {
				if version.Timestamp <= approverLastDecisionTime {
					availableDecisions = version.ValidDecisions
				} else {
					break // Found the version after approval, stop
				}
			}

			// Current decision must be from newly added decisions only
			currentDecisions := doc.ValidDecisions
			newDecisions := s.findAdded(availableDecisions, currentDecisions)

			if len(newDecisions) == 0 {
				return fmt.Errorf("reconsideration not allowed - no new decision options were added in version %d", doc.LatestVersion)
			}

			// Note: The actual decision validation against new decisions will be handled
			// in the main approval logic, this is just structural validation
		}

		// If content changed, full reconsideration is allowed (no additional restrictions)
	}

	return nil
}

// validateReconsiderationDecision validates decision choices for reconsideration cases
func (s *SmartContract) validateReconsiderationDecision(doc *Document, approver, decision string) error {
	if len(doc.Versions) < 2 {
		return nil // No additional validation for first version
	}

	// Check if this is a reconsideration (approver has previous decision)
	if previousDecision, exists := doc.ApprovalsMap[approver]; exists && previousDecision.Status != Pending {
		// Get the last version changes
		lastVersion := doc.Versions[len(doc.Versions)-1]

		// Check if only decisions changed (and not content or approvers)
		contentChanged := false
		if len(doc.Versions) >= 2 {
			previousVersion := doc.Versions[len(doc.Versions)-2]
			contentChanged = lastVersion.Hash != previousVersion.Hash
		}

		approversChanged := lastVersion.ApproversChanged
		decisionsChanged := lastVersion.DecisionsChanged

		if !contentChanged && !approversChanged && decisionsChanged {
			// Only decisions changed - restrict to new decisions only
			approverLastDecisionTime := previousDecision.Timestamp

			// Find what decisions were available when approver last decided
			var availableDecisions []string
			for _, version := range doc.Versions {
				if version.Timestamp <= approverLastDecisionTime {
					availableDecisions = version.ValidDecisions
				} else {
					break
				}
			}

			// Check if the current decision is from newly added decisions
			newDecisions := s.findAdded(availableDecisions, doc.ValidDecisions)

			isNewDecision := false
			for _, newDec := range newDecisions {
				if decision == newDec {
					isNewDecision = true
					break
				}
			}

			if !isNewDecision {
				return fmt.Errorf("reconsideration to '%s' not allowed - can only reconsider to newly added decisions: %v", decision, newDecisions)
			}
		}
	}

	return nil
}