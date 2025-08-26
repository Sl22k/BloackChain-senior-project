// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"document-approval/chaincode-go/chaincode"
)

func main() {
	docChaincode, err := contractapi.NewChaincode(&chaincode.SmartContract{})
	if err != nil {
		log.Panicf("Error creating document approval chaincode: %v", err)
	}

	if err := docChaincode.Start(); err != nil {
		log.Panicf("Error starting document approval chaincode: %v", err)
	}
}
