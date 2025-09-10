package chaincode

import (
	"encoding/json"
	"fmt"
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
	Date    string         `json:"Date,omitempty"`
	Time    string         `json:"Time,omitempty"`
}

type DocumentVersion struct {
	Version          int                 `json:"Version"`
	Hash             string              `json:"Hash"`
	Submitter        string              `json:"Submitter"`
	Timestamp        int64               `json:"Timestamp"`
	Date             string              `json:"Date"`
	Time             string              `json:"Time"`
	ApprovalsMap     map[string]Decision `json:"ApprovalsMap"`
	ValidDecisions   []string            `json:"ValidDecisions"`
	ApproversChanged bool                `json:"ApproversChanged"`
	DecisionsChanged bool                `json:"DecisionsChanged"`
	WorkflowSnapshot *WorkflowConfig     `json:"WorkflowSnapshot"`
}

type Document struct {
	ID                 string                       `json:"ID"`
	LatestVersion      int                          `json:"LatestVersion"`
	Versions           []DocumentVersion            `json:"Versions"`
	Uploader           string                       `json:"Uploader"`
	PrivilegedEditors  []string                     `json:"PrivilegedEditors"`
	Editors            []string                     `json:"Editors"`
	ApprovalsMap       map[string]Decision          `json:"ApprovalsMap"`
	ValidDecisions     []string                     `json:"ValidDecisions"`
	ApprovedCount      int                          `json:"ApprovedCount"`
	RejectedCount      int                          `json:"RejectedCount"`
	PendingCount       int                          `json:"PendingCount"`
	LastModifiedDate   string                       `json:"LastModifiedDate,omitempty"`
	LastModifiedTime   string                       `json:"LastModifiedTime,omitempty"`
	Workflow           WorkflowConfig               `json:"Workflow,omitempty"`
	VersionWorkflows   map[string]*WorkflowConfig   `json:"VersionWorkflows"`
}

// NOTE: The following are placeholder structs and functions.
// You need to provide the actual implementation for them.

// WorkflowConfig defines the configuration for a workflow
type WorkflowConfig struct {
	Enabled         bool               `json:"Enabled"`
	Stages          []WorkflowStage    `json:"Stages"`
	CurrentStage    int                `json:"CurrentStage"`
	CompletedStages []int              `json:"CompletedStages"`
	StageHistory    []StageTransition  `json:"StageHistory"`
}

// WorkflowStage defines a stage in a workflow
type WorkflowStage struct {
	Name      string              `json:"Name"`
	Approvers []string            `json:"Approvers"`
	Approvals map[string]Decision `json:"Approvals"`
}

// StageTransition represents a transition in a workflow
type StageTransition struct {
	FromStage  int    `json:"FromStage"`
	ToStage    int    `json:"ToStage"`
	Transition string `json:"Transition"`
	Actor      string `json:"Actor"`
	Date       string `json:"Date"`
	Time       string `json:"Time"`
	Comment    string `json:"Comment"`
}

type StatusResponse struct {
	ID                string              `json:"ID"`
	LatestVersion     int                 `json:"LatestVersion"`
	Versions          []DocumentVersion   `json:"Versions"`
	Uploader          string              `json:"Uploader"`
	PrivilegedEditors []string            `json:"PrivilegedEditors"`
	Editors           []string            `json:"Editors"`
	ApprovalsMap      map[string]Decision `json:"ApprovalsMap"`
	DecisionCounts    map[string]int      `json:"DecisionCounts"`
	ValidDecisions    []string            `json:"ValidDecisions"`
	ApprovedCount     int                 `json:"ApprovedCount"`
	RejectedCount     int                 `json:"RejectedCount"`
	PendingCount      int                 `json:"PendingCount"`
}

type SmartContract struct {
	contractapi.Contract
}

func (s *SmartContract) validateInputs(inputs map[string]string) error {
	// TODO: Implement your input validation logic here
	return nil
}

func (s *SmartContract) getCurrentDateTime(ctx contractapi.TransactionContextInterface) (string, string, error) {
	// TODO: Implement your logic to get current date and time
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", "", err
	}
	t := time.Unix(txTimestamp.GetSeconds(), int64(txTimestamp.GetNanos()))
	return t.Format("2006-01-02"), t.Format("15:04:05"), nil
}

