package chaincode

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type ApproverStatus string

const (
  Pending  ApproverStatus = "PENDING"
  Approved ApproverStatus = "APPROVED"
  Rejected ApproverStatus = "REJECTED"
)

type Document struct {
  ID           string                     `json:"ID"`
  Hash         string                     `json:"Hash"`
  Uploader     string                     `json:"Uploader"`
  Approvers    []string                   `json:"Approvers"`
  ApprovalsMap map[string]ApproverStatus `json:"ApprovalsMap"`
}

// StatusResponse gives document status summary
type StatusResponse struct {
  ID            string                     `json:"ID"`
  Approvers     []string                   `json:"Approvers"`
  ApprovalsMap  map[string]ApproverStatus `json:"ApprovalsMap"`
  ApprovedCount int                        `json:"ApprovedCount"`
  RejectedCount int                        `json:"RejectedCount"`
  PendingCount  int                        `json:"PendingCount"`
}

type SmartContract struct {
  contractapi.Contract
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
  docs := []Document{
    {
      ID: "doc1",
      Hash: "hash_abc123",
      Uploader: "alice@uni.edu",
      Approvers: []string{"profA@uni.edu", "profB@uni.edu"},
    },
    {
      ID: "doc2",
      Hash: "hash_def456",
      Uploader: "bob@uni.edu",
      Approvers: []string{"profA@uni.edu", "profC@uni.edu"},
    },
  }
  for _, doc := range docs {
    doc.ApprovalsMap = make(map[string]ApproverStatus)
    for _, a := range doc.Approvers {
      doc.ApprovalsMap[a] = Pending
    }
    data, err := json.Marshal(doc)
    if err != nil { return err }
    if err := ctx.GetStub().PutState(doc.ID, data); err != nil {
      return fmt.Errorf("failed to seed %s: %v", doc.ID, err)
    }
  }
  return nil
}

func (s *SmartContract) SubmitDocument(ctx contractapi.TransactionContextInterface, id, hash, uploader, approversJSON string) error {
  exists, err := s.DocumentExists(ctx, id)
  if err != nil { return err }
  if exists { return fmt.Errorf("document %s already exists", id) }

  var approvers []string
  if err := json.Unmarshal([]byte(approversJSON), &approvers); err != nil {
    return fmt.Errorf("invalid approvers JSON: %v", err)
  }
  approvals := make(map[string]ApproverStatus)
  for _, a := range approvers {
    approvals[a] = Pending
  }
  doc := Document{ID: id, Hash: hash, Uploader: uploader, Approvers: approvers, ApprovalsMap: approvals}
  data, err := json.Marshal(doc)
  if err != nil { return err }
  return ctx.GetStub().PutState(id, data)
}

func (s *SmartContract) ApproveDocument(ctx contractapi.TransactionContextInterface, id, approver, decision string) error {
  data, err := ctx.GetStub().GetState(id)
  if err != nil || data == nil {
    return fmt.Errorf("document %s not found", id)
  }
  var doc Document
  if err := json.Unmarshal(data, &doc); err != nil {
    return err
  }
  status, ok := doc.ApprovalsMap[approver]
  if !ok {
    return fmt.Errorf("%s not authorized to decide", approver)
  }
  if status != Pending {
    return fmt.Errorf("%s already decided", approver)
  }
  switch decision {
  case "APPROVE":
    doc.ApprovalsMap[approver] = Approved
  case "REJECT":
    doc.ApprovalsMap[approver] = Rejected
  default:
    return fmt.Errorf("invalid decision '%s'", decision)
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
  var resp StatusResponse
  resp.ID = doc.ID
  resp.Approvers = doc.Approvers
  resp.ApprovalsMap = doc.ApprovalsMap
  for _, s := range doc.ApprovalsMap {
    switch s {
    case Approved:
      resp.ApprovedCount++
    case Rejected:
      resp.RejectedCount++
    case Pending:
      resp.PendingCount++
    }
  }
  return &resp, nil
}

func (s *SmartContract) DocumentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
  data, err := ctx.GetStub().GetState(id)
  if err != nil { return false, err }
  return data != nil, nil
}

// GetAllDocuments returns all documents on the ledger
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

		var document Document
		err = json.Unmarshal(queryResponse.Value, &document)
		if err != nil {
			return nil, err
		}
		documents = append(documents, &document)
	}

	return documents, nil
}