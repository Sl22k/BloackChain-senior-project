package chaincode

import (
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Core document and version related data structures
type ApproverStatus string

const (
	Pending  ApproverStatus = "PENDING"
	Approved ApproverStatus = "APPROVED"
	Rejected ApproverStatus = "REJECTED"
)

type Decision struct {
	Status    ApproverStatus `json:"Status"`
	Comment   string         `json:"Comment"`
	Timestamp string         `json:"Timestamp"` // RFC3339 format: "2023-09-13T14:30:15+03:00"
}

type DocumentVersion struct {
	Version          int                 `json:"Version"`
	Hash             string              `json:"Hash"`
	Submitter        string              `json:"Submitter"`
	Timestamp        string              `json:"Timestamp"`        // RFC3339 format: "2023-09-13T14:30:15+03:00"
	ApprovalsMap     map[string]Decision `json:"ApprovalsMap"`     // Approvals state at this version
	ValidDecisions   []string            `json:"ValidDecisions"`   // Valid decisions at this version
	ApproversChanged bool                `json:"ApproversChanged"` // Flag if approvers were modified
	DecisionsChanged bool                `json:"DecisionsChanged"` // Flag if valid decisions were modified
	ContentChanged   bool                `json:"ContentChanged"`   // Flag if content hash was modified
	// NEW: Ownership change tracking
	ChangeType       string              `json:"ChangeType"`       // "content", "approvers", "ownership", "decisions", or combinations like "content+approvers"
	PreviousOwner    string              `json:"PreviousOwner"`    // Previous owner for ownership transfers
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
	Timestamp  string `json:"Timestamp"`  // RFC3339 format: "2023-09-13T14:30:15+03:00"
	Comment    string `json:"Comment"`
}

// NEW: Workflow snapshot for version history - FIX FOR BUG #2
type WorkflowSnapshot struct {
	Enabled            bool   `json:"Enabled"`
	CurrentStage       int    `json:"CurrentStage"`
	CompletedStages    []int  `json:"CompletedStages"`
	TotalStages        int    `json:"TotalStages"`
	CompletionTimestamp string `json:"CompletionTimestamp"` // RFC3339 format: "2023-09-13T14:30:15+03:00"
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

// Deadline configuration for documents or workflow stages
type DeadlineConfig struct {
	Enabled            bool                `json:"Enabled"`
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
	DocumentId           string                         `json:"DocumentId"`
	CurrentStageStatus   string                         `json:"CurrentStageStatus"`
	CurrentStageDeadline string                         `json:"CurrentStageDeadline"`
	TimeRemaining        int64                          `json:"TimeRemaining"`
	StageStatuses        map[string]StageDeadlineStatus `json:"StageStatuses"`
	WarningsTriggered    []DeadlineWarning              `json:"WarningsTriggered"`
	BreachesRecorded     []DeadlineBreach               `json:"BreachesRecorded"`
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
	// NEW: Ownership transfer support
	PendingOwnershipTransfer OwnershipTransferRequest `json:"PendingOwnershipTransfer"` // Pending ownership transfer request
	// NEW: Version workflow history - FIX FOR BUG #2
	VersionWorkflows     map[string]*WorkflowConfig `json:"VersionWorkflows"`     // Complete workflow state per version
	DeadlineConfig       DeadlineConfig             `json:"DeadlineConfig"`       // Document deadline configuration
	DeadlineStatus       DeadlineStatus             `json:"DeadlineStatus"`       // Real-time deadline status
	CreatedTimestamp     string                     `json:"CreatedTimestamp"`     // RFC3339 format: "2023-09-13T14:30:15+03:00"
	LastModifiedTimestamp string                     `json:"LastModifiedTimestamp"` // RFC3339 format: "2023-09-13T14:30:15+03:00"
}

type StatusResponse struct {
	ID                string              `json:"ID"`
	Title             string              `json:"Title"`
	Description       string              `json:"Description"`
	LatestVersion     int                 `json:"LatestVersion"`
	// Versions field removed - use GetDocumentHistory() for version history
	Uploader          string              `json:"Uploader"`
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"`
	DecisionCounts    map[string]int      `json:"DecisionCounts"`
	// API-compatible individual count fields
	ApprovedCount    int            `json:"ApprovedCount"`
	RejectedCount    int            `json:"RejectedCount"`
	PendingCount     int            `json:"PendingCount"`
	ValidDecisions   []string       `json:"ValidDecisions"`
	Workflow         WorkflowConfig `json:"Workflow"`
	CurrentStageInfo *WorkflowStage `json:"CurrentStageInfo,omitempty"`
	// NEW: Ownership transfer support
	PendingOwnershipTransfer OwnershipTransferRequest `json:"PendingOwnershipTransfer"`
	// VersionWorkflows field removed - use GetDocumentHistory() for version workflows
	// User-friendly workflow progress information
	WorkflowProgress     *WorkflowProgressOverview `json:"WorkflowProgress,omitempty"`
	DeadlineConfig       DeadlineConfig            `json:"DeadlineConfig"`
	DeadlineStatus       DeadlineStatus            `json:"DeadlineStatus"`
	CreatedTimestamp     string                    `json:"CreatedTimestamp"`     // RFC3339 format: "2023-09-13T14:30:15+03:00"
	LastModifiedTimestamp string                    `json:"LastModifiedTimestamp"` // RFC3339 format: "2023-09-13T14:30:15+03:00"
}

type EventPayload struct {
	DocumentID  string                 `json:"documentID"`
	Version     int                    `json:"version"`
	EventType   string                 `json:"eventType"`
	Actor       string                 `json:"actor"`
	Timestamp   string                 `json:"timestamp"`  // RFC3339 format: "2023-09-13T14:30:15+03:00"
	Details     map[string]interface{} `json:"details"`
	Description string                 `json:"description"`
}

// NEW: Ownership Transfer structures

// OwnershipTransferRequest represents a pending ownership transfer request
type OwnershipTransferRequest struct {
	RequestedBy      string `json:"requestedBy"`
	NewOwner         string `json:"newOwner"`
	RequestTimestamp string `json:"requestTimestamp"`
	ExpirationTime   string `json:"expirationTime"`
	ExpirationDays   int    `json:"expirationDays"`
	Message          string `json:"message"`
}

// NEW: Enhanced versioning structures for unified modification approach

// NewVersionState represents the complete desired state for a new version - UNIFIED MODEL
type NewVersionState struct {
	ContentHash      string                    `json:"contentHash"`      // New content hash (always required)
	StageUpdates     map[string]StageUpdate    `json:"stageUpdates"`     // Stage-specific updates (key = stage number as string)
	ValidDecisions   []string                  `json:"validDecisions"`   // Complete new valid decisions list
	ResetApprovals   bool                      `json:"resetApprovals"`   // User choice
	ChangeReason     string                    `json:"changeReason"`     // Required explanation
}

// StageUpdate represents changes to a specific workflow stage
type StageUpdate struct {
	Approvers       []string `json:"approvers"`        // New approver list for this stage
	RequiredCount   int      `json:"requiredCount"`    // Required approvals (0 = all approvers)
	AutoAdvance     bool     `json:"autoAdvance"`      // Auto-advance setting
}

// DetectedChanges represents what actually changed between versions
type DetectedChanges struct {
	ContentChanged     bool     `json:"contentChanged"`
	ApproversAdded     []string `json:"approversAdded"`
	ApproversRemoved   []string `json:"approversRemoved"`
	DecisionsAdded     []string `json:"decisionsAdded"`
	DecisionsRemoved   []string `json:"decisionsRemoved"`
	NoChangesDetected  bool     `json:"noChangesDetected"`
	// Enhanced reconsideration rules
	AllowReconsideration bool     `json:"allowReconsideration"`
	NewDecisionsOnly     []string `json:"newDecisionsOnly"`
}

// ReconsiderationInfo tells frontend what reconsideration options are available
type ReconsiderationInfo struct {
	Allowed            bool     `json:"allowed"`
	AvailableDecisions []string `json:"availableDecisions"`
	Reason             string   `json:"reason"`
}

// NewVersionResponse provides complete information after version creation
type NewVersionResponse struct {
	Success             bool                `json:"success"`
	Version             int                 `json:"version"`
	ChangesDetected     DetectedChanges     `json:"changesDetected"`
	ReconsiderationInfo ReconsiderationInfo `json:"reconsiderationInfo"`
}

// Enhanced ReturnDocumentToStage structures

// DeadlineRollbackOptions specifies how to handle deadline conflicts during rollback
type DeadlineRollbackOptions struct {
	RemoveConflicts bool            `json:"removeConflicts"` // Remove conflicting deadlines completely
	NewDeadlines    map[string]int  `json:"newDeadlines"`    // Set new deadlines: stage number -> hours from now
}

// DeadlineConflict represents a deadline that conflicts with rollback
type DeadlineConflict struct {
	StageNumber  int    `json:"stageNumber"`
	Deadline     string `json:"deadline"`     // RFC3339 timestamp
	HoursOverdue int    `json:"hoursOverdue"` // How many hours past deadline
	Status       string `json:"status"`       // Current deadline status
}

// ReturnToStageResponse provides complete information after rollback
type ReturnToStageResponse struct {
	Success              bool                       `json:"success"`
	ReturnedToStage      int                        `json:"returnedToStage"`
	ResetStages          []int                      `json:"resetStages"`
	RemovedFromCompleted []int                      `json:"removedFromCompleted"`
	ApprovalsCleared     int                        `json:"approvalsCleared"`
	NewCurrentStage      int                        `json:"newCurrentStage"`
	DeadlineConflicts    []DeadlineConflict         `json:"deadlineConflicts"`
	DeadlineChanges      map[string]string          `json:"deadlineChanges"`
	StageHistoryAdded    StageTransition            `json:"stageHistoryAdded"`
}

// Enhanced ConfigureDocumentWorkflow structures

// WorkflowConfigurationRequest represents the complete workflow reconfiguration request
type WorkflowConfigurationRequest struct {
	StageOperations       map[string]StageOperation `json:"stageOperations"`       // stage number -> operation (1-based)
	NewStages            []WorkflowStage           `json:"newStages"`             // stages to add at the end
	StagesToRemove       []int                     `json:"stagesToRemove"`        // stage numbers to remove (1-based)
	KeepExistingApprovals bool                      `json:"keepExistingApprovals"` // user choice for approval preservation
	ConfigurationReason  string                    `json:"configurationReason"`   // required explanation
	DeadlineUpdates      map[string]int            `json:"deadlineUpdates"`       // stage number -> hours from now
}

// StageOperation represents an operation to perform on a specific stage
type StageOperation struct {
	Action       string         `json:"action"`       // "update", "keep", "remove"
	UpdatedStage *WorkflowStage `json:"updatedStage"` // new stage configuration (for "update" action)
}

// WorkflowConfigurationResponse provides comprehensive information after workflow reconfiguration
type WorkflowConfigurationResponse struct {
	Success             bool            `json:"success"`
	PreviousStageCount  int             `json:"previousStageCount"`
	NewStageCount       int             `json:"newStageCount"`
	StagesAdded         []int           `json:"stagesAdded"`         // stage numbers that were added
	StagesRemoved       []int           `json:"stagesRemoved"`       // stage numbers that were removed
	StagesModified      []int           `json:"stagesModified"`      // stage numbers that were modified
	ApprovalsReset      bool            `json:"approvalsReset"`      // whether approvals were reset
	CurrentStageReset   bool            `json:"currentStageReset"`   // whether current stage was reset to 1
	CompletedStagesCleared bool         `json:"completedStagesCleared"` // whether completed stages were cleared
	StageHistoryAdded   StageTransition `json:"stageHistoryAdded"`   // audit trail entry
	DeadlineChanges     map[string]string `json:"deadlineChanges"`   // deadline changes made
}

type SmartContract struct {
	contractapi.Contract
}