func (s *SmartContract) isUserInList(user string, list []string) bool {
	// TODO: Implement your logic to check if a user is in a list
	for _, item := range list {
		if item == user {
			return true
		}
	}
	return false
}

func (s *SmartContract) deepCopyWorkflowConfig(config WorkflowConfig) WorkflowConfig {
	// TODO: Implement your logic to deep copy a workflow config
	return config
}

func (s *SmartContract) createWorkflowSnapshot(config WorkflowConfig, date, timeStr string) *WorkflowConfig {
	// TODO: Implement your logic to create a workflow snapshot
	return &config
}

func (s *SmartContract) initializeStageApprovals(stages []WorkflowStage, date, timeStr string) []WorkflowStage {
	// TODO: Implement your logic to initialize stage approvals
	return stages
}

func (s *SmartContract) updateCompatibilityApprovalsMap(doc *Document) {
	// TODO: Implement your logic to update the compatibility approvals map
}

func (s *SmartContract) emitEvent(ctx contractapi.TransactionContextInterface, eventName, submitter, description, docID string, latestVersion int, details map[string]interface{}) error {
	// TODO: Implement your event emission logic here
	return nil
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

func (s *SmartContract) SubmitDocument(ctx contractapi.TransactionContextInterface, id, hash, uploader, approversJSON, validDecisionsJSON, invokerId string, resetApprovals bool) error {
	exists, err := s.DocumentExists(ctx, id)
	if err != nil {
		return err
	}

	var doc Document
	if exists {
		data, err := ctx.GetStub().GetState(id)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return err
		}

		if doc.Versions[doc.LatestVersion-1].Hash == hash {
			return fmt.Errorf("new version hash is the same as the latest version hash")
		}

		doc.LatestVersion++
		if resetApprovals {
			for approver := range doc.ApprovalsMap {
				doc.ApprovalsMap[approver] = Decision{Status: Pending, Comment: ""}
			}
		}
	} else {
		var approvers []string
		if err := json.Unmarshal([]byte(approversJSON), &approvers); err != nil {
			return fmt.Errorf("invalid approvers JSON: %v", err)
		}
		var validDecisions []string
		if err := json.Unmarshal([]byte(validDecisionsJSON), &validDecisions); err != nil {
			return fmt.Errorf("invalid validDecisions JSON: %v", err)
		}
		if validDecisions == nil {
			validDecisions = []string{}
		}

		approvalsMap := make(map[string]Decision)
		for _, approver := range approvers {
			approvalsMap[approver] = Decision{Status: Pending, Comment: ""}
		}

		doc = Document{
			ID:                id,
			LatestVersion:     1,
			Uploader:          uploader,
			PrivilegedEditors: []string{uploader},
			Editors:           []string{uploader},
			ValidDecisions:    validDecisions,
			ApprovalsMap:      approvalsMap,
			ApprovedCount:     0,
			RejectedCount:     0,
			PendingCount:      len(approvers),
		}
	}

	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	newVersion := DocumentVersion{
		Version:   doc.LatestVersion,
		Hash:      hash,
		Submitter: uploader,
		Timestamp: timestamp.GetSeconds(),
	}
	doc.Versions = append(doc.Versions, newVersion)

	hashKey := fmt.Sprintf("hash->%s", hash)
	if err := ctx.GetStub().PutState(hashKey, []byte(id)); err != nil {
		return fmt.Errorf("failed to create hash-to-ID index: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, data)
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

	// STEP 3: Create snapshot of FINAL state before reset - preserves approval trail
	// This captures the completed approval state for the historical version
	finalApprovalState := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		finalApprovalState[k] = v
	}

	// STEP 4: Reset approvals and workflow for new version - FIX FOR BUG #2
	if resetApprovals {
		// Reset compatibility ApprovalsMap
		for approver := range doc.ApprovalsMap {
			doc.ApprovalsMap[approver] = Decision{
				Status:  Pending,
				Comment: "",
				Date:    date,
				Time:    timeStr,
			}
		}

		// Reset workflow state for new version - FIX FOR BUG #2  
		if doc.Workflow.Enabled {
			// Reset workflow to initial state
			doc.Workflow.CurrentStage = 1
			doc.Workflow.CompletedStages = []int{}

			// Reset all stage approvals to pending - FIX FOR BUG #1
			doc.Workflow.Stages = s.initializeStageApprovals(doc.Workflow.Stages, date, timeStr)

			// Update compatibility ApprovalsMap from stage approvals - FIX FOR BUG #1
			s.updateCompatibilityApprovalsMap(&doc)

			// Add version reset transition to history
			doc.Workflow.StageHistory = append(doc.Workflow.StageHistory, StageTransition{
				FromStage:  0,
				ToStage:    1,
				Transition: "version_reset",
				Actor:      submitter,
				Date:       date,
				Time:       timeStr,
				Comment:    fmt.Sprintf("Workflow reset for version %d submission", doc.LatestVersion),
			})
		}
	}

	// Add the new version to the history - FIX FOR BUG #2
	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:             hash,
		Submitter:        submitter,
		Date:             date,
		Time:             timeStr,
		ApprovalsMap:     finalApprovalState, // Use the snapshot taken before reset
		ValidDecisions:   append([]string{}, doc.ValidDecisions...),
		ApproversChanged: false,
		DecisionsChanged: false,
		WorkflowSnapshot: nil, // Will be populated as workflow progresses
	}
	// Get the timestamp from the transaction context
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %v", err)
	}
	newVersion.Timestamp = timestamp.GetSeconds()
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

