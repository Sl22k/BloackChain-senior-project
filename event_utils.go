package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// getCurrentDateTime returns current date and time in consistent formats
func (s *SmartContract) getCurrentDateTime(ctx contractapi.TransactionContextInterface) (string, string, error) {
	now := time.Now()
	date := now.Format("2006-01-02")
	timestamp := now.Format(time.RFC3339)
	return date, timestamp, nil
}

// emitEvent emits structured events for the blockchain network
func (s *SmartContract) emitEvent(ctx contractapi.TransactionContextInterface, eventType, actor, description string, documentID string, version int, details map[string]interface{}) error {
	eventData := map[string]interface{}{
		"eventType":   eventType,
		"actor":       actor,
		"description": description,
		"documentID":  documentID,
		"version":     version,
		"timestamp":   time.Now().Format(time.RFC3339),
		"details":     details,
	}

	eventJSON, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %v", err)
	}

	err = ctx.GetStub().SetEvent(eventType, eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}