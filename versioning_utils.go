package chaincode

import (
	"fmt"
	"sort"
	"time"
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

	// Stage updates change detection
	if len(newState.StageUpdates) > 0 {
		// Extract all current approvers from all stages
		currentApprovers := s.extractAllApprovers(current.Workflow.Stages)

		// DETERMINISM FIX: Sort stage keys to ensure consistent processing order across all peers
		var sortedStageKeys []string
		for stageKey := range newState.StageUpdates {
			sortedStageKeys = append(sortedStageKeys, stageKey)
		}
		sort.Strings(sortedStageKeys)

		// Extract all new approvers from stage updates (in deterministic order)
		var newApprovers []string
		for _, stageKey := range sortedStageKeys {
			stageUpdate := newState.StageUpdates[stageKey]
			newApprovers = append(newApprovers, stageUpdate.Approvers...)
		}

		// Remove duplicates for accurate comparison
		newApprovers = s.removeDuplicates(newApprovers)

		changes.ApproversAdded = s.findAdded(currentApprovers, newApprovers)
		changes.ApproversRemoved = s.findRemoved(currentApprovers, newApprovers)
	}

	// Decisions change detection - compare against the last version's ValidDecisions, not current document state
	if len(newState.ValidDecisions) > 0 {
		var previousValidDecisions []string
		if len(current.Versions) > 0 {
			// Use the last version's ValidDecisions for comparison
			previousValidDecisions = current.Versions[len(current.Versions)-1].ValidDecisions
		} else {
			// Use current document ValidDecisions if no versions exist yet
			previousValidDecisions = current.ValidDecisions
		}
		changes.DecisionsAdded = s.findAdded(previousValidDecisions, newState.ValidDecisions)
		changes.DecisionsRemoved = s.findRemoved(previousValidDecisions, newState.ValidDecisions)
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
func (s *SmartContract) validateReconsiderationAttempt(doc *Document, approver string) error {
	if len(doc.Versions) < 2 {
		return nil // No reconsideration rules for first version
	}

	// Get the approver's previous decision from the compatibility map
	if previousDecision, exists := doc.ApprovalsMap[approver]; exists && previousDecision.Status != Pending {
		// This is a reconsideration attempt

		// Get the last version changes to determine reconsideration rules
		lastVersion := doc.Versions[len(doc.Versions)-1]

		// CRITICAL FIX: Check timestamp validity for reconsideration
		// Approver can only reconsider if their last decision was made BEFORE the current version was created
		approverTime, err := time.Parse(time.RFC3339, previousDecision.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid approver timestamp format: %v", err)
		}

		versionTime, err := time.Parse(time.RFC3339, lastVersion.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid version timestamp format: %v", err)
		}

		// If approver's decision was made AFTER the current version was created,
		// then they already had a chance to reconsider and cannot do it again
		if approverTime.After(versionTime) {
			return fmt.Errorf("reconsideration not allowed - you already made a decision for version %d at %s (after version creation at %s)",
				doc.LatestVersion, previousDecision.Timestamp, lastVersion.Timestamp)
		}

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

		// PRIORITY RULE: Ownership transfers - no reconsideration allowed
		if lastVersion.ChangeType == "ownership" {
			return fmt.Errorf("reconsideration not allowed - ownership transfer in version %d does not require approver reconsideration", doc.LatestVersion)
		}

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

// removeDuplicates removes duplicate strings from a slice
func (s *SmartContract) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}