func (s *SmartContract) ApproveDocument(ctx contractapi.TransactionContextInterface, id, approver, decision, comment string) error {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}

	decisionInfo, ok := doc.ApprovalsMap[approver]
	if !ok {
		return fmt.Errorf("%s not authorized to decide", approver)
	}

	if decisionInfo.Status != Pending {
		return fmt.Errorf("%s already decided", approver)
	}

	validDecision := false
	for _, d := range doc.ValidDecisions {
		if decision == d {
			validDecision = true
			break
		}
	}
	if !validDecision {
		return fmt.Errorf("invalid decision '%s' for this document", decision)
	}

	oldStatus := decisionInfo.Status
	decisionInfo.Status = ApproverStatus(decision)
	decisionInfo.Comment = comment
	doc.ApprovalsMap[approver] = decisionInfo

	if oldStatus == Pending {
		doc.PendingCount--
	}
	if ApproverStatus(decision) == Approved {
		doc.ApprovedCount++
	} else if ApproverStatus(decision) == Rejected {
		doc.RejectedCount++
	}

	updated, _ := json.Marshal(doc)
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) QueryDocumentStatus(ctx contractapi.TransactionContextInterface, id string) (*StatusResponse, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Populate new fields with default values for backward compatibility
	if doc.LastModifiedDate == "" {
		doc.LastModifiedDate = "N/A"
	}
	if doc.LastModifiedTime == "" {
		doc.LastModifiedTime = "N/A"
	}
	if doc.VersionWorkflows == nil {
		doc.VersionWorkflows = make(map[string]*WorkflowConfig)
	}
	if doc.Workflow.CompletedStages == nil {
		doc.Workflow.CompletedStages = []int{}
	}
	if doc.Workflow.StageHistory == nil {
		doc.Workflow.StageHistory = []StageTransition{}
	}
	if doc.Workflow.Stages == nil {
		doc.Workflow.Stages = []WorkflowStage{}
	}

	for i := range doc.Versions {
		if doc.Versions[i].ApprovalsMap == nil {
			doc.Versions[i].ApprovalsMap = make(map[string]Decision)
		} else {
			newVersionApprovalsMap := make(map[string]Decision)
			for k, v := range doc.Versions[i].ApprovalsMap {
				if v.Date == "" {
					v.Date = "N/A"
				}
				if v.Time == "" {
					v.Time = "N/A"
				}
				newVersionApprovalsMap[k] = v
			}
			doc.Versions[i].ApprovalsMap = newVersionApprovalsMap
		}

		if doc.Versions[i].ValidDecisions == nil {
			doc.Versions[i].ValidDecisions = []string{}
		}
		if doc.Versions[i].WorkflowSnapshot == nil {
			doc.Versions[i].WorkflowSnapshot = &WorkflowConfig{
				CompletedStages: []int{},
				StageHistory:    []StageTransition{},
				Stages:          []WorkflowStage{},
			}
		}
	}
	
	newApprovalsMap := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		if v.Date == "" {
			v.Date = "N/A"
		}
		if v.Time == "" {
			v.Time = "N/A"
		}
		newApprovalsMap[k] = v
	}
	doc.ApprovalsMap = newApprovalsMap

	if doc.ValidDecisions == nil {
		doc.ValidDecisions = []string{}
	}

	resp := &StatusResponse{
		ID:                doc.ID,
		LatestVersion:     doc.LatestVersion,
		Versions:          doc.Versions,
		Uploader:          doc.Uploader,
		PrivilegedEditors: doc.PrivilegedEditors,
		Editors:           doc.Editors,
		ApprovalsMap:      doc.ApprovalsMap,
		ValidDecisions:    doc.ValidDecisions,
		ApprovedCount:     doc.ApprovedCount,
		RejectedCount:     doc.RejectedCount,
		PendingCount:      doc.PendingCount,
		DecisionCounts:    make(map[string]int),
	}

	return resp, nil
}

