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

type ApproverStatus string

const (
	Pending  ApproverStatus = "PENDING"
	Approved ApproverStatus = "APPROVED"
	Rejected ApproverStatus = "REJECTED"
)

type Decision struct {
	Status  ApproverStatus `json:"Status"`
	Comment string         `json:"Comment"`
	Date    string         `json:"Date"`
	Time    string         `json:"Time"`
}

type DocumentVersion struct {
	Version          int                 `json:"Version"`
	Hash             string              `json:"Hash"`
	Submitter        string              `json:"Submitter"`
	Date             string              `json:"Date"`
	Time             string              `json:"Time"`
	ApprovalsMap     map[string]Decision `json:"ApprovalsMap"`     // Approvals state at this version
	ValidDecisions   []string            `json:"ValidDecisions"`   // Valid decisions at this version
	ApproversChanged bool                `json:"ApproversChanged"` // Flag if approvers were modified
	DecisionsChanged bool                `json:"DecisionsChanged"` // Flag if valid decisions were modified
	// NEW: Version-specific workflow state - FIX FOR BUG #2
	WorkflowSnapshot *WorkflowSnapshot `json:"WorkflowSnapshot,omitempty"` // Workflow state snapshot
}

// Workflow stage configuration
type WorkflowStage struct {
	StageNumber    int                 `json:"StageNumber"`
	StageName      string              `json:"StageName"`
	RequiredCount  int                 `json:"RequiredCount"`  // Number of approvals needed to pass stage
	Approvers      []string            `json:"Approvers"`      // Who can approve at this stage
	StageApprovals map[string]Decision `json:"StageApprovals"` // Stage-specific approvals - FIX FOR BUG #1
	AutoAdvance    bool                `json:"AutoAdvance"`    // Auto advance to next stage when requirements met
	Description    string              `json:"Description"`    // Stage description
}

type WorkflowConfig struct {
	Enabled         bool              `json:"Enabled"`
	Stages          []WorkflowStage   `json:"Stages"`
	CurrentStage    int               `json:"CurrentStage"`
	CompletedStages []int             `json:"CompletedStages"`
	StageHistory    []StageTransition `json:"StageHistory"`
}

type StageTransition struct {
	FromStage  int    `json:"FromStage"`
	ToStage    int    `json:"ToStage"`
	Transition string `json:"Transition"` // "advanced", "returned", "rejected", "version_reset"
	Actor      string `json:"Actor"`
	Date       string `json:"Date"`
	Time       string `json:"Time"`
	Comment    string `json:"Comment"`
}

// NEW: Workflow snapshot for version history - FIX FOR BUG #2
type WorkflowSnapshot struct {
	Enabled         bool   `json:"Enabled"`
	CurrentStage    int    `json:"CurrentStage"`
	CompletedStages []int  `json:"CompletedStages"`
	TotalStages     int    `json:"TotalStages"`
	CompletionDate  string `json:"CompletionDate,omitempty"`
	CompletionTime  string `json:"CompletionTime,omitempty"`
}

// NEW: Workflow history response for audit queries
type WorkflowHistoryResponse struct {
	DocumentID       string                       `json:"DocumentID"`
	VersionCount     int                          `json:"VersionCount"`
	CurrentVersion   int                          `json:"CurrentVersion"`
	VersionSnapshots map[string]*WorkflowSnapshot `json:"VersionSnapshots"`
	VersionWorkflows map[string]*WorkflowConfig   `json:"VersionWorkflows"`
	CurrentWorkflow  *WorkflowConfig              `json:"CurrentWorkflow"`
}

// NEW: Stage progress summary for user-friendly UI
type StageProgressSummary struct {
	StageNumber       int      `json:"StageNumber"`
	StageName         string   `json:"StageName"`
	Status            string   `json:"Status"` // "pending", "in_progress", "completed", "rejected"
	RequiredCount     int      `json:"RequiredCount"`
	ApprovedCount     int      `json:"ApprovedCount"`
	RejectedCount     int      `json:"RejectedCount"`
	PendingCount      int      `json:"PendingCount"`
	CompletionPercent float64  `json:"CompletionPercent"`
	Approvers         []string `json:"Approvers"`
	PendingApprovers  []string `json:"PendingApprovers"`
}

// NEW: Workflow progress overview
type WorkflowProgressOverview struct {
	CurrentStageNumber   int                    `json:"CurrentStageNumber"`
	CurrentStageName     string                 `json:"CurrentStageName"`
	CurrentStageProgress float64                `json:"CurrentStageProgress"`
	OverallProgress      float64                `json:"OverallProgress"`
	CompletedStages      int                    `json:"CompletedStages"`
	TotalStages          int                    `json:"TotalStages"`
	StatusSummary        string                 `json:"StatusSummary"`
	NextAction           string                 `json:"NextAction"`
	StageDetails         []StageProgressSummary `json:"StageDetails"`
}

type Document struct {
	ID                string              `json:"ID"`
	Title             string              `json:"Title"`       // Document title/name
	Description       string              `json:"Description"` // Document description/purpose
	LatestVersion     int                 `json:"LatestVersion"`
	Versions          []DocumentVersion   `json:"Versions"`
	Uploader          string              `json:"Uploader"` // Document owner with highest privileges
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"` // Compatibility layer - shows latest decisions
	ValidDecisions    []string            `json:"ValidDecisions"`
	Workflow          WorkflowConfig      `json:"Workflow"` // Current active workflow configuration
	// NEW: Version workflow history - FIX FOR BUG #2
	VersionWorkflows map[string]*WorkflowConfig `json:"VersionWorkflows"`         // Complete workflow state per version
	DeadlineConfig   *DeadlineConfig            `json:"DeadlineConfig,omitempty"` // Document deadline configuration
	DeadlineStatus   *DeadlineStatus            `json:"DeadlineStatus,omitempty"` // Real-time deadline status
	CreatedDate      string                     `json:"CreatedDate"`
	CreatedTime      string                     `json:"CreatedTime"`
	LastModifiedDate string                     `json:"LastModifiedDate"`
	LastModifiedTime string                     `json:"LastModifiedTime"`
}

type StatusResponse struct {
	ID                string              `json:"ID"`
	Title             string              `json:"Title"`
	Description       string              `json:"Description"`
	LatestVersion     int                 `json:"LatestVersion"`
	Versions          []DocumentVersion   `json:"Versions"`
	Uploader          string              `json:"Uploader"`
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"`
	DecisionCounts    map[string]int      `json:"DecisionCounts"`
	ValidDecisions    []string            `json:"ValidDecisions"`
	Workflow          WorkflowConfig      `json:"Workflow"`
	CurrentStageInfo  *WorkflowStage      `json:"CurrentStageInfo,omitempty"`
	// NEW: Version workflow history - FIX FOR BUG #2
	VersionWorkflows map[string]*WorkflowConfig `json:"VersionWorkflows"`
	// NEW: User-friendly workflow progress information
	WorkflowProgress *WorkflowProgressOverview `json:"WorkflowProgress,omitempty"`
	DeadlineConfig   *DeadlineConfig           `json:"DeadlineConfig,omitempty"`
	DeadlineStatus   *DeadlineStatus           `json:"DeadlineStatus,omitempty"`
	CreatedDate      string                    `json:"CreatedDate"`
	CreatedTime      string                    `json:"CreatedTime"`
	LastModifiedDate string                    `json:"LastModifiedDate"`
	LastModifiedTime string                    `json:"LastModifiedTime"`
}

type EventPayload struct {
	DocumentID  string                 `json:"documentID"`
	Version     int                    `json:"version"`
	EventType   string                 `json:"eventType"`
	Actor       string                 `json:"actor"`
	Date        string                 `json:"date"`
	Time        string                 `json:"time"`
	Details     map[string]interface{} `json:"details"`
	Description string                 `json:"description"`
}

type SmartContract struct {
	contractapi.Contract
}

// Helper function to get consistent timestamps with local timezone
func (s *SmartContract) getCurrentDateTime(ctx contractapi.TransactionContextInterface) (string, string, error) {
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	var currentTime time.Time
	if err != nil {
		// Fallback to current time if tx timestamp fails
		currentTime = time.Now()
	} else {
		currentTime = time.Unix(txTimestamp.GetSeconds(), 0)
	}

	// Convert to Asia/Riyadh timezone (+03)
	location, err := time.LoadLocation("Asia/Riyadh")
	if err != nil {
		// Fallback to UTC+3 if timezone loading fails
		location = time.FixedZone("AST", 3*3600) // +3 hours
	}
	localTime := currentTime.In(location)
	date := localTime.Format("2006-01-02")
	timeStr := localTime.Format("15:04:05")
	return date, timeStr, nil
}

// Helper function to emit standardized events
func (s *SmartContract) emitEvent(ctx contractapi.TransactionContextInterface, eventType, actor, description string, documentID string, version int, details map[string]interface{}) error {
	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %v", err)
	}

	payload := EventPayload{
		DocumentID:  documentID,
		Version:     version,
		EventType:   eventType,
		Actor:       actor,
		Date:        date,
		Time:        timeStr,
		Details:     details,
		Description: description,
	}

	eventBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	return ctx.GetStub().SetEvent("DocumentUpdated", eventBytes)
}

// Helper function to validate input parameters
func (s *SmartContract) validateInputs(params map[string]string) error {
	// Enhanced validation with specific error messages
	fieldNames := map[string]string{
		"id":          "Document ID",
		"title":       "Document Title",
		"description": "Document Description",
		"hash":        "Document Hash",
		"uploader":    "Uploader Identity",
		"approvers":   "Approvers List",
		"decisions":   "Valid Decisions List",
		"approver":    "Approver Identity",
		"decision":    "Decision",
		"documentID":  "Document ID",
		"stageNumber": "Stage Number",
		"invokerID":   "Invoker Identity",
	}

	for key, value := range params {
		if value == "" {
			friendlyName := fieldNames[key]
			if friendlyName == "" {
				friendlyName = key
			}
			return fmt.Errorf("❌ %s cannot be empty. Please provide a valid %s", friendlyName, friendlyName)
		}

		// Additional validation rules
		switch key {
		case "id", "documentID":
			if len(value) < 3 {
				return fmt.Errorf("❌ Document ID must be at least 3 characters long")
			}
			if len(value) > 50 {
				return fmt.Errorf("❌ Document ID must be no more than 50 characters long")
			}
		case "title":
			if len(value) > 100 {
				return fmt.Errorf("❌ Document Title must be no more than 100 characters long")
			}
		case "description":
			if len(value) > 500 {
				return fmt.Errorf("❌ Document Description must be no more than 500 characters long")
			}
		case "hash":
			if len(value) < 8 {
				return fmt.Errorf("❌ Document Hash must be at least 8 characters long")
			}
		}
	}
	return nil
}

