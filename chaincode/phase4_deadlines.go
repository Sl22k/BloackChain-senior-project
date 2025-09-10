package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Deadline configuration for documents or workflow stages
type DeadlineConfig struct {
	Enabled            bool                `json:"Enabled"`
	DocumentDeadline   string              `json:"DocumentDeadline"`
	StageDeadlines     map[string]string   `json:"StageDeadlines"`
	WarningThreshold   int                 `json:"WarningThreshold"`
	AutoReject         bool                `json:"AutoReject"`
	EscalationEnabled  bool                `json:"EscalationEnabled"`
	EscalationTargets  []string            `json:"EscalationTargets"`
	AutomationHooks    []AutomationHook    `json:"AutomationHooks"`
}

type AutomationHook struct {
	HookId      string                 `json:"HookId"`
	TriggerType string                 `json:"TriggerType"`
	Conditions  map[string]interface{} `json:"Conditions"`
	Actions     []AutomationAction     `json:"Actions"`
	Enabled     bool                   `json:"Enabled"`
}

type AutomationAction struct {
	ActionType string                 `json:"ActionType"`
	Parameters map[string]interface{} `json:"Parameters"`
}

type DeadlineStatus struct {
	DocumentId         string                         `json:"DocumentId"`
	OverallStatus      string                         `json:"OverallStatus"`
	DocumentDeadline   string                         `json:"DocumentDeadline"`
	TimeRemaining      int64                          `json:"TimeRemaining"`
	StageStatuses      map[string]StageDeadlineStatus `json:"StageStatuses"`
	WarningsTriggered  []DeadlineWarning              `json:"WarningsTriggered"`
	BreachesRecorded   []DeadlineBreach               `json:"BreachesRecorded"`
}

type StageDeadlineStatus struct {
	StageNumber    int    `json:"StageNumber"`
	Deadline       string `json:"Deadline"`
	Status         string `json:"Status"`
	TimeRemaining  int64  `json:"TimeRemaining"`
}

type DeadlineWarning struct {
	TriggeredAt    string `json:"TriggeredAt"`
	DeadlineType   string `json:"DeadlineType"`
	StageNumber    int    `json:"StageNumber,omitempty"`
	HoursRemaining int    `json:"HoursRemaining"`
}

type DeadlineBreach struct {
	BreachedAt     string   `json:"BreachedAt"`
	DeadlineType   string   `json:"DeadlineType"`
	StageNumber    int      `json:"StageNumber,omitempty"`
	HoursOverdue   int      `json:"HoursOverdue"`
	AutoActions    []string `json:"AutoActions"`
}