func (s *SmartContract) AddEditor(ctx contractapi.TransactionContextInterface, id, newEditor, invokerId string) error {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	isPrivileged := false
	for _, editor := range doc.PrivilegedEditors {
		if editor == invokerId {
			isPrivileged = true
			break
		}
	}
	if !isPrivileged {
		return fmt.Errorf("submitter %s is not a privileged editor for document %s", invokerId, id)
	}

	doc.Editors = append(doc.Editors, newEditor)

	updated, _ := json.Marshal(doc)
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) AddPrivilegedEditor(ctx contractapi.TransactionContextInterface, id, newEditor, invokerId string) error {
	data, err := ctx.GetStub().GetState(id)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", id)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	if doc.Uploader != invokerId {
		return fmt.Errorf("only the original uploader can add privileged editors")
	}

	doc.PrivilegedEditors = append(doc.PrivilegedEditors, newEditor)
	doc.Editors = append(doc.Editors, newEditor)

	updated, _ := json.Marshal(doc)
	return ctx.GetStub().PutState(id, updated)
}

func (s *SmartContract) UpdateValidDecisions(ctx contractapi.TransactionContextInterface, documentID string, validDecisionsJSON string, invokerId string) error {
	data, err := ctx.GetStub().GetState(documentID)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", documentID)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	isPrivilegedEditor := false
	for _, editor := range doc.PrivilegedEditors {
		if editor == invokerId {
			isPrivilegedEditor = true
			break
		}
	}
	if !isPrivilegedEditor {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, documentID)
	}

	var newValidDecisions []string
	if err := json.Unmarshal([]byte(validDecisionsJSON), &newValidDecisions); err != nil {
		return fmt.Errorf("invalid validDecisions JSON: %v", err)
	}

	doc.ValidDecisions = newValidDecisions

	updatedDocBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(documentID, updatedDocBytes)
}