// Helper function to ensure workflow arrays are properly initialized (never null)
func (s *SmartContract) ensureWorkflowArraysInitialized(workflow *WorkflowConfig) {
	if workflow.CompletedStages == nil {
		workflow.CompletedStages = []int{}
	}
	if workflow.StageHistory == nil {
		workflow.StageHistory = []StageTransition{}
	}
	if workflow.Stages == nil {
		workflow.Stages = []WorkflowStage{}
	}

	// Also ensure each stage has proper array initialization
	for i := range workflow.Stages {
		if workflow.Stages[i].Approvers == nil {
			workflow.Stages[i].Approvers = []string{}
		}
	}
}

// Helper function to ensure ALL document arrays and objects are properly initialized (never null)
func (s *SmartContract) ensureDocumentArraysInitialized(doc *Document) {
	// Initialize arrays
	if doc.Editors == nil {
		doc.Editors = []string{}
	}
	if doc.PrivilegedEditors == nil {
		doc.PrivilegedEditors = []string{}
	}
	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}
	if doc.Versions == nil {
		doc.Versions = []DocumentVersion{}
	}

	// Initialize ApprovalsMap
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}

	// Initialize workflow arrays
	s.ensureWorkflowArraysInitialized(&doc.Workflow)

	// SCHEMA FIX: Initialize VersionWorkflows map
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}

	// CRITICAL FIX: Ensure DeadlineConfig and DeadlineStatus are never null to meet schema requirements
	if doc.DeadlineConfig == nil {
		doc.DeadlineConfig = &DeadlineConfig{
			Enabled:           false,
			StageDeadlines:    make(map[string]string),
			EscalationTargets: []string{},
			AutomationHooks:   []AutomationHook{},
		}
	}
	if doc.DeadlineStatus == nil {
		doc.DeadlineStatus = &DeadlineStatus{
			DocumentId:        doc.ID,
			OverallStatus:     "not_configured",
			StageStatuses:     make(map[string]StageDeadlineStatus),
			WarningsTriggered: []DeadlineWarning{},
			BreachesRecorded:  []DeadlineBreach{},
		}
	}

	// Initialize deadline status arrays if deadline status exists
	if doc.DeadlineStatus != nil {
		s.ensureDeadlineArraysInitialized(doc.DeadlineStatus)
	}

	// Initialize each version's arrays and maps
	for i := range doc.Versions {
		if doc.Versions[i].ApprovalsMap == nil {
			doc.Versions[i].ApprovalsMap = make(map[string]Decision)
		}
		if doc.Versions[i].ValidDecisions == nil {
			doc.Versions[i].ValidDecisions = []string{}
		}
	}
}

// Helper function to check if user is in a list
func (s *SmartContract) isUserInList(user string, list []string) bool {
	for _, u := range list {
		if u == user {
			return true
		}
	}
	return false
}

// NEW: Helper functions for workflow bug fixes
func (s *SmartContract) deepCopyWorkflowConfig(src WorkflowConfig) WorkflowConfig {
	// Deep copy workflow configuration
	stages := make([]WorkflowStage, len(src.Stages))
	for i, stage := range src.Stages {
		// Copy stage approvals map
		stageApprovals := make(map[string]Decision)
		if stage.StageApprovals != nil {
			for k, v := range stage.StageApprovals {
				stageApprovals[k] = v
			}
		}

		// Copy approvers slice
		approvers := make([]string, len(stage.Approvers))
		copy(approvers, stage.Approvers)

		stages[i] = WorkflowStage{
			StageNumber:    stage.StageNumber,
			StageName:      stage.StageName,
			RequiredCount:  stage.RequiredCount,
			Approvers:      approvers,
			StageApprovals: stageApprovals,
			AutoAdvance:    stage.AutoAdvance,
			Description:    stage.Description,
		}
	}

	// Copy completed stages slice
	completedStages := make([]int, len(src.CompletedStages))
	copy(completedStages, src.CompletedStages)

	// Copy stage history
	stageHistory := make([]StageTransition, len(src.StageHistory))
	copy(stageHistory, src.StageHistory)

	return WorkflowConfig{
		Enabled:         src.Enabled,
		Stages:          stages,
		CurrentStage:    src.CurrentStage,
		CompletedStages: completedStages,
		StageHistory:    stageHistory,
	}
}

func (s *SmartContract) deepCopyApprovalsMap(src map[string]Decision) map[string]Decision {
	if src == nil {
		return make(map[string]Decision)
	}
	dst := make(map[string]Decision)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (s *SmartContract) initializeStageApprovals(stages []WorkflowStage, date, timeStr string) []WorkflowStage {
	// Initialize StageApprovals for all stages to prevent null errors
	for i := range stages {

		if stages[i].StageApprovals == nil {
			stages[i].StageApprovals = make(map[string]Decision)
		}
		// Initialize all approvers to PENDING for this stage
		for _, approver := range stages[i].Approvers {
			stages[i].StageApprovals[approver] = Decision{
				Status:  Pending,
				Comment: "",
				Date:    date,
				Time:    timeStr,
			}
		}
	}
	return stages
}

func (s *SmartContract) updateCompatibilityApprovalsMap(doc *Document) {
	// Handle workflow vs non-workflow documents differently
	if !doc.Workflow.Enabled || len(doc.Workflow.Stages) == 0 {
		// For non-workflow documents, preserve existing simple approver names unchanged
		if doc.ApprovalsMap == nil {
			doc.ApprovalsMap = make(map[string]Decision)
		}
		return
	}

	// For WORKFLOW documents: Use smart key format based on workflow complexity
	doc.ApprovalsMap = make(map[string]Decision)

	// Determine key format: simple workflow (1 stage) vs advanced workflow (multiple stages)
	isSimpleWorkflow := len(doc.Workflow.Stages) == 1

	// Show ONLY current stage approvers with appropriate key format
	currentStageIndex := doc.Workflow.CurrentStage - 1
	if currentStageIndex >= 0 && currentStageIndex < len(doc.Workflow.Stages) {
		currentStage := doc.Workflow.Stages[currentStageIndex]

		// Add current stage approvers using appropriate key format
		if currentStage.StageApprovals != nil {
			for approver, decision := range currentStage.StageApprovals {
				var key string
				if isSimpleWorkflow {
					// Simple workflow - use plain approver names (cleaner UX)
					key = approver
				} else {
					// Advanced workflow - use composite keys to distinguish stages
					key = fmt.Sprintf("#%d %s", doc.Workflow.CurrentStage, approver)
				}
				doc.ApprovalsMap[key] = decision
			}
		}
	}
}

// NEW: Build ALL composite keys for new version (for version reset)
func (s *SmartContract) buildAllCompositeKeysApprovalsMap(doc *Document, date, timeStr string) {
	if !doc.Workflow.Enabled || len(doc.Workflow.Stages) == 0 {
		return
	}

	// For new version reset: Show ALL stage approvers with composite keys as PENDING
	doc.ApprovalsMap = make(map[string]Decision)

	for _, stage := range doc.Workflow.Stages {
		for _, approver := range stage.Approvers {
			compositeKey := fmt.Sprintf("#%d %s", stage.StageNumber, approver)
			doc.ApprovalsMap[compositeKey] = Decision{
				Status:  Pending,
				Comment: "",
				Date:    date,
				Time:    timeStr,
			}
		}
	}
}

// syncCurrentApprovalsToLatestVersion - Syncs ALL stage approvals to the latest version ApprovalsMap
func (s *SmartContract) syncCurrentApprovalsToLatestVersion(doc *Document) {
	if doc.Versions == nil || len(doc.Versions) == 0 {
		return
	}

	// Find the latest version (typically the last one in the array)
	latestVersionIndex := -1
	for i, version := range doc.Versions {
		if version.Version == doc.LatestVersion {
			latestVersionIndex = i
			break
		}
	}

	if latestVersionIndex == -1 {
		return // Latest version not found
	}

	// Ensure version ApprovalsMap is initialized
	if doc.Versions[latestVersionIndex].ApprovalsMap == nil {
		doc.Versions[latestVersionIndex].ApprovalsMap = make(map[string]Decision)
	}

	// For workflow documents: Build comprehensive ApprovalsMap with ALL stage approvals
	if doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		// Clear and rebuild with ALL stages
		doc.Versions[latestVersionIndex].ApprovalsMap = make(map[string]Decision)
		
		// Add approvals from ALL stages using composite keys
		for _, stage := range doc.Workflow.Stages {
			if stage.StageApprovals != nil {
				for approver, decision := range stage.StageApprovals {
					compositeKey := fmt.Sprintf("#%d %s", stage.StageNumber, approver)
					doc.Versions[latestVersionIndex].ApprovalsMap[compositeKey] = decision
				}
			}
		}
	} else {
		// For non-workflow documents: Use current document ApprovalsMap
		doc.Versions[latestVersionIndex].ApprovalsMap = s.deepCopyApprovalsMap(doc.ApprovalsMap)
	}
}

func (s *SmartContract) createWorkflowSnapshot(workflow WorkflowConfig, date, timeStr string) *WorkflowSnapshot {
	snapshot := &WorkflowSnapshot{
		Enabled:         workflow.Enabled,
		CurrentStage:    workflow.CurrentStage,
		CompletedStages: make([]int, len(workflow.CompletedStages)),
		TotalStages:     len(workflow.Stages),
	}
	copy(snapshot.CompletedStages, workflow.CompletedStages)

	// Check if workflow is complete
	if len(workflow.CompletedStages) == len(workflow.Stages) && len(workflow.Stages) > 0 {
		snapshot.CompletionDate = date
		snapshot.CompletionTime = timeStr
	}

	return snapshot
}