// SetDocumentDeadline sets or updates deadline for a document (simplified embedded version)
func (s *SmartContract) SetDocumentDeadline(ctx contractapi.TransactionContextInterface, id, invokerId, deadlineHours string) error {
	if err := s.validateInputs(map[string]string{
		"id": id, "invokerId": invokerId, "deadlineHours": deadlineHours,
	}); err != nil {
		return err
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, &doc) {
		return fmt.Errorf("only document owner or privileged editors can set deadlines")
	}

	hours, err := strconv.Atoi(deadlineHours)
	if err != nil || hours <= 0 {
		return fmt.Errorf("invalid deadline hours: %s", deadlineHours)
	}

	currentTime := time.Now()
	deadlineTime := currentTime.Add(time.Duration(hours) * time.Hour)
	// Convert to Asia/Riyadh timezone for human-readable format  
	location, _ := time.LoadLocation("Asia/Riyadh")
	if location == nil {
		location = time.FixedZone("AST", 3*3600)
	}
	deadlineTimeLocal := deadlineTime.In(location)
	deadlineTimestamp := deadlineTimeLocal.Format("2006-01-02 15:04:05")

	doc.DeadlineConfig = &DeadlineConfig{
		Enabled:          true,
		DocumentDeadline: deadlineTimestamp,
		StageDeadlines:   make(map[string]string),
		WarningThreshold: func() int { if hours/4 > 1 { return hours/4 } else { return 1 } }(),
		AutoReject:       false,
		EscalationEnabled: false,
		EscalationTargets: []string{},
		AutomationHooks:   []AutomationHook{},
	}

	hoursRemaining := int64(deadlineTime.Sub(currentTime).Hours())
	doc.DeadlineStatus = &DeadlineStatus{
		DocumentId:        id,
		OverallStatus:     "active",
		DocumentDeadline:  deadlineTimestamp,
		TimeRemaining:     hoursRemaining,
		StageStatuses:     make(map[string]StageDeadlineStatus),
		WarningsTriggered: []DeadlineWarning{},
		BreachesRecorded:  []DeadlineBreach{},
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	s.ensureDocumentArraysInitialized(&doc)

	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) GetDeadlineStatus(ctx contractapi.TransactionContextInterface, id string) (*DeadlineStatus, error) {
	if err := s.validateInputs(map[string]string{"id": id}); err != nil {
		return nil, err
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	if doc.DeadlineConfig == nil || !doc.DeadlineConfig.Enabled {
		return &DeadlineStatus{
			DocumentId: id, OverallStatus: "no_deadlines",
			StageStatuses: make(map[string]StageDeadlineStatus),
			WarningsTriggered: []DeadlineWarning{}, BreachesRecorded: []DeadlineBreach{},
		}, nil
	}

	currentTime := time.Now()
	deadline, _ := time.Parse("2006-01-02 15:04:05", doc.DeadlineConfig.DocumentDeadline)
	hoursRemaining := int64(deadline.Sub(currentTime).Hours())

	status := "active"
	if hoursRemaining < 0 {
		status = "breached"
	} else if hoursRemaining <= int64(doc.DeadlineConfig.WarningThreshold) {
		status = "warning"
	}

	result := &DeadlineStatus{
		DocumentId: id, OverallStatus: status, DocumentDeadline: doc.DeadlineConfig.DocumentDeadline,
		TimeRemaining: hoursRemaining, StageStatuses: make(map[string]StageDeadlineStatus),
		WarningsTriggered: []DeadlineWarning{}, BreachesRecorded: []DeadlineBreach{},
	}

	if doc.DeadlineStatus != nil {
		result.WarningsTriggered = doc.DeadlineStatus.WarningsTriggered
		result.BreachesRecorded = doc.DeadlineStatus.BreachesRecorded
	}
	return s.ensureDeadlineArraysInitialized(result), nil
}

func (s *SmartContract) GetDocumentsWithUpcomingDeadlines(ctx contractapi.TransactionContextInterface, hoursAhead string) ([]*Document, error) {
	hoursAheadInt, err := strconv.Atoi(hoursAhead)
	if err != nil {
		return nil, fmt.Errorf("invalid hoursAhead parameter: %v", err)
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var upcomingDeadlineDocs []*Document
	currentTime := time.Now()

	for _, doc := range allDocs {
		if doc.DeadlineConfig == nil || !doc.DeadlineConfig.Enabled {
			continue
		}

		deadline, err := time.Parse("2006-01-02 15:04:05", doc.DeadlineConfig.DocumentDeadline)
		if err != nil {
			continue
		}

		hoursRemaining := deadline.Sub(currentTime).Hours()
		if hoursRemaining >= 0 && hoursRemaining <= float64(hoursAheadInt) {
			doc.DeadlineStatus = &DeadlineStatus{
				DocumentId: doc.ID,
				OverallStatus: func() string {
					if hoursRemaining <= 0 { return "breached" }
					if int64(hoursRemaining) <= int64(doc.DeadlineConfig.WarningThreshold) { return "warning" }
					return "active"
				}(),
				DocumentDeadline: doc.DeadlineConfig.DocumentDeadline, TimeRemaining: int64(hoursRemaining),
				StageStatuses: make(map[string]StageDeadlineStatus),
				WarningsTriggered: []DeadlineWarning{}, BreachesRecorded: []DeadlineBreach{},
			}
			s.ensureDeadlineArraysInitialized(doc.DeadlineStatus)
			upcomingDeadlineDocs = append(upcomingDeadlineDocs, doc)
		}
	}
	return upcomingDeadlineDocs, nil
}

func (s *SmartContract) RemoveDocumentDeadline(ctx contractapi.TransactionContextInterface, id, invokerId string) error {
	if err := s.validateInputs(map[string]string{"id": id, "invokerId": invokerId}); err != nil {
		return err
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	if !s.hasPrivilegedAccess(invokerId, &doc) {
		return fmt.Errorf("only document owner or privileged editors can remove deadlines")
	}

	// Reset to default disabled state instead of nil to maintain schema compliance
	doc.DeadlineConfig = &DeadlineConfig{
		Enabled:          false,
		DocumentDeadline: "",
		StageDeadlines:   make(map[string]string),
		WarningThreshold: 0,
		AutoReject:       false,
		EscalationEnabled: false,
		EscalationTargets: []string{},
		AutomationHooks:  []AutomationHook{},
	}
	doc.DeadlineStatus = &DeadlineStatus{
		DocumentId:        id,
		OverallStatus:     "not_configured",
		DocumentDeadline:  "",
		TimeRemaining:     0,
		StageStatuses:     make(map[string]StageDeadlineStatus),
		WarningsTriggered: []DeadlineWarning{},
		BreachesRecorded:  []DeadlineBreach{},
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	s.ensureDocumentArraysInitialized(&doc)

	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) ensureDeadlineArraysInitialized(status *DeadlineStatus) *DeadlineStatus {
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
