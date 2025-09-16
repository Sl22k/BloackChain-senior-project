package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// loadDocument safely loads a document with essential error handling
func (s *SmartContract) loadDocument(ctx contractapi.TransactionContextInterface, id string) (*Document, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if data == nil {
		return nil, fmt.Errorf("document %s does not exist", id)
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %v", err)
	}

	return &doc, nil
}

// validateAuthorization performs essential security validation
func (s *SmartContract) validateAuthorization(userID string, doc *Document, accessType string) error {
	if userID == "" {
		return fmt.Errorf("unauthorized")
	}

	switch accessType {
	case "approve":
		// Basic authorization - detailed workflow validation happens in handlers
		return nil
	case "edit":
		if doc.Uploader != userID && !s.hasPrivilegedAccess(userID, doc) {
			return fmt.Errorf("unauthorized")
		}
	default:
		return fmt.Errorf("invalid access type")
	}

	return nil
}

// hasPrivilegedAccess checks if user has administrative access to document
func (s *SmartContract) hasPrivilegedAccess(invokerId string, doc *Document) bool {
	// Check if user is the uploader
	if doc.Uploader == invokerId {
		return true
	}

	// Check if user is in the privileged editors list
	if s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return true
	}

	// This can be extended to include admin roles, department managers, etc.
	return false
}

// isUserInList checks if a user exists in a given list
func (s *SmartContract) isUserInList(user string, list []string) bool {
	for _, listUser := range list {
		if listUser == user {
			return true
		}
	}
	return false
}

// hasEditorAccess checks if user has editor access (uploader, privileged, or regular editor)
func (s *SmartContract) hasEditorAccess(invokerId string, doc *Document) bool {
	// Check if user is the uploader (highest authority)
	if doc.Uploader == invokerId {
		return true
	}

	// Check if user is in privileged editors list
	if s.isUserInList(invokerId, doc.PrivilegedEditors) {
		return true
	}

	// Check if user is in regular editors list
	if s.isUserInList(invokerId, doc.Editors) {
		return true
	}

	return false
}

// saveDocument marshals and saves a document to the ledger
func (s *SmartContract) saveDocument(ctx contractapi.TransactionContextInterface, doc *Document) error {
	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}
	return ctx.GetStub().PutState(doc.ID, updated)
}

// validateInputs performs essential input validation (non-empty strings only)
func (s *SmartContract) validateInputs(inputs map[string]string) error {
	for _, value := range inputs {
		if value == "" {
			return fmt.Errorf("empty input")
		}
	}
	return nil
}