// NEW: Helper function to calculate stage progress summary
func (s *SmartContract) calculateStageProgress(stage WorkflowStage, isCurrentStage, isCompleted bool) StageProgressSummary {
	approvedCount := 0
	rejectedCount := 0
	pendingCount := 0
	// SCHEMA FIX: Initialize as empty slice, never nil
	pendingApprovers := make([]string, 0)

	// Count approvals in this stage
	for _, approver := range stage.Approvers {
		if decision, exists := stage.StageApprovals[approver]; exists {
			switch decision.Status {
			case Approved:
				approvedCount++
			case Rejected:
				rejectedCount++
			case Pending:
				pendingCount++
				pendingApprovers = append(pendingApprovers, approver)
			}
		} else {
			// No decision recorded = pending
			pendingCount++
			pendingApprovers = append(pendingApprovers, approver)
		}
	}

	// Calculate completion percentage
	completionPercent := 0.0
	totalApprovers := len(stage.Approvers)
	if totalApprovers > 0 {
		completionPercent = (float64(approvedCount) / float64(totalApprovers)) * 100
	}

	// Determine stage status
	status := "pending"
	if isCompleted {
		status = "completed"
	} else if rejectedCount > 0 {
		status = "rejected"
	} else if isCurrentStage {
		status = "in_progress"
	}

	// SCHEMA FIX: Ensure slices are never nil for schema compliance
	approvers := make([]string, 0, len(stage.Approvers))
	approvers = append(approvers, stage.Approvers...)

	// Double-check: ensure never nil
	if approvers == nil {
		approvers = make([]string, 0)
	}
	if pendingApprovers == nil {
		pendingApprovers = make([]string, 0)
	}

	return StageProgressSummary{
		StageNumber:       stage.StageNumber,
		StageName:         stage.StageName,
		Status:            status,
		RequiredCount:     stage.RequiredCount,
		ApprovedCount:     approvedCount,
		RejectedCount:     rejectedCount,
		PendingCount:      pendingCount,
		CompletionPercent: completionPercent,
		Approvers:         approvers,
		PendingApprovers:  pendingApprovers,
	}
}

// NEW: Helper function to calculate overall workflow progress
func (s *SmartContract) calculateWorkflowProgress(workflow WorkflowConfig) *WorkflowProgressOverview {
	if !workflow.Enabled || len(workflow.Stages) == 0 {
		return nil
	}

	totalStages := len(workflow.Stages)
	completedStages := len(workflow.CompletedStages)
	currentStage := workflow.CurrentStage

	// Calculate overall progress
	overallProgress := 0.0
	if totalStages > 0 {
		overallProgress = (float64(completedStages) / float64(totalStages)) * 100
	}

	// Current stage info
	currentStageName := ""
	currentStageProgress := 0.0
	statusSummary := ""
	nextAction := ""

	// Build stage details - ensure never nil for schema compliance
	stageDetails := make([]StageProgressSummary, 0, len(workflow.Stages))
	for _, stage := range workflow.Stages {
		isCurrentStage := (stage.StageNumber == currentStage)
		isCompleted := false

		// Check if this stage is completed
		for _, completed := range workflow.CompletedStages {
			if completed == stage.StageNumber {
				isCompleted = true
				break
			}
		}

		stageSummary := s.calculateStageProgress(stage, isCurrentStage, isCompleted)
		stageDetails = append(stageDetails, stageSummary)

		// Set current stage info
		if isCurrentStage {
			currentStageName = stage.StageName
			currentStageProgress = stageSummary.CompletionPercent

			// Generate next action
			if stageSummary.PendingCount > 0 {
				nextAction = fmt.Sprintf("Waiting for %d approvers in %s", stageSummary.PendingCount, stage.StageName)
			} else if stageSummary.ApprovedCount >= stage.RequiredCount {
				nextAction = "Ready to advance to next stage"
			}
		}
	}

	// Generate status summary
	if completedStages == totalStages {
		statusSummary = "Workflow completed - all stages approved"
		nextAction = "No action required"
	} else if currentStage > totalStages {
		statusSummary = "Workflow rejected"
		nextAction = "Workflow has been rejected"
	} else {
		statusSummary = fmt.Sprintf("Stage %d of %d in progress: %s", currentStage, totalStages, currentStageName)
	}

	return &WorkflowProgressOverview{
		CurrentStageNumber:   currentStage,
		CurrentStageName:     currentStageName,
		CurrentStageProgress: currentStageProgress,
		OverallProgress:      overallProgress,
		CompletedStages:      completedStages,
		TotalStages:          totalStages,
		StatusSummary:        statusSummary,
		NextAction:           nextAction,
		StageDetails:         stageDetails,
	}
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// SubmitDocument - Unified document submission supporting both simple and advanced workflows
// Progressive disclosure: if workflowJSON is empty, creates simple workflow from approversJSON
// If workflowJSON is provided, uses advanced multi-stage workflow
func (s *SmartContract) SubmitDocument(ctx contractapi.TransactionContextInterface, id, title, description, hash, uploader, validDecisionsJSON, approversJSON, workflowJSON, deadlineHours, stageDeadlineHours string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":          id,
		"title":       title,
		"description": description,
		"hash":        hash,
		"uploader":    uploader,
		"decisions":   validDecisionsJSON,
	}); err != nil {
		return err
	}

	// Mode detection - must have either approversJSON (simple) or workflowJSON (advanced)
	if approversJSON == "" && workflowJSON == "" {
		return fmt.Errorf("❌ Must provide either approversJSON (simple mode) or workflowJSON (advanced mode)")
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

	// Check if document already exists
	exists, err := s.DocumentExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("❌ Document '%s' already exists. Use SubmitNewVersion function to add new versions to existing documents", id)
	}

	// Parse valid decisions
	var validDecisions []string
	if err := json.Unmarshal([]byte(validDecisionsJSON), &validDecisions); err != nil {
		return fmt.Errorf("❌ Invalid decisions format. Expected JSON array like [\"APPROVED\",\"REJECTED\"]. Error: %v", err)
	}

	if len(validDecisions) == 0 {
		return fmt.Errorf("❌ At least one valid decision must be specified (e.g., APPROVED, REJECTED)")
	}

	// Create workflow configuration based on mode
	var workflowConfig WorkflowConfig
	var deadlineConfig DeadlineConfig

	if workflowJSON == "" {
		// SIMPLE MODE: Create single-stage workflow from approversJSON
		var approvers []string
		if err := json.Unmarshal([]byte(approversJSON), &approvers); err != nil {
			return fmt.Errorf("❌ Invalid approvers format. Expected JSON array like [\"user1\",\"user2\"]. Error: %v", err)
		}

		if len(approvers) == 0 {
			return fmt.Errorf("❌ At least one approver must be specified")
		}

		// Create single-stage workflow
		stageApprovals := make(map[string]Decision)
		for _, approver := range approvers {
			// Store plain approver names in StageApprovals (composite keys created separately)
			stageApprovals[approver] = Decision{
				Status:  "PENDING",
				Comment: "",
				Date:    date,
				Time:    timeStr,
			}
		}

		workflowConfig = WorkflowConfig{
			Enabled: true,
			Stages: []WorkflowStage{{
				StageNumber:    1,
				StageName:      "Review and Approval",
				RequiredCount:  0, // 0 means all approvers needed
				Approvers:      approvers,
				StageApprovals: stageApprovals,
				AutoAdvance:    true,
				Description:    "Primary approval stage",
			}},
			CurrentStage:    1,
			CompletedStages: []int{},
			StageHistory: []StageTransition{{
				FromStage:  0,
				ToStage:    1,
				Transition: "initialized",
				Actor:      uploader,
				Date:       date,
				Time:       timeStr,
				Comment:    "Simple workflow document created",
			}},
		}

		// Simple deadline (document-level)
		if deadlineHours != "" {
			hours, err := strconv.Atoi(deadlineHours)
			if err != nil || hours <= 0 {
				return fmt.Errorf("❌ Invalid deadline hours: %s. Must be a positive integer", deadlineHours)
			}

			currentTime := time.Now()
			deadlineTime := currentTime.Add(time.Duration(hours) * time.Hour)
			location, _ := time.LoadLocation("Asia/Riyadh")
			if location == nil {
				location = time.FixedZone("AST", 3*3600)
			}
			deadlineTimeLocal := deadlineTime.In(location)
			deadlineTimestamp := deadlineTimeLocal.Format("2006-01-02 15:04:05")

			deadlineConfig = DeadlineConfig{
				Enabled:          true,
				DocumentDeadline: deadlineTimestamp,
				StageDeadlines:   make(map[string]string),
				WarningThreshold: func() int {
					if hours/4 > 1 {
						return hours / 4
					} else {
						return 1
					}
				}(),
				AutoReject:        false,
				EscalationEnabled: false,
				EscalationTargets: []string{},
				AutomationHooks:   []AutomationHook{},
			}
		} else {
			deadlineConfig = DeadlineConfig{
				Enabled:           false,
				DocumentDeadline:  "",
				StageDeadlines:    make(map[string]string),
				WarningThreshold:  6,
				AutoReject:        false,
				EscalationEnabled: false,
				EscalationTargets: []string{},
				AutomationHooks:   []AutomationHook{},
			}
		}

	} else {
		// ADVANCED MODE: Use provided workflow
		var workflowStages []WorkflowStage
		if err := json.Unmarshal([]byte(workflowJSON), &workflowStages); err != nil {
			return fmt.Errorf("❌ Invalid workflow format. Expected JSON array of workflow stages. Error: %v", err)
		}

		if len(workflowStages) == 0 {
			return fmt.Errorf("❌ At least one workflow stage must be defined")
		}

		// Process and validate workflow stages
		for i := range workflowStages {
			if workflowStages[i].StageNumber != i+1 {
				workflowStages[i].StageNumber = i + 1
			}
			if len(workflowStages[i].Approvers) == 0 {
				return fmt.Errorf("❌ Stage %d must have at least one approver", i+1)
			}
			if workflowStages[i].RequiredCount <= 0 {
				workflowStages[i].RequiredCount = len(workflowStages[i].Approvers)
			}

			// Initialize stage-specific approvals map with plain approver names
			stageApprovals := make(map[string]Decision)
			for _, approver := range workflowStages[i].Approvers {
				// Store plain approver names in StageApprovals (composite keys created separately)
				stageApprovals[approver] = Decision{
					Status:  "PENDING",
					Comment: "",
					Date:    date,
					Time:    timeStr,
				}
			}
			workflowStages[i].StageApprovals = stageApprovals
		}

		workflowConfig = WorkflowConfig{
			Enabled:         true,
			Stages:          workflowStages,
			CurrentStage:    1,
			CompletedStages: []int{},
			StageHistory: []StageTransition{{
				FromStage:  0,
				ToStage:    1,
				Transition: "initialized",
				Actor:      uploader,
				Date:       date,
				Time:       timeStr,
				Comment:    "Advanced workflow document created",
			}},
		}

		// Advanced deadlines (per-stage)
		if stageDeadlineHours != "" {
			stageHoursSlice := strings.Split(stageDeadlineHours, ",")
			if len(stageHoursSlice) != len(workflowStages) {
				return fmt.Errorf("❌ Number of stage deadlines (%d) must match number of stages (%d)", len(stageHoursSlice), len(workflowStages))
			}

			deadlineConfig = DeadlineConfig{
				Enabled:           true,
				DocumentDeadline:  "",
				StageDeadlines:    make(map[string]string),
				WarningThreshold:  6,
				AutoReject:        false,
				EscalationEnabled: false,
				EscalationTargets: []string{},
				AutomationHooks:   []AutomationHook{},
			}

			currentTime := time.Now()
			location, _ := time.LoadLocation("Asia/Riyadh")
			if location == nil {
				location = time.FixedZone("AST", 3*3600)
			}

			var finalStageHours int
			for i, stageHoursStr := range stageHoursSlice {
				hours, err := strconv.Atoi(strings.TrimSpace(stageHoursStr))
				if err != nil || hours <= 0 {
					return fmt.Errorf("❌ Invalid stage deadline hours at stage %d: %s", i+1, stageHoursStr)
				}

				stageDeadlineTime := currentTime.Add(time.Duration(hours) * time.Hour)
				stageDeadlineLocal := stageDeadlineTime.In(location)
				stageDeadlineTimestamp := stageDeadlineLocal.Format("2006-01-02 15:04:05")
				deadlineConfig.StageDeadlines[strconv.Itoa(workflowStages[i].StageNumber)] = stageDeadlineTimestamp
				finalStageHours = hours
			}

			// Set document deadline to the final stage deadline
			finalDeadlineTime := currentTime.Add(time.Duration(finalStageHours) * time.Hour)
			finalDeadlineLocal := finalDeadlineTime.In(location)
			deadlineConfig.DocumentDeadline = finalDeadlineLocal.Format("2006-01-02 15:04:05")
		} else {
			deadlineConfig = DeadlineConfig{
				Enabled:           false,
				DocumentDeadline:  "",
				StageDeadlines:    make(map[string]string),
				WarningThreshold:  6,
				AutoReject:        false,
				EscalationEnabled: false,
				EscalationTargets: []string{},
				AutomationHooks:   []AutomationHook{},
			}
		}
	}

	// Create ApprovalsMap with appropriate key format based on workflow type
	approvalsMap := make(map[string]Decision)
	for approver, decision := range workflowConfig.Stages[0].StageApprovals {
		var key string
		if workflowJSON == "" {
			// Simple workflow - use plain approver names (no stage prefix needed)
			key = approver
		} else {
			// Advanced workflow - use composite keys to distinguish stages
			key = fmt.Sprintf("#%d %s", workflowConfig.Stages[0].StageNumber, approver)
		}
		approvalsMap[key] = decision
	}

	// Calculate decision counts
	decisionCounts := make(map[string]int)
	for _, decision := range approvalsMap {
		decisionCounts[string(decision.Status)]++
	}

	// Create the document
	document := Document{
		ID:            id,
		Title:         title,
		Description:   description,
		LatestVersion: 1,
		Versions: []DocumentVersion{{
			Version:          1,
			Hash:             hash,
			Submitter:        uploader,
			Date:             date,
			Time:             timeStr,
			ApprovalsMap:     approvalsMap,
			ValidDecisions:   validDecisions,
			ApproversChanged: false,
			DecisionsChanged: false,
			WorkflowSnapshot: &WorkflowSnapshot{
				Enabled:         false,
				CurrentStage:    0,
				CompletedStages: []int{},
				TotalStages:     0,
				CompletionDate:  "Not Completed",
				CompletionTime:  "Not Completed",
			},
		}},
		Uploader:          uploader,
		PrivilegedEditors: []string{uploader},
		Editors:           []string{uploader},
		ApprovalsMap:      approvalsMap,
		ValidDecisions:    validDecisions,
		Workflow:          workflowConfig,
		VersionWorkflows:  make(map[string]*WorkflowConfig),
		DeadlineConfig:    &deadlineConfig,
		DeadlineStatus: func() *DeadlineStatus {
			timeRemaining := int64(0)
			overallStatus := "no_deadlines"
			if deadlineConfig.Enabled {
				overallStatus = "active"
				if deadlineConfig.DocumentDeadline != "" {
					if dl, err := time.Parse("2006-01-02 15:04:05", deadlineConfig.DocumentDeadline); err == nil {
						timeRemaining = int64(time.Until(dl).Hours())
					}
				}
			}
			return &DeadlineStatus{
				DocumentId:        id,
				OverallStatus:     overallStatus,
				DocumentDeadline:  deadlineConfig.DocumentDeadline,
				TimeRemaining:     timeRemaining,
				StageStatuses:     make(map[string]StageDeadlineStatus),
				WarningsTriggered: []DeadlineWarning{},
				BreachesRecorded:  []DeadlineBreach{},
			}
		}(),
		CreatedDate:      date,
		CreatedTime:      timeStr,
		LastModifiedDate: date,
		LastModifiedTime: timeStr,
	}

	// Store the document
	documentJSON, err := json.Marshal(document)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(id, documentJSON); err != nil {
		return err
	}

	return nil
}

