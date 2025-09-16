package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// AddEditor adds a new editor to a document
func (s *SmartContract) AddEditor(ctx contractapi.TransactionContextInterface, id, newEditor, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if err := s.validateEditorManagement(invokerId, doc); err != nil {
		return err
	}

	if s.hasEditorAccess(newEditor, doc) {
		return fmt.Errorf("user %s is already an editor for document %s", newEditor, id)
	}

	doc.Editors = addToUserList(newEditor, doc.Editors)
	doc.LastModifiedTimestamp = timestamp

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	description := fmt.Sprintf("Editor %s added to document %s by %s", newEditor, id, invokerId)
	details := map[string]interface{}{
		"newEditor": newEditor,
	}

	return s.emitEvent(ctx, "EditorAdded", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// RemoveEditor removes an editor from a document
func (s *SmartContract) RemoveEditor(ctx contractapi.TransactionContextInterface, id, editorToRemove, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if err := s.validateEditorManagement(invokerId, doc); err != nil {
		return err
	}

	if editorToRemove == doc.Uploader {
		return fmt.Errorf("cannot remove document uploader %s from editors", editorToRemove)
	}

	if !s.hasEditorAccess(editorToRemove, doc) {
		return fmt.Errorf("user %s is not an editor for document %s", editorToRemove, id)
	}

	// Remove from regular editors list (privileged editors should be removed via RemovePrivilegedEditor)
	if s.isUserInList(editorToRemove, doc.PrivilegedEditors) {
		return fmt.Errorf("user %s is a privileged editor - use RemovePrivilegedEditor to remove", editorToRemove)
	}

	doc.Editors = removeFromUserList(editorToRemove, doc.Editors)
	doc.LastModifiedTimestamp = timestamp

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	description := fmt.Sprintf("Editor %s removed from document %s by %s", editorToRemove, id, invokerId)
	details := map[string]interface{}{
		"removedEditor": editorToRemove,
	}

	return s.emitEvent(ctx, "EditorRemoved", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// AddPrivilegedEditor adds a new privileged editor to a document
func (s *SmartContract) AddPrivilegedEditor(ctx contractapi.TransactionContextInterface, id, newPrivilegedEditor, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if err := s.validatePrivilegedEditorManagement(invokerId, doc, false); err != nil {
		return err
	}

	if s.isUserInList(newPrivilegedEditor, doc.PrivilegedEditors) {
		return fmt.Errorf("user %s is already a privileged editor for document %s", newPrivilegedEditor, id)
	}

	doc.PrivilegedEditors = addToUserList(newPrivilegedEditor, doc.PrivilegedEditors)
	doc.LastModifiedTimestamp = timestamp

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	description := fmt.Sprintf("Privileged editor %s added to document %s by %s", newPrivilegedEditor, id, invokerId)
	details := map[string]interface{}{
		"newPrivilegedEditor": newPrivilegedEditor,
	}

	return s.emitEvent(ctx, "PrivilegedEditorAdded", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// RemovePrivilegedEditor removes a privileged editor from a document
func (s *SmartContract) RemovePrivilegedEditor(ctx contractapi.TransactionContextInterface, id, privilegedEditorToRemove, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	doc, err := s.loadDocument(ctx, id)
	if err != nil {
		return err
	}

	if err := s.validatePrivilegedEditorManagement(invokerId, doc, true); err != nil {
		return err
	}

	if privilegedEditorToRemove == doc.Uploader {
		return fmt.Errorf("cannot remove document owner %s from privileged editors", privilegedEditorToRemove)
	}

	if !s.isUserInList(privilegedEditorToRemove, doc.PrivilegedEditors) {
		return fmt.Errorf("user %s is not a privileged editor for document %s", privilegedEditorToRemove, id)
	}

	doc.PrivilegedEditors = removeFromUserList(privilegedEditorToRemove, doc.PrivilegedEditors)
	doc.LastModifiedTimestamp = timestamp

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	description := fmt.Sprintf("Privileged editor %s removed from document %s by %s", privilegedEditorToRemove, id, invokerId)
	details := map[string]interface{}{
		"removedPrivilegedEditor": privilegedEditorToRemove,
	}

	return s.emitEvent(ctx, "PrivilegedEditorRemoved", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// UpdateDocumentApprovers updates the list of approvers for a document
func (s *SmartContract) UpdateDocumentApprovers(ctx contractapi.TransactionContextInterface, documentID string, newApproversJSON string, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	if err := s.validateEditorManagement(invokerId, doc); err != nil {
		return err
	}

	var newApprovers []string
	if err := json.Unmarshal([]byte(newApproversJSON), &newApprovers); err != nil {
		return fmt.Errorf("invalid newApprovers JSON: %v", err)
	}

	oldApprovers := createDeterministicApproversList(doc.ApprovalsMap)
	doc.ApprovalsMap = rebuildApprovalsMap(doc.ApprovalsMap, newApprovers, timestamp)
	synchronizeWorkflowApprovers(doc, newApprovers)

	doc.LastModifiedTimestamp = timestamp
	doc.LatestVersion++

	currentHash := "APPROVERS_UPDATED"
	if len(doc.Versions) > 0 {
		currentHash = doc.Versions[len(doc.Versions)-1].Hash
	}

	currentApprovals := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		currentApprovals[k] = v
	}

	// Create workflow snapshot for version history
	workflowSnapshot := &WorkflowSnapshot{
		Enabled:            doc.Workflow.Enabled,
		CurrentStage:       doc.Workflow.CurrentStage,
		CompletedStages:    append([]int{}, doc.Workflow.CompletedStages...),
		TotalStages:        len(doc.Workflow.Stages),
		CompletionTimestamp: timestamp,
	}

	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:            currentHash,
		Submitter:       invokerId,
		Timestamp:       timestamp,
		ApprovalsMap:    currentApprovals,
		ValidDecisions:  append([]string{}, doc.ValidDecisions...),
		ApproversChanged: true,
		DecisionsChanged: false,
		WorkflowSnapshot: workflowSnapshot,
	}
	doc.Versions = append(doc.Versions, newVersion)

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	sort.Strings(newApprovers)
	description := fmt.Sprintf("Approvers updated for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"oldApprovers": oldApprovers,
		"newApprovers": newApprovers,
	}

	return s.emitEvent(ctx, "ApproversUpdated", invokerId, description, doc.ID, doc.LatestVersion, details)
}

// RequestOwnershipTransfer initiates an ownership transfer request
func (s *SmartContract) RequestOwnershipTransfer(ctx contractapi.TransactionContextInterface, documentID, newOwner, message, invokerId string, expirationDays int) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Validate that only current owner can request transfer
	if doc.Uploader != invokerId {
		return fmt.Errorf("only document owner %s can request ownership transfer", doc.Uploader)
	}

	// Check if document is completed (restriction: cannot transfer completed documents)
	if s.isDocumentFullyApproved(*doc) {
		return fmt.Errorf("cannot transfer ownership of completed document %s", documentID)
	}

	// Check if there's already a pending transfer request
	if doc.PendingOwnershipTransfer.RequestedBy != "" {
		return fmt.Errorf("document %s already has a pending ownership transfer request", documentID)
	}

	// Validate expiration days
	if expirationDays < 1 || expirationDays > 365 {
		return fmt.Errorf("expiration days must be between 1 and 365")
	}

	// Validate new owner is not the same as current owner
	if newOwner == doc.Uploader {
		return fmt.Errorf("cannot transfer ownership to yourself")
	}

	// Calculate expiration time
	expirationTime := time.Now().AddDate(0, 0, expirationDays).Format(time.RFC3339)

	// Create transfer request
	doc.PendingOwnershipTransfer = OwnershipTransferRequest{
		RequestedBy:      invokerId,
		NewOwner:         newOwner,
		RequestTimestamp: timestamp,
		ExpirationTime:   expirationTime,
		ExpirationDays:   expirationDays,
		Message:          message,
	}

	doc.LastModifiedTimestamp = timestamp

	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Emit events for both current owner and new owner
	description := fmt.Sprintf("Ownership transfer requested for document %s to %s", documentID, newOwner)
	details := map[string]interface{}{
		"currentOwner":    doc.Uploader,
		"newOwner":        newOwner,
		"message":         message,
		"expirationDays":  expirationDays,
		"expirationTime":  expirationTime,
	}

	// Event for current owner (confirmation)
	s.emitEvent(ctx, "OwnershipTransferRequested", invokerId, description, doc.ID, doc.LatestVersion, details)

	// Event for new owner (notification)
	receivedDescription := fmt.Sprintf("You have received an ownership transfer request for document %s from %s", documentID, invokerId)
	return s.emitEvent(ctx, "OwnershipTransferReceived", newOwner, receivedDescription, doc.ID, doc.LatestVersion, details)
}

// AcceptOwnershipTransfer accepts a pending ownership transfer request
func (s *SmartContract) AcceptOwnershipTransfer(ctx contractapi.TransactionContextInterface, documentID, invokerId string, resetEditors bool) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Check if there's a pending transfer request
	if doc.PendingOwnershipTransfer.RequestedBy == "" {
		return fmt.Errorf("no pending ownership transfer request for document %s", documentID)
	}

	// Validate that only the target new owner can accept
	if doc.PendingOwnershipTransfer.NewOwner != invokerId {
		return fmt.Errorf("only the target new owner %s can accept the ownership transfer", doc.PendingOwnershipTransfer.NewOwner)
	}

	// Check if transfer has expired
	expirationTime, err := time.Parse(time.RFC3339, doc.PendingOwnershipTransfer.ExpirationTime)
	if err != nil {
		return fmt.Errorf("invalid expiration time format: %v", err)
	}
	if time.Now().After(expirationTime) {
		return fmt.Errorf("ownership transfer request has expired")
	}

	// Store previous owner for version history
	previousOwner := doc.Uploader

	// Transfer ownership
	doc.Uploader = invokerId

	// Handle editor reset if requested
	if resetEditors {
		doc.Editors = []string{}
		doc.PrivilegedEditors = []string{}
	} else {
		// Remove new owner from editor lists if they were in them
		doc.Editors = removeFromUserList(invokerId, doc.Editors)
		doc.PrivilegedEditors = removeFromUserList(invokerId, doc.PrivilegedEditors)
	}

	// Create new version for ownership change
	doc.LatestVersion++
	doc.LastModifiedTimestamp = timestamp

	// Get current hash from latest version or set default
	currentHash := "OWNERSHIP_TRANSFERRED"
	if len(doc.Versions) > 0 {
		currentHash = doc.Versions[len(doc.Versions)-1].Hash
	}

	// Create deep copy of current approvals
	currentApprovals := make(map[string]Decision)
	for k, v := range doc.ApprovalsMap {
		currentApprovals[k] = v
	}

	// Create workflow snapshot for version history
	workflowSnapshot := &WorkflowSnapshot{
		Enabled:            doc.Workflow.Enabled,
		CurrentStage:       doc.Workflow.CurrentStage,
		CompletedStages:    append([]int{}, doc.Workflow.CompletedStages...),
		TotalStages:        len(doc.Workflow.Stages),
		CompletionTimestamp: timestamp,
	}

	// Create new version entry
	newVersion := DocumentVersion{
		Version:          doc.LatestVersion,
		Hash:            currentHash,
		Submitter:       invokerId,
		Timestamp:       timestamp,
		ApprovalsMap:    currentApprovals,
		ValidDecisions:  append([]string{}, doc.ValidDecisions...),
		ApproversChanged: false,
		DecisionsChanged: false,
		ChangeType:      "ownership",
		PreviousOwner:   previousOwner,
		WorkflowSnapshot: workflowSnapshot,
	}
	doc.Versions = append(doc.Versions, newVersion)

	// Clear pending transfer request
	doc.PendingOwnershipTransfer = OwnershipTransferRequest{}

	// Save document
	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Prepare event details
	description := fmt.Sprintf("Ownership transfer accepted for document %s by %s", documentID, invokerId)
	details := map[string]interface{}{
		"previousOwner": previousOwner,
		"newOwner":     invokerId,
		"resetEditors": resetEditors,
	}

	// Event for new owner (confirmation)
	s.emitEvent(ctx, "OwnershipTransferAccepted", invokerId, description, doc.ID, doc.LatestVersion, details)

	// Event for previous owner (notification)
	previousOwnerDescription := fmt.Sprintf("Your ownership transfer request for document %s has been accepted by %s", documentID, invokerId)
	return s.emitEvent(ctx, "OwnershipTransferAcceptedNotification", previousOwner, previousOwnerDescription, doc.ID, doc.LatestVersion, details)
}

// RejectOwnershipTransfer rejects a pending ownership transfer request
func (s *SmartContract) RejectOwnershipTransfer(ctx contractapi.TransactionContextInterface, documentID, rejectionMessage, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Check if there's a pending transfer request
	if doc.PendingOwnershipTransfer.RequestedBy == "" {
		return fmt.Errorf("no pending ownership transfer request for document %s", documentID)
	}

	// Validate that only the target new owner can reject
	if doc.PendingOwnershipTransfer.NewOwner != invokerId {
		return fmt.Errorf("only the target new owner %s can reject the ownership transfer", doc.PendingOwnershipTransfer.NewOwner)
	}

	// Store transfer details for events before clearing
	requestedBy := doc.PendingOwnershipTransfer.RequestedBy
	requestMessage := doc.PendingOwnershipTransfer.Message

	// Clear pending transfer request
	doc.PendingOwnershipTransfer = OwnershipTransferRequest{}
	doc.LastModifiedTimestamp = timestamp

	// Save document
	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Prepare event details
	details := map[string]interface{}{
		"requestedBy":       requestedBy,
		"rejectedBy":        invokerId,
		"originalMessage":   requestMessage,
		"rejectionMessage":  rejectionMessage,
	}

	// Event for target owner (confirmation)
	description := fmt.Sprintf("Ownership transfer request for document %s rejected", documentID)
	s.emitEvent(ctx, "OwnershipTransferRejected", invokerId, description, doc.ID, doc.LatestVersion, details)

	// Event for original requestor (notification)
	requestorDescription := fmt.Sprintf("Your ownership transfer request for document %s has been rejected by %s", documentID, invokerId)
	return s.emitEvent(ctx, "OwnershipTransferRejectedNotification", requestedBy, requestorDescription, doc.ID, doc.LatestVersion, details)
}

// CancelOwnershipTransfer cancels a pending ownership transfer request (owner only)
func (s *SmartContract) CancelOwnershipTransfer(ctx contractapi.TransactionContextInterface, documentID, invokerId string) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Load document
	doc, err := s.loadDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Check if there's a pending transfer request
	if doc.PendingOwnershipTransfer.RequestedBy == "" {
		return fmt.Errorf("no pending ownership transfer request for document %s", documentID)
	}

	// Validate that only the original requestor (current owner) can cancel
	if doc.PendingOwnershipTransfer.RequestedBy != invokerId {
		return fmt.Errorf("only the original requestor %s can cancel the ownership transfer", doc.PendingOwnershipTransfer.RequestedBy)
	}

	// Additional validation: verify invokerId is still the current document owner
	if doc.Uploader != invokerId {
		return fmt.Errorf("only the current document owner %s can cancel ownership transfer", doc.Uploader)
	}

	// Store transfer details for events before clearing
	targetOwner := doc.PendingOwnershipTransfer.NewOwner
	originalMessage := doc.PendingOwnershipTransfer.Message

	// Clear pending transfer request
	doc.PendingOwnershipTransfer = OwnershipTransferRequest{}
	doc.LastModifiedTimestamp = timestamp

	// Save document
	if err := s.saveDocument(ctx, doc); err != nil {
		return err
	}

	// Prepare event details
	details := map[string]interface{}{
		"cancelledBy":       invokerId,
		"targetOwner":       targetOwner,
		"originalMessage":   originalMessage,
	}

	// Event for current owner (confirmation)
	description := fmt.Sprintf("Ownership transfer request for document %s cancelled", documentID)
	s.emitEvent(ctx, "OwnershipTransferCancelled", invokerId, description, doc.ID, doc.LatestVersion, details)

	// Event for target owner (notification)
	targetOwnerDescription := fmt.Sprintf("The ownership transfer request for document %s from %s has been cancelled", documentID, invokerId)
	return s.emitEvent(ctx, "OwnershipTransferCancelledNotification", targetOwner, targetOwnerDescription, doc.ID, doc.LatestVersion, details)
}