func (s *SmartContract) UpdateDocumentApprovers(ctx contractapi.TransactionContextInterface, documentID string, newApproversJSON string, invokerId string) error {
	data, err := ctx.GetStub().GetState(documentID)
	if err != nil || data == nil {
		return fmt.Errorf("document %s not found", documentID)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	isPrivilegedEditor := false
	for _, editor := range doc.PrivilegedEditors {
		if editor == invokerId {
			isPrivilegedEditor = true
			break
		}
	}
	if !isPrivilegedEditor {
		return fmt.Errorf("invoker %s is not a privileged editor for document %s", invokerId, documentID)
	}

	var newApprovers []string
	if err := json.Unmarshal([]byte(newApproversJSON), &newApprovers); err != nil {
		return fmt.Errorf("invalid newApprovers JSON: %v", err)
	}

	updatedApprovalsMap := make(map[string]Decision)
	for _, approver := range newApprovers {
		if existingDecision, ok := doc.ApprovalsMap[approver]; ok {
			updatedApprovalsMap[approver] = existingDecision
		} else {
			updatedApprovalsMap[approver] = Decision{Status: Pending, Comment: ""}
		}
	}
	doc.ApprovalsMap = updatedApprovalsMap

	updatedDocBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(documentID, updatedDocBytes)
}

func (s *SmartContract) GetDocumentIdByHash(ctx contractapi.TransactionContextInterface, hash string) (string, error) {
	hashKey := fmt.Sprintf("hash->%s", hash)
	idBytes, err := ctx.GetStub().GetState(hashKey)
	if err != nil || idBytes == nil {
		return "", fmt.Errorf("no document found with hash %s", hash)
	}
	return string(idBytes), nil
}

func (s *SmartContract) DocumentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, err
	}
	return data != nil, nil
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

		if len(queryResponse.Key) > 5 && string(queryResponse.Key[:5]) == "hash->" {
			continue
		}

		var document Document
		err = json.Unmarshal(queryResponse.Value, &document)
		if err != nil {
			// For backward compatibility, we can try to unmarshal into a map first
			// to avoid errors with old document structures.
			// However, for now, we will just log the error and continue.
			// log.Printf("Error unmarshalling document %s: %v", queryResponse.Key, err)
			continue
		}

		// Populate new fields with default values for backward compatibility
		if document.LastModifiedDate == "" {
			document.LastModifiedDate = "N/A"
		}
		if document.LastModifiedTime == "" {
			document.LastModifiedTime = "N/A"
		}
		if document.VersionWorkflows == nil {
			document.VersionWorkflows = make(map[string]*WorkflowConfig)
		}
		if document.Workflow.CompletedStages == nil {
			document.Workflow.CompletedStages = []int{}
		}
		if document.Workflow.StageHistory == nil {
			document.Workflow.StageHistory = []StageTransition{}
		}
		if document.Workflow.Stages == nil {
			document.Workflow.Stages = []WorkflowStage{}
		}

		for i := range document.Versions {
			if document.Versions[i].ApprovalsMap == nil {
				document.Versions[i].ApprovalsMap = make(map[string]Decision)
			} else {
				newVersionApprovalsMap := make(map[string]Decision)
				for k, v := range document.Versions[i].ApprovalsMap {
					if v.Date == "" {
						v.Date = "N/A"
					}
					if v.Time == "" {
						v.Time = "N/A"
					}
					newVersionApprovalsMap[k] = v
				}
				document.Versions[i].ApprovalsMap = newVersionApprovalsMap
			}

			if document.Versions[i].ValidDecisions == nil {
				document.Versions[i].ValidDecisions = []string{}
			}
			if document.Versions[i].WorkflowSnapshot == nil {
				document.Versions[i].WorkflowSnapshot = &WorkflowConfig{
					CompletedStages: []int{},
					StageHistory:    []StageTransition{},
					Stages:          []WorkflowStage{},
				}
			}
		}
		
		newApprovalsMap := make(map[string]Decision)
		for k, v := range document.ApprovalsMap {
			if v.Date == "" {
				v.Date = "N/A"
			}
			if v.Time == "" {
				v.Time = "N/A"
			}
			newApprovalsMap[k] = v
		}
		document.ApprovalsMap = newApprovalsMap

		if document.ValidDecisions == nil {
			document.ValidDecisions = []string{}
		}
		documents = append(documents, &document)
	}

	return documents, nil
}

func (s *SmartContract) GetHistory(ctx contractapi.TransactionContextInterface, documentID string) ([]map[string]interface{}, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(documentID)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var history []map[string]interface{}
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var record map[string]interface{}
		if err := json.Unmarshal(response.Value, &record); err != nil {
			return nil, err
		}

		historyEntry := map[string]interface{}{
			"TxId":      response.TxId,
			"Timestamp": time.Unix(response.Timestamp.GetSeconds(), int64(response.Timestamp.GetNanos())).Format(time.RFC3339),
			"IsDelete":  response.IsDelete,
			"Value":     record,
		}
		history = append(history, historyEntry)
	}

	return history, nil
}

func (s *SmartContract) QueryDocumentsByEditor(ctx contractapi.TransactionContextInterface, editorID string) ([]*Document, error) {
	allDocuments, err := s.GetAllDocuments(ctx)
	if err != nil {
		return nil, err
	}

	var editableDocuments []*Document
	for _, doc := range allDocuments {
		isEditor := s.isUserInList(editorID, doc.Editors)
		isPrivilegedEditor := s.isUserInList(editorID, doc.PrivilegedEditors)

		if isEditor || isPrivilegedEditor {
			editableDocuments = append(editableDocuments, doc)
		}
	}

	if len(editableDocuments) == 0 {
		return nil, fmt.Errorf("document editable not found")
	}

	return editableDocuments, nil
}