// SubmitNewVersion submits a new version of an existing document
func (s *SmartContract) SubmitNewVersion(ctx contractapi.TransactionContextInterface, id, hash, submitter string, resetApprovals bool) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":        id,
		"hash":      hash,
		"submitter": submitter,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

	// Check if document exists
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found. Use SubmitDocument to create new documents", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Security Check: Verify the submitter is an editor or privileged editor
	if !s.isUserInList(submitter, doc.Editors) && !s.isUserInList(submitter, doc.PrivilegedEditors) {
		return fmt.Errorf("submitter %s is not an editor or privileged editor for document %s", submitter, id)
	}

	// Security Check: Verify the hash is different from the latest version
	if len(doc.Versions) > 0 && doc.Versions[len(doc.Versions)-1].Hash == hash {
		return fmt.Errorf("new version hash is the same as the latest version hash")
	}

	// STEP 1: Preserve current workflow state for the current version - FIX FOR BUG #2
	if doc.Workflow.Enabled {
		// Initialize VersionWorkflows map if nil
		if doc.VersionWorkflows == nil {
			doc.VersionWorkflows = make(map[string]*WorkflowConfig)
		}

		// Deep copy current workflow state for preservation
		currentWorkflowState := s.deepCopyWorkflowConfig(doc.Workflow)
		doc.VersionWorkflows[fmt.Sprintf("%d", doc.LatestVersion)] = &currentWorkflowState

		// Create workflow snapshot for version history
		workflowSnapshot := s.createWorkflowSnapshot(doc.Workflow, date, timeStr)

		// Update current version's workflow snapshot in existing versions
		if len(doc.Versions) > 0 {
			for i := range doc.Versions {
				if doc.Versions[i].Version == doc.LatestVersion {
					doc.Versions[i].WorkflowSnapshot = workflowSnapshot
					break
				}
			}
		}
	}

	// STEP 2: Update document for new version
	doc.LatestVersion++
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// STEP 3: Reset approvals and workflow for new version - FIX FOR BUG #2
	var newVersionApprovals map[string]Decision

	if resetApprovals {
		// Reset workflow state FIRST for new version - COMPLETE FIX
		if doc.Workflow.Enabled {
			// Reset workflow to initial state
			doc.Workflow.CurrentStage = 1
			doc.Workflow.CompletedStages = []int{}

			// Reset all stage approvals to pending - FIX FOR BUG #1
			doc.Workflow.Stages = s.initializeStageApprovals(doc.Workflow.Stages, date, timeStr)

			// CRITICAL FIX: Build ALL composite keys for new version (not just current stage)
			// This ensures version shows all 9 approvals as PENDING with fresh timestamps
			s.buildAllCompositeKeysApprovalsMap(&doc, date, timeStr)

			// RESET stage history for new version - FIX FOR BUG #3
			doc.Workflow.StageHistory = []StageTransition{{
				FromStage:  0,
				ToStage:    1,
				Transition: "version_reset",
				Actor:      submitter,
				Date:       date,
				Time:       timeStr,
				Comment:    fmt.Sprintf("Workflow reset for version %d submission", doc.LatestVersion),
			}}
		} else {
			// For non-workflow documents, reset existing approvers to pending
			for approver := range doc.ApprovalsMap {
				doc.ApprovalsMap[approver] = Decision{
					Status:  Pending,
					Comment: "",
					Date:    date,
					Time:    timeStr,
				}
			}
		}

		// STEP 4: Create snapshot of RESET state for new version
		newVersionApprovals = make(map[string]Decision)
		for k, v := range doc.ApprovalsMap {
			newVersionApprovals[k] = v
		}
	} else {
		// If not resetting, preserve current approvals for new version
		newVersionApprovals = make(map[string]Decision)
		for k, v := range doc.ApprovalsMap {
			newVersionApprovals[k] = v
		}
	}

	// Add the new version to the history - FIX FOR BUG #2
	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             hash,
		Submitter:        submitter,
		Date:             date,
		Time:             timeStr,
		ApprovalsMap:     newVersionApprovals, // Use the reset state for new version
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: false,
		DecisionsChanged: false,
		WorkflowSnapshot: nil, // Will be populated as workflow progresses
	}
	doc.Versions = append(doc.Versions, newVersion)

	// Create the hash-to-ID index
	hashKey := fmt.Sprintf("hash->%s", hash)
	if err := ctx.GetStub().PutState(hashKey, []byte(id)); err != nil {
		return fmt.Errorf("failed to create hash-to-ID index: %v", err)
	}

	// Save document
	updatedData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(id, updatedData); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Emit event
	description := fmt.Sprintf("New version %d of document %s submitted by %s", doc.LatestVersion, id, submitter)
	details := map[string]interface{}{
		"hash":           hash,
		"resetApprovals": resetApprovals,
	}

	return s.emitEvent(ctx, "NewVersionSubmitted", submitter, description, doc.ID, doc.LatestVersion, details)
}

