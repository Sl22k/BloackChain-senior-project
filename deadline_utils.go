package chaincode

import (
	"fmt"
	"strconv"
	"time"
)

// calculateDeadlineFromHours converts hours to RFC3339 deadline timestamp
func calculateDeadlineFromHours(hours int) string {
	return time.Now().Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)
}

// calculateDeadlineStatus determines deadline status and remaining hours
func calculateDeadlineStatus(deadlineRFC3339 string, warningThreshold int) (string, int64, error) {
	deadline, err := time.Parse(time.RFC3339, deadlineRFC3339)
	if err != nil {
		return "", 0, fmt.Errorf("invalid deadline format: %v", err)
	}

	hoursRemaining := int64(time.Until(deadline).Hours())

	status := "active"
	if hoursRemaining < 0 {
		status = "breached"
	} else if hoursRemaining <= int64(warningThreshold) {
		status = "warning"
	}

	return status, hoursRemaining, nil
}

// parseDeadlineString safely converts RFC3339 deadline string to time
func parseDeadlineString(deadlineRFC3339 string) (time.Time, error) {
	return time.Parse(time.RFC3339, deadlineRFC3339)
}

// validateDeadlineHours ensures deadline hours input is valid
func validateDeadlineHours(deadlineHours string) (int, error) {
	hours, err := strconv.Atoi(deadlineHours)
	if err != nil || hours <= 0 {
		return 0, fmt.Errorf("invalid deadline hours: %s", deadlineHours)
	}
	return hours, nil
}

// createDefaultDeadlineConfig creates a default deadline configuration
func createDefaultDeadlineConfig(enabled bool) DeadlineConfig {
	return DeadlineConfig{
		Enabled:           enabled,
		StageDeadlines:    make(map[string]string),
		WarningThreshold:  24, // Default 24 hour warning
		AutoReject:        false,
		EscalationEnabled: false,
		EscalationTargets: []string{},
		AutomationHooks:   []AutomationHook{},
	}
}

// createDefaultDeadlineStatus creates a default deadline status
func createDefaultDeadlineStatus(documentId, status, deadlineRFC3339 string, hoursRemaining int64) DeadlineStatus {
	return DeadlineStatus{
		DocumentId:           documentId,
		CurrentStageStatus:   status,
		CurrentStageDeadline: deadlineRFC3339,
		TimeRemaining:        hoursRemaining,
		StageStatuses:        make(map[string]StageDeadlineStatus),
		WarningsTriggered:    []DeadlineWarning{},
		BreachesRecorded:     []DeadlineBreach{},
	}
}

// isDeadlineInRange checks if deadline is within specified hours ahead
func isDeadlineInRange(deadlineRFC3339 string, hoursAhead int) (bool, float64, error) {
	deadline, err := parseDeadlineString(deadlineRFC3339)
	if err != nil {
		return false, 0, err
	}

	hoursRemaining := time.Until(deadline).Hours()
	return hoursRemaining >= 0 && hoursRemaining <= float64(hoursAhead), hoursRemaining, nil
}