// ApproveDocument - Unified approval function with progressive disclosure
// Automatically detects document workflow type and handles both simple and multi-stage approvals
func (s *SmartContract) ApproveDocument(ctx contractapi.TransactionContextInterface, id, approver, decision, comment string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":       id,
		"approver": approver,
		"decision": decision,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
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

	// AUTO-DETECT WORKFLOW TYPE - Progressive Disclosure Pattern
	if doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		// MULTI-STAGE WORKFLOW MODE
		return s.handleMultiStageApproval(ctx, &doc, approver, decision, comment, date, timeStr)
	} else {
		// SIMPLE APPROVAL MODE (legacy single-stage)
		return s.handleSimpleApproval(ctx, &doc, approver, decision, comment, date, timeStr)
	}
}

// handleSimpleApproval - Handles legacy single-stage approval process
func (s *SmartContract) handleSimpleApproval(ctx contractapi.TransactionContextInterface, doc *Document, approver, decision, comment, date, timeStr string) error {
	// Initialize ApprovalsMap if needed
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}

	// Check if user is authorized (anyone can approve in simple mode unless restricted)
	// For simple mode, we allow any approver that was set during document submission

	// Validate decision
	validDecision := false
	for _, validDec := range doc.ValidDecisions {
		if decision == validDec {
			validDecision = true
			break
		}
	}
	if !validDecision {
		return fmt.Errorf("invalid decision '%s'. Valid decisions: %v", decision, doc.ValidDecisions)
	}

	// Update approval decision in ApprovalsMap (use plain approver name for simple workflows)
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}
	
	// Remove any pending entry with composite key format for this approver
	pendingKey := fmt.Sprintf("#1 %s", approver)
	delete(doc.ApprovalsMap, pendingKey)
	
	// Add the new decision with plain approver key
	doc.ApprovalsMap[approver] = Decision{
		Status:  ApproverStatus(decision),
		Comment: comment,
		Date:    date,
		Time:    timeStr,
	}

	// Check if document should be marked as approved/rejected
	s.updateDocumentStatusFromMap(doc)

	// Sync current approvals to latest version ApprovalsMap
	s.syncCurrentApprovalsToLatestVersion(doc)

	// Save document
	updatedData, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(doc.ID, updatedData); err != nil {
		return err
	}

	// Emit event
	detailsMap := map[string]interface{}{
		"approver": approver,
		"mode":     "simple",
	}
	return s.emitEvent(ctx, "DocumentApproved", approver, decision, doc.ID, doc.LatestVersion, detailsMap)
}

// handleMultiStageApproval - Handles multi-stage workflow approval process
func (s *SmartContract) handleMultiStageApproval(ctx contractapi.TransactionContextInterface, doc *Document, approver, decision, comment, date, timeStr string) error {
	// Get current stage
	if doc.Workflow.CurrentStage <= 0 || doc.Workflow.CurrentStage > len(doc.Workflow.Stages) {
		return fmt.Errorf("invalid current stage %d", doc.Workflow.CurrentStage)
	}

	currentStage := doc.Workflow.Stages[doc.Workflow.CurrentStage-1]

	// Check if approver is authorized for this stage
	authorized := false
	for _, stageApprover := range currentStage.Approvers {
		if stageApprover == approver {
			authorized = true
			break
		}
	}
	if !authorized {
		return fmt.Errorf("user %s is not authorized to approve stage %d", approver, doc.Workflow.CurrentStage)
	}

	// Validate decision
	validDecision := false
	for _, validDec := range doc.ValidDecisions {
		if decision == validDec {
			validDecision = true
			break
		}
	}
	if !validDecision {
		return fmt.Errorf("invalid decision '%s'. Valid decisions: %v", decision, doc.ValidDecisions)
	}

	// Initialize stage approvals if needed
	if doc.Workflow.Stages[doc.Workflow.CurrentStage-1].StageApprovals == nil {
		doc.Workflow.Stages[doc.Workflow.CurrentStage-1].StageApprovals = make(map[string]Decision)
	}

	// Update stage approval
	doc.Workflow.Stages[doc.Workflow.CurrentStage-1].StageApprovals[approver] = Decision{
		Status:  ApproverStatus(decision),
		Comment: comment,
		Date:    date,
		Time:    timeStr,
	}

	// Update global ApprovalsMap - remove old pending entry and add new decision
	if doc.ApprovalsMap == nil {
		doc.ApprovalsMap = make(map[string]Decision)
	}
	
	// Determine appropriate key format based on workflow complexity
	isSimpleWorkflow := len(doc.Workflow.Stages) == 1
	
	var approverKey string
	if isSimpleWorkflow {
		// Simple workflow - use plain approver names
		approverKey = approver
	} else {
		// Advanced workflow - use composite keys to distinguish stages
		approverKey = fmt.Sprintf("#%d %s", doc.Workflow.CurrentStage, approver)
	}
	
	// Find and remove the pending entry for this approver
	delete(doc.ApprovalsMap, approverKey)
	
	// Add the new decision with appropriate key format
	doc.ApprovalsMap[approverKey] = Decision{
		Status:  ApproverStatus(decision),
		Comment: comment,
		Date:    date,
		Time:    timeStr,
	}

	// Check stage completion and auto-advance logic
	s.processStageCompletion(doc)

	// Sync current approvals to latest version ApprovalsMap  
	s.syncCurrentApprovalsToLatestVersion(doc)

	// Save document
	updatedData, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(doc.ID, updatedData); err != nil {
		return err
	}

	// Emit event
	detailsMap := map[string]interface{}{
		"stage":    doc.Workflow.CurrentStage,
		"approver": approver,
		"mode":     "workflow",
	}
	return s.emitEvent(ctx, "StageApproved", approver, decision, doc.ID, doc.LatestVersion, detailsMap)
}

// updateDocumentStatusFromMap - Updates document status based on ApprovalsMap for simple mode
func (s *SmartContract) updateDocumentStatusFromMap(doc *Document) {
	if doc.ApprovalsMap == nil {
		return
	}

	// Count approvals and rejections
	approvals := 0
	rejections := 0

	for _, decision := range doc.ApprovalsMap {
		if decision.Status == Approved {
			approvals++
		} else if decision.Status == Rejected {
			rejections++
		}
	}

	// Status is computed dynamically by determineDocumentStatus function
	// This function ensures ApprovalsMap is ready for status computation
	// No direct status field to set - status is computed on-demand
}

// processStageCompletion - Checks stage completion and handles auto-advance
func (s *SmartContract) processStageCompletion(doc *Document) {
	if doc.Workflow.CurrentStage <= 0 || doc.Workflow.CurrentStage > len(doc.Workflow.Stages) {
		return
	}

	currentStage := &doc.Workflow.Stages[doc.Workflow.CurrentStage-1]

	if currentStage.StageApprovals == nil {
		return
	}

	// Count stage approvals and rejections
	approvals := 0
	rejections := 0

	for _, decision := range currentStage.StageApprovals {
		if decision.Status == Approved {
			approvals++
		} else if decision.Status == Rejected {
			rejections++
		}
	}

	requiredCount := currentStage.RequiredCount
	if requiredCount <= 0 {
		requiredCount = len(currentStage.Approvers)
	}

	// Check if stage is complete
	stageComplete := false
	if rejections > 0 {
		// Any rejection fails the stage
		stageComplete = true
		// Note: Status is computed dynamically, no direct field to set
	} else if approvals >= requiredCount {
		// Sufficient approvals complete the stage
		stageComplete = true

		// Check if this is the last stage
		if doc.Workflow.CurrentStage >= len(doc.Workflow.Stages) {
			// Workflow complete - status computed dynamically
		} else if currentStage.AutoAdvance {
			doc.Workflow.CurrentStage++
		}
	}

	if stageComplete {
		// Update stage counters if fields exist
		// Note: WorkflowStage struct may not have these fields
	}
}

func (s *SmartContract) QueryDocumentStatus(ctx contractapi.TransactionContextInterface, id string) (*StatusResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// CRITICAL FIX: Ensure all arrays and maps are properly initialized before returning
	s.ensureDocumentArraysInitialized(&doc)

	// WORKFLOW FIX: Update compatibility ApprovalsMap to show current stage only
	if doc.Workflow.Enabled {
		s.updateCompatibilityApprovalsMap(&doc)
	}

	// SCHEMA FIX: Ensure VersionWorkflows is never nil
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}

	// SCHEMA FIX: Ensure all versions have WorkflowSnapshot (never nil)
	fixedVersions := make([]DocumentVersion, len(doc.Versions))
	for i, version := range doc.Versions {
		fixedVersions[i] = version

		// Note: Version ApprovalsMap now always contains clean composite keys from creation
		// No cleanup needed since SubmitDocumentAdvanced creates proper composite keys from start

		if fixedVersions[i].WorkflowSnapshot == nil {
			// Create empty workflow snapshot for schema compliance with user-friendly status
			fixedVersions[i].WorkflowSnapshot = &WorkflowSnapshot{
				Enabled:         false,
				CurrentStage:    0,
				CompletedStages: []int{},
				TotalStages:     0,
				CompletionDate:  "Not Completed", // User-friendly status message
				CompletionTime:  "Not Completed", // User-friendly status message
			}
		} else {
			// Ensure existing snapshots have user-friendly completion status if not completed
			if fixedVersions[i].WorkflowSnapshot.CompletionDate == "" {
				fixedVersions[i].WorkflowSnapshot.CompletionDate = "Not Completed"
			}
			if fixedVersions[i].WorkflowSnapshot.CompletionTime == "" {
				fixedVersions[i].WorkflowSnapshot.CompletionTime = "Not Completed"
			}
		}
	}

	resp := &StatusResponse{
		ID:                doc.ID,
		Title:             doc.Title,
		Description:       doc.Description,
		LatestVersion:     doc.LatestVersion,
		Versions:          fixedVersions, // Use the fixed versions with proper WorkflowSnapshot
		Uploader:          doc.Uploader,
		PrivilegedEditors: doc.PrivilegedEditors,
		Editors:           doc.Editors,
		ApprovalsMap:      doc.ApprovalsMap,
		ValidDecisions:    doc.ValidDecisions,
		Workflow:          doc.Workflow,
		CurrentStageInfo:  nil, // Will be set below if workflow is enabled
		// SCHEMA FIX: Include version workflow history - never nil
		VersionWorkflows: doc.VersionWorkflows,
		// NEW: Add user-friendly workflow progress information
		WorkflowProgress: s.calculateWorkflowProgress(doc.Workflow),
		DeadlineConfig:   doc.DeadlineConfig,
		DeadlineStatus:   doc.DeadlineStatus,
		CreatedDate:      doc.CreatedDate,
		CreatedTime:      doc.CreatedTime,
		LastModifiedDate: doc.LastModifiedDate,
		LastModifiedTime: doc.LastModifiedTime,
		DecisionCounts:   make(map[string]int),
	}

	// Initialize counts for all possible decisions
	resp.DecisionCounts[string(Pending)] = 0
	for _, decision := range doc.ValidDecisions {
		resp.DecisionCounts[decision] = 0
	}

	// CRITICAL FIX: Count decisions properly - use StageApprovals if ApprovalsMap is empty/inconsistent
	if len(doc.ApprovalsMap) == 0 && doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		// For documents with empty ApprovalsMap (old Simple docs), count from current stage
		currentStageIndex := doc.Workflow.CurrentStage - 1
		if currentStageIndex >= 0 && currentStageIndex < len(doc.Workflow.Stages) {
			for _, decision := range doc.Workflow.Stages[currentStageIndex].StageApprovals {
				resp.DecisionCounts[string(decision.Status)]++
			}
		}
	} else {
		// Use ApprovalsMap as normal (for Advanced docs and fixed Simple docs)
		for _, decision := range doc.ApprovalsMap {
			resp.DecisionCounts[string(decision.Status)]++
		}
	}

	// Set CurrentStageInfo - always provide a value to avoid schema validation errors
	if doc.Workflow.Enabled && doc.Workflow.CurrentStage > 0 && doc.Workflow.CurrentStage <= len(doc.Workflow.Stages) {
		currentStage := doc.Workflow.Stages[doc.Workflow.CurrentStage-1]
		resp.CurrentStageInfo = &currentStage
	} else {
		// For non-workflow documents, provide an empty stage info
		resp.CurrentStageInfo = &WorkflowStage{
			StageNumber:    0,
			StageName:      "",
			RequiredCount:  0,
			Approvers:      []string{},
			StageApprovals: make(map[string]Decision), // FIX: Initialize empty map instead of null
			AutoAdvance:    false,
			Description:    "",
		}
	}

	return resp, nil
}

func (s *SmartContract) AddEditor(ctx contractapi.TransactionContextInterface, id, newEditor, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":        id,
		"newEditor": newEditor,
		"invokerId": invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
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

	// Security Check: Verify the invoker is a privileged editor
	if !s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, id)
	}

	// Check if user is already an editor
	if s.isUserInList(newEditor, doc.Editors) {
		return fmt.Errorf("user %s is already an editor for document %s", newEditor, id)
	}

	// Add the new editor
	doc.Editors = append(doc.Editors, newEditor)
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Save document
	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(id, updated); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Emit event
	description := fmt.Sprintf("Editor %s added to document %s by %s", newEditor, id, invokerId)
	details := map[string]interface{}{
		"newEditor": newEditor,
	}

	return s.emitEvent(ctx, "EditorAdded", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) RemoveEditor(ctx contractapi.TransactionContextInterface, id, editorToRemove, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":             id,
		"editorToRemove": editorToRemove,
		"invokerId":      invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
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

	// Security Check: Verify the invoker is a privileged editor
	if !s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, id)
	}

	// Cannot remove the uploader from editors
	if editorToRemove == doc.Uploader {
		return fmt.Errorf("cannot remove document uploader %s from editors", editorToRemove)
	}

	// Check if user is actually an editor
	if !s.isUserInList(editorToRemove, doc.Editors) {
		return fmt.Errorf("user %s is not an editor for document %s", editorToRemove, id)
	}

	// Remove the editor
	var newEditors []string
	for _, editor := range doc.Editors {
		if editor != editorToRemove {
			newEditors = append(newEditors, editor)
		}
	}
	doc.Editors = newEditors
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Save document
	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(id, updated); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Emit event
	description := fmt.Sprintf("Editor %s removed from document %s by %s", editorToRemove, id, invokerId)
	details := map[string]interface{}{
		"removedEditor": editorToRemove,
	}

	return s.emitEvent(ctx, "EditorRemoved", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) AddPrivilegedEditor(ctx contractapi.TransactionContextInterface, id, newPrivilegedEditor, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":                  id,
		"newPrivilegedEditor": newPrivilegedEditor,
		"invokerId":           invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
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

	// Security Check: Verify the invoker is a privileged editor
	if !s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, id)
	}

	// Check if user is already a privileged editor
	if s.isUserInList(newPrivilegedEditor, doc.PrivilegedEditors) {
		return fmt.Errorf("user %s is already a privileged editor for document %s", newPrivilegedEditor, id)
	}

	// Add the new privileged editor
	doc.PrivilegedEditors = append(doc.PrivilegedEditors, newPrivilegedEditor)

	// Also add to regular editors if not already there
	if !s.isUserInList(newPrivilegedEditor, doc.Editors) {
		doc.Editors = append(doc.Editors, newPrivilegedEditor)
	}

	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Save document
	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(id, updated); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Emit event
	description := fmt.Sprintf("Privileged editor %s added to document %s by %s", newPrivilegedEditor, id, invokerId)
	details := map[string]interface{}{
		"newPrivilegedEditor": newPrivilegedEditor,
	}

	return s.emitEvent(ctx, "PrivilegedEditorAdded", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) RemovePrivilegedEditor(ctx contractapi.TransactionContextInterface, id, privilegedEditorToRemove, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"id":                       id,
		"privilegedEditorToRemove": privilegedEditorToRemove,
		"invokerId":                invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
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

	// Security Check: Only the document owner (uploader) can remove privileged editors
	if invokerId != doc.Uploader {
		return fmt.Errorf("only the document owner %s can remove privileged editors", doc.Uploader)
	}

	// Cannot remove the document owner from privileged editors
	if privilegedEditorToRemove == doc.Uploader {
		return fmt.Errorf("cannot remove document owner %s from privileged editors", privilegedEditorToRemove)
	}

	// Check if user is actually a privileged editor
	if !s.isUserInList(privilegedEditorToRemove, doc.PrivilegedEditors) {
		return fmt.Errorf("user %s is not a privileged editor for document %s", privilegedEditorToRemove, id)
	}

	// Remove from privileged editors list
	var newPrivilegedEditors []string
	for _, editor := range doc.PrivilegedEditors {
		if editor != privilegedEditorToRemove {
			newPrivilegedEditors = append(newPrivilegedEditors, editor)
		}
	}
	doc.PrivilegedEditors = newPrivilegedEditors

	// Note: Keep them in regular editors list - they can still edit documents

	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Save document
	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(id, updated); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// Emit event
	description := fmt.Sprintf("Privileged editor %s removed from document %s by %s", privilegedEditorToRemove, id, invokerId)
	details := map[string]interface{}{
		"removedPrivilegedEditor": privilegedEditorToRemove,
	}

	return s.emitEvent(ctx, "PrivilegedEditorRemoved", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) UpdateDocumentApprovers(ctx contractapi.TransactionContextInterface, documentID string, newApproversJSON string, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"documentID":       documentID,
		"newApproversJSON": newApproversJSON,
		"invokerId":        invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

	// 1. Retrieve the document
	data, err := ctx.GetStub().GetState(documentID)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", documentID)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// 2. Security Check: Verify invokerId is a Privileged Editor for this document
	if !s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, documentID)
	}

	// 3. Parse new approvers
	var newApprovers []string
	if err := json.Unmarshal([]byte(newApproversJSON), &newApprovers); err != nil {
		return fmt.Errorf("invalid newApprovers JSON: %v", err)
	}

	// 4. Store old approvers for event details (sorted for deterministic consensus)
	oldApprovers := make([]string, 0, len(doc.ApprovalsMap))
	for approver := range doc.ApprovalsMap {
		oldApprovers = append(oldApprovers, approver)
	}
	sort.Strings(oldApprovers) // CRITICAL: Ensure deterministic order for consensus

	// 5. Simple approach: Rebuild ApprovalsMap from scratch
	// Compare new list with old list, preserve existing decisions where possible
	updatedApprovalsMap := make(map[string]Decision)

	for _, approver := range newApprovers {
		if existingDecision, exists := doc.ApprovalsMap[approver]; exists {
			// Approver already exists - keep their existing decision
			updatedApprovalsMap[approver] = existingDecision
		} else {
			// New approver - set to PENDING with current timestamp
			updatedApprovalsMap[approver] = Decision{
				Status:  Pending,
				Comment: "",
				Date:    date,
				Time:    timeStr,
			}
		}
	}

	// Replace the entire ApprovalsMap with the new one
	doc.ApprovalsMap = updatedApprovalsMap

	// CRITICAL FIX: Also update workflow stage approvers to maintain synchronization
	if doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		if len(doc.Workflow.Stages) == 1 {
			// Single-stage workflow: update the only stage's approvers
			doc.Workflow.Stages[0].Approvers = newApprovers
		} else {
			// Multi-stage workflow: update current stage's approvers
			// Note: This is a simple approach - in complex scenarios you might want to update specific stages
			currentStageIndex := doc.Workflow.CurrentStage - 1
			if currentStageIndex >= 0 && currentStageIndex < len(doc.Workflow.Stages) {
				doc.Workflow.Stages[currentStageIndex].Approvers = newApprovers
			}
		}
	}

	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Increment version for approver changes (for consistency and testing)
	doc.LatestVersion++

	// Get current hash from latest version
	currentHash := "APPROVERS_UPDATED"
	if len(doc.Versions) > 0 {
		currentHash = doc.Versions[len(doc.Versions)-1].Hash
	}

	// Create snapshot of current approvals (the new ones we just set)
	currentApprovals := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		currentApprovals[k] = v
	}

	// Create new version entry
	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             currentHash,
		Submitter:        invokerId,
		Date:             date,
		Time:             timeStr,
		ApprovalsMap:     currentApprovals,
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: true,
		DecisionsChanged: false,
	}
	doc.Versions = append(doc.Versions, newVersion)

	// 7. Persist the updated document
	updatedDocBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(documentID, updatedDocBytes); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// 7. Emit event (ensure newApprovers is also sorted for deterministic consensus)
	sort.Strings(newApprovers) // CRITICAL: Ensure deterministic order for consensus
	description := fmt.Sprintf("Approvers updated for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"oldApprovers": oldApprovers,
		"newApprovers": newApprovers,
	}

	return s.emitEvent(ctx, "ApproversUpdated", invokerId, description, doc.ID, doc.LatestVersion, details)
}

func (s *SmartContract) GetDocumentIdByHash(ctx contractapi.TransactionContextInterface, hash string) (string, error) {
	if hash == "" {
		return "", fmt.Errorf("hash cannot be empty")
	}

	hashKey := fmt.Sprintf("hash->%s", hash)
	idBytes, err := ctx.GetStub().GetState(hashKey)
	if err != nil || idBytes == nil {
		return "", fmt.Errorf("no document found with hash %s", hash)
	}
	return string(idBytes), nil
}

func (s *SmartContract) DocumentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("document ID cannot be empty")
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

func (s *SmartContract) GetDocHistory(ctx contractapi.TransactionContextInterface, id string) ([]map[string]interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// CRITICAL FIX: Ensure all arrays and maps are properly initialized before returning
	s.ensureDocumentArraysInitialized(&doc)

	// WORKFLOW FIX: Update compatibility ApprovalsMap to show current stage only
	if doc.Workflow.Enabled {
		s.updateCompatibilityApprovalsMap(&doc)
	}

	var history []map[string]interface{}
	for _, version := range doc.Versions {
		// Count approval statuses for this version
		approvalCounts := make(map[string]int)
		approvalCounts[string(Pending)] = 0
		for _, decision := range doc.ValidDecisions {
			approvalCounts[decision] = 0
		}

		for _, decision := range version.ApprovalsMap {
			approvalCounts[string(decision.Status)]++
		}

		// Create detailed approval info
		approvalDetails := make(map[string]map[string]interface{})
		for approver, decision := range version.ApprovalsMap {
			approvalDetails[approver] = map[string]interface{}{
				"status":  string(decision.Status),
				"comment": decision.Comment,
				"date":    decision.Date,
				"time":    decision.Time,
			}
		}

		versionInfo := map[string]interface{}{
			"version":          version.Version,
			"hash":             version.Hash,
			"submitter":        version.Submitter,
			"date":             version.Date,
			"time":             version.Time,
			"approvalsMap":     approvalDetails,
			"validDecisions":   version.ValidDecisions,
			"approvalCounts":   approvalCounts,
			"approversChanged": version.ApproversChanged,
			"decisionsChanged": version.DecisionsChanged,
		}

		// Add change indicators
		if version.ApproversChanged {
			versionInfo["changeType"] = "Approvers Modified"
		} else if version.DecisionsChanged {
			versionInfo["changeType"] = "Valid Decisions Modified"
		} else {
			versionInfo["changeType"] = "Document Content Updated"
		}

		history = append(history, versionInfo)
	}

	return history, nil
}

func (s *SmartContract) GetAllDocuments(ctx contractapi.TransactionContextInterface) ([]*Document, error) {
	iterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	var documents []*Document
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		// Skip the hash-to-ID index entries
		if len(queryResponse.Key) > 5 && string(queryResponse.Key[:5]) == "hash->" {
			continue
		}

		var document Document
		err = json.Unmarshal(queryResponse.Value, &document)
		if err != nil {
			// This might happen if there are other non-document entries
			// For now, we'll just skip them
			continue
		}

		// Apply defensive initialization to ensure arrays are never null
		s.ensureDocumentArraysInitialized(&document)

		// WORKFLOW FIX: Update compatibility ApprovalsMap to show current stage only
		if document.Workflow.Enabled {
			s.updateCompatibilityApprovalsMap(&document)
		}

		// SCHEMA FIX: Ensure all versions have WorkflowSnapshot (never nil)
		for i := range document.Versions {
			// Note: Version ApprovalsMap now always contains clean composite keys from creation
			// No cleanup needed since SubmitDocumentAdvanced creates proper composite keys from start

			if document.Versions[i].WorkflowSnapshot == nil {
				// Create empty workflow snapshot for schema compliance with required fields
				document.Versions[i].WorkflowSnapshot = &WorkflowSnapshot{
					Enabled:         false,
					CurrentStage:    0,
					CompletedStages: []int{},
					TotalStages:     0,
					CompletionDate:  "Not Completed", // User-friendly status message
					CompletionTime:  "Not Completed", // User-friendly status message
				}
			} else {
				// Ensure existing snapshots have user-friendly completion status if not completed
				if document.Versions[i].WorkflowSnapshot.CompletionDate == "" {
					document.Versions[i].WorkflowSnapshot.CompletionDate = "Not Completed"
				}
				if document.Versions[i].WorkflowSnapshot.CompletionTime == "" {
					document.Versions[i].WorkflowSnapshot.CompletionTime = "Not Completed"
				}
			}
		}

		documents = append(documents, &document)
	}

	return documents, nil
}

// Additional utility functions for better document management

func (s *SmartContract) GetDocumentsByEditor(ctx contractapi.TransactionContextInterface, editorId string) ([]*Document, error) {
	if editorId == "" {
		return nil, fmt.Errorf("editor ID cannot be empty")
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var editorDocs []*Document
	for _, doc := range allDocs {
		if s.isUserInList(editorId, doc.Editors) {
			editorDocs = append(editorDocs, doc)
		}
	}

	return editorDocs, nil
}

func (s *SmartContract) GetDocumentsByApprover(ctx contractapi.TransactionContextInterface, approverId string) ([]*Document, error) {
	if approverId == "" {
		return nil, fmt.Errorf("approver ID cannot be empty")
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var approverDocs []*Document
	for _, doc := range allDocs {
		if _, exists := doc.ApprovalsMap[approverId]; exists {
			approverDocs = append(approverDocs, doc)
		}
	}

	return approverDocs, nil
}

func (s *SmartContract) GetPendingApprovals(ctx contractapi.TransactionContextInterface, approverId string) ([]*Document, error) {
	if approverId == "" {
		return nil, fmt.Errorf("approver ID cannot be empty")
	}

	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var pendingDocs []*Document
	for _, doc := range allDocs {
		if decision, exists := doc.ApprovalsMap[approverId]; exists && decision.Status == Pending {
			pendingDocs = append(pendingDocs, doc)
		}
	}

	return pendingDocs, nil
}

func (s *SmartContract) GetDocumentSummary(ctx contractapi.TransactionContextInterface, id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Initialize arrays and update compatibility ApprovalsMap - same as QueryDocumentStatus
	s.ensureDocumentArraysInitialized(&doc)

	// Update compatibility ApprovalsMap to show current stage only
	if doc.Workflow.Enabled {
		s.updateCompatibilityApprovalsMap(&doc)
	}

	// CRITICAL FIX: Calculate approval statistics properly - use StageApprovals if ApprovalsMap is empty
	var totalApprovers int
	decisionCounts := make(map[string]int)
	decisionCounts[string(Pending)] = 0
	for _, decision := range doc.ValidDecisions {
		decisionCounts[decision] = 0
	}

	// Use same logic as QueryDocumentStatus fix
	if len(doc.ApprovalsMap) == 0 && doc.Workflow.Enabled && len(doc.Workflow.Stages) > 0 {
		// For documents with empty ApprovalsMap (old Simple docs), count from current stage
		currentStageIndex := doc.Workflow.CurrentStage - 1
		if currentStageIndex >= 0 && currentStageIndex < len(doc.Workflow.Stages) {
			totalApprovers = len(doc.Workflow.Stages[currentStageIndex].StageApprovals)
			for _, decision := range doc.Workflow.Stages[currentStageIndex].StageApprovals {
				decisionCounts[string(decision.Status)]++
			}
		}
	} else {
		// Use ApprovalsMap as normal (for Advanced docs and fixed Simple docs)
		totalApprovers = len(doc.ApprovalsMap)
		for _, decision := range doc.ApprovalsMap {
			decisionCounts[string(decision.Status)]++
		}
	}

	// Calculate completion percentage
	completionPercent := 0.0
	if totalApprovers > 0 {
		completedDecisions := totalApprovers - decisionCounts[string(Pending)]
		completionPercent = (float64(completedDecisions) / float64(totalApprovers)) * 100
	}

	// Calculate workflow progress information
	workflowProgress := s.calculateWorkflowProgress(doc.Workflow)

	summary := map[string]interface{}{
		"id":                     doc.ID,
		"title":                  doc.Title,
		"description":            doc.Description,
		"latestVersion":          doc.LatestVersion,
		"totalVersions":          len(doc.Versions),
		"uploader":               doc.Uploader,
		"totalEditors":           len(doc.Editors),
		"totalPrivilegedEditors": len(doc.PrivilegedEditors),
		"totalApprovers":         totalApprovers,
		"decisionCounts":         decisionCounts,
		"completionPercent":      completionPercent,
		"validDecisions":         doc.ValidDecisions,
		"createdDate":            doc.CreatedDate,
		"createdTime":            doc.CreatedTime,
		"lastModifiedDate":       doc.LastModifiedDate,
		"lastModifiedTime":       doc.LastModifiedTime,
		"status":                 s.determineDocumentStatus(decisionCounts, doc.ValidDecisions),
		// NEW: Add user-friendly workflow progress information
		"workflowProgress": workflowProgress,
		"workflowEnabled":  doc.Workflow.Enabled,
	}

	return summary, nil
}

// Helper function to determine overall document status
func (s *SmartContract) determineDocumentStatus(decisionCounts map[string]int, validDecisions []string) string {
	pendingCount := decisionCounts[string(Pending)]

	if pendingCount == 0 {
		// All decisions made
		hasRejected := false
		for _, decision := range validDecisions {
			if decision == string(Rejected) && decisionCounts[decision] > 0 {
				hasRejected = true
				break
			}
		}

		if hasRejected {
			return "REJECTED"
		} else {
			return "FULLY_APPROVED"
		}
	} else {
		// Some decisions pending
		totalDecisions := 0
		for _, count := range decisionCounts {
			totalDecisions += count
		}

		if pendingCount == totalDecisions {
			return "PENDING"
		} else {
			return "PARTIALLY_APPROVED"
		}
	}
}

func (s *SmartContract) UpdateValidDecisions(ctx contractapi.TransactionContextInterface, documentID string, newDecisionsJSON string, invokerId string) error {
	// Validate inputs
	if err := s.validateInputs(map[string]string{
		"documentID":       documentID,
		"newDecisionsJSON": newDecisionsJSON,
		"invokerId":        invokerId,
	}); err != nil {
		return err
	}

	date, timeStr, err := s.getCurrentDateTime(ctx)
	if err != nil {
		return err
	}

	// 1. Retrieve the document
	data, err := ctx.GetStub().GetState(documentID)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", documentID)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	// 2. Security Check: Verify invokerId is a Privileged Editor
	if !s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, documentID)
	}

	// 3. Parse new valid decisions
	var newDecisions []string
	if err := json.Unmarshal([]byte(newDecisionsJSON), &newDecisions); err != nil {
		return fmt.Errorf("invalid newDecisions JSON: %v", err)
	}

	if len(newDecisions) == 0 {
		return fmt.Errorf("at least one valid decision must be specified")
	}

	// 4. Store old decisions for comparison
	oldDecisions := append([]string{}, doc.ValidDecisions...)

	// 5. Update valid decisions and CREATE NEW VERSION (per your request)
	doc.ValidDecisions = newDecisions
	doc.LastModifiedDate = date
	doc.LastModifiedTime = timeStr

	// Increment version for valid decisions changes
	doc.LatestVersion++

	// Get current hash from latest version
	currentHash := "DECISIONS_UPDATED"
	if len(doc.Versions) > 0 {
		currentHash = doc.Versions[len(doc.Versions)-1].Hash
	}

	// Create snapshot of current approvals
	currentApprovals := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		currentApprovals[k] = v
	}

	// Create new version entry
	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             currentHash,
		Submitter:        invokerId,
		Date:             date,
		Time:             timeStr,
		ApprovalsMap:     currentApprovals,
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: false,
		DecisionsChanged: true,
	}
	doc.Versions = append(doc.Versions, newVersion)

	// 7. Persist the updated document
	updatedDocBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	if err = ctx.GetStub().PutState(documentID, updatedDocBytes); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	// 7. Emit event
	description := fmt.Sprintf("Valid decisions updated for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"oldDecisions": oldDecisions,
		"newDecisions": newDecisions,
	}

	return s.emitEvent(ctx, "ValidDecisionsUpdated", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// Rich Query Functions (CouchDB only)

// GetDocumentsByDateRange returns documents created within a date range, sorted by date
func (s *SmartContract) GetDocumentsByDateRange(ctx contractapi.TransactionContextInterface, startDate, endDate string) ([]*Document, error) {
	if startDate == "" || endDate == "" {
		return nil, fmt.Errorf("startDate and endDate cannot be empty")
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"CreatedDate": {
				"$gte": "%s",
				"$lte": "%s"
			}
		},
		"sort": [
			{"CreatedDate": "desc"},
			{"CreatedTime": "desc"}
		]
	}`, startDate, endDate)

	return s.getQueryResult(ctx, queryString)
}

// GetRecentDocuments returns the most recently modified documents
func (s *SmartContract) GetRecentDocuments(ctx contractapi.TransactionContextInterface, limit string) ([]*Document, error) {
	limitInt := 10 // default
	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			limitInt = parsed
		}
	}

	queryString := fmt.Sprintf(`{
		"selector": {},
		"sort": [
			{"LastModifiedDate": "desc"},
			{"LastModifiedTime": "desc"}
		],
		"limit": %d
	}`, limitInt)

	return s.getQueryResult(ctx, queryString)
}

// GetDocumentsByUploaderSorted returns documents by uploader, sorted by creation date
func (s *SmartContract) GetDocumentsByUploaderSorted(ctx contractapi.TransactionContextInterface, uploader, sortOrder string) ([]*Document, error) {
	if uploader == "" {
		return nil, fmt.Errorf("uploader cannot be empty")
	}

	order := "desc"
	if sortOrder == "asc" {
		order = "asc"
	}

	queryString := fmt.Sprintf(`{
		"selector": {
			"Uploader": "%s"
		},
		"sort": [
			{"CreatedDate": "%s"},
			{"CreatedTime": "%s"}
		]
	}`, uploader, order, order)

	return s.getQueryResult(ctx, queryString)
}

// GetDocumentsWithPendingApprovals returns documents that have pending approvals
func (s *SmartContract) GetDocumentsWithPendingApprovals(ctx contractapi.TransactionContextInterface) ([]*Document, error) {
	// This is a complex query - we'll get all documents and filter in code
	// since CouchDB can't easily query nested map structures
	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var pendingDocs []*Document
	for _, doc := range allDocs {
		hasPending := false
		for _, decision := range doc.ApprovalsMap {
			if decision.Status == Pending {
				hasPending = true
				break
			}
		}
		if hasPending {
			pendingDocs = append(pendingDocs, doc)
		}
	}

	return pendingDocs, nil
}

// GetDocumentStats returns statistics about documents
func (s *SmartContract) GetDocumentStats(ctx contractapi.TransactionContextInterface) (map[string]interface{}, error) {
	allDocs, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"totalDocuments":    len(allDocs),
		"totalVersions":     0,
		"uploaderCounts":    make(map[string]int),
		"statusCounts":      make(map[string]int),
		"avgVersionsPerDoc": 0.0,
	}

	totalVersions := 0
	uploaderCounts := make(map[string]int)
	statusCounts := map[string]int{
		"PENDING":            0,
		"PARTIALLY_APPROVED": 0,
		"FULLY_APPROVED":     0,
		"REJECTED":           0,
	}

	for _, doc := range allDocs {
		totalVersions += len(doc.Versions)
		uploaderCounts[doc.Uploader]++

		// Determine document status
		decisionCounts := make(map[string]int)
		for _, decision := range doc.ApprovalsMap {
			decisionCounts[string(decision.Status)]++
		}

		status := s.determineDocumentStatus(decisionCounts, doc.ValidDecisions)
		statusCounts[status]++
	}

	stats["totalVersions"] = totalVersions
	stats["uploaderCounts"] = uploaderCounts
	stats["statusCounts"] = statusCounts

	if len(allDocs) > 0 {
		stats["avgVersionsPerDoc"] = float64(totalVersions) / float64(len(allDocs))
	}

	return stats, nil
}

// Helper function to execute CouchDB queries
func (s *SmartContract) getQueryResult(ctx contractapi.TransactionContextInterface, queryString string) ([]*Document, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var documents []*Document
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var document Document
		err = json.Unmarshal(queryResponse.Value, &document)
		if err != nil {
			return nil, err
		}

		// Apply defensive initialization to ensure arrays and pointers are never null
		s.ensureDocumentArraysInitialized(&document)

		// WORKFLOW FIX: Update compatibility ApprovalsMap to show current stage only
		if document.Workflow.Enabled {
			s.updateCompatibilityApprovalsMap(&document)
		}

		// SCHEMA FIX: Ensure all versions have WorkflowSnapshot (never nil)
		for i := range document.Versions {
			if document.Versions[i].WorkflowSnapshot == nil {
				// Create empty workflow snapshot for schema compliance with required fields
				document.Versions[i].WorkflowSnapshot = &WorkflowSnapshot{
					Enabled:         false,
					CurrentStage:    0,
					CompletedStages: []int{},
					TotalStages:     0,
					CompletionDate:  "Not Completed", // User-friendly status message
					CompletionTime:  "Not Completed", // User-friendly status message
				}
			} else {
				// Ensure existing snapshots have user-friendly completion status if not completed
				if document.Versions[i].WorkflowSnapshot.CompletionDate == "" {
					document.Versions[i].WorkflowSnapshot.CompletionDate = "Not Completed"
				}
				if document.Versions[i].WorkflowSnapshot.CompletionTime == "" {
					document.Versions[i].WorkflowSnapshot.CompletionTime = "Not Completed"
				}
			}
		}

		documents = append(documents, &document)
	}

	return documents, nil
}
