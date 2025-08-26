#!/bin/bash

# This script is based on Hyperledger Fabric's test-network script.
# It provides a simplified way to manage a custom Fabric network for the Document Approval System.

# Current Fabric version
FABRIC_VERSION=2.5.13
CA_VERSION=1.5.7

# Channel name
CHANNEL_NAME="approvalchannel"

# Chaincode details
CHAINCODE_NAME="document_approval"
CHAINCODE_PATH="../chaincode-go"
CHAINCODE_LANGUAGE="go"
CHAINCODE_VERSION="1.0"

# Function to print the usage of the script
function printHelp() {
  echo "Usage: "
  echo "  network.sh <Mode> [Flags]"
  echo "    Mode:"
  echo "      up            - Bring up the Fabric network (create CAs, generate crypto, create genesis block, start orderer and peers)"
  echo "      down          - Bring down the Fabric network (stop and remove containers, delete crypto and artifacts)"
  echo "      createChannel - Create a channel for the network"
  echo "      deployCC      - Deploy chaincode to the channel"
  echo "    Flags:"
  echo "      -c <channel name> - Channel name (default: approvalchannel)"
  echo "      -ccn <chaincode name> - Chaincode name (default: document_approval)"
  echo "      -ccp <chaincode path> - Chaincode path (default: ../chaincode)"
  echo "      -ccl <chaincode language> - Chaincode language (default: go)"
  echo "      -ccv <chaincode version> - Chaincode version (default: 1.0)"
  echo "      -p                - Start postgres containers"
  echo ""
  echo "Examples:"
  echo "  network.sh up"
  echo "  network.sh up -p"
  echo "  network.sh createChannel"
  echo "  network.sh deployCC"
  echo "  network.sh down"
}

# Function to generate crypto material using cryptogen
function generateCerts() {
  echo "Generating certificates using cryptogen..."
  if [ -d "organizations/peerOrganizations" ]; then
    rm -Rf organizations/peerOrganizations && rm -Rf organizations/ordererOrganizations
  fi

  set -x
  cryptogen generate --config=./organizations/crypto-config.yaml --output="organizations"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  echo "Certificates generated successfully."
}

# Function to create genesis block and channel transaction
function createConsortiumAndChannelArtifacts() {
  echo "Generating genesis block and channel transaction..."
  export FABRIC_CFG_PATH=$(pwd)
  set -x
  configtxgen -profile TwoOrgsOrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate orderer genesis block..."
    exit 1
  fi

  set -x
  configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/${CHANNEL_NAME}.tx -channelID $CHANNEL_NAME
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate channel configuration transaction..."
    exit 1
  fi
  echo "Genesis block and channel transaction generated successfully."
}

# Function to bring up the network
function networkUp() {
  generateCerts
  mkdir -p system-genesis-block channel-artifacts
  createConsortiumAndChannelArtifacts

  COMPOSE_FILES="-f docker-compose.yaml"
  if [ "$POSTGRES" == "true" ]; then
    COMPOSE_FILES="$COMPOSE_FILES -f ../../network/compose/compose-postgres.yaml"
  fi

  echo "Starting Fabric network..."
  docker-compose $COMPOSE_FILES up -d 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR! Unable to start network..."
    exit 1
  fi
  echo "Fabric network started successfully."
}

# Function to create a channel
function createChannel() {
  echo "Creating channel '$CHANNEL_NAME'..."
  # Ensure the network is up before creating channel
  docker-compose -f docker-compose.yaml ps | grep "Up"
  if [ $? -ne 0 ]; then
    echo "Network is not up. Please run 'network.sh up' first."
    exit 1
  fi

  # Set environment variables for peer0.creator.example.com
  export CORE_PEER_TLS_ENABLED=true
  export CORE_PEER_LOCALMSPID="CreatorMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/creator.example.com/peers/peer0.creator.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/creator.example.com/users/Admin@creator.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:7051

  set -x
  peer channel create -o localhost:7050 -c $CHANNEL_NAME --ordererTLSHostnameOverride orderer.example.com -f ./channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ./channel-artifacts/${CHANNEL_NAME}.block --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to create channel..."
    exit 1
  fi
  echo "Channel '$CHANNEL_NAME' created successfully."

  echo "Joining peer0.creator.example.com to channel '$CHANNEL_NAME'..."
  set -x
  peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to join peer0.creator.example.com to channel..."
    exit 1
  fi
  echo "peer0.creator.example.com joined channel successfully."

  echo "Joining peer0.approver.example.com to channel '$CHANNEL_NAME'..."
  # Set environment variables for peer0.approver.example.com
  export CORE_PEER_LOCALMSPID="ApproverMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/approver.example.com/peers/peer0.approver.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/approver.example.com/users/Admin@approver.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:9051

  set -x
  peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to join peer0.approver.example.com to channel..."
    exit 1
  fi
  echo "peer0.approver.example.com joined channel successfully."

  echo "Updating anchor peers for CreatorOrg..."
  export CORE_PEER_LOCALMSPID="CreatorMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/creator.example.com/peers/peer0.creator.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/creator.example.com/users/Admin@creator.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:7051
  set -x
  configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/CreatorMSPanchors.tx -channelID $CHANNEL_NAME -asOrg CreatorMSP
  peer channel update -o localhost:7050 -c $CHANNEL_NAME --ordererTLSHostnameOverride orderer.example.com -f ./channel-artifacts/CreatorMSPanchors.tx --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to update anchor peers for CreatorOrg..."
    exit 1
  fi
  echo "Anchor peers for CreatorOrg updated successfully."

  echo "Updating anchor peers for ApproverOrg..."
  export CORE_PEER_LOCALMSPID="ApproverMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/approver.example.com/peers/peer0.approver.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/approver.example.com/users/Admin@approver.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:9051
  set -x
  configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/ApproverMSPanchors.tx -channelID $CHANNEL_NAME -asOrg ApproverMSP
  peer channel update -o localhost:7050 -c $CHANNEL_NAME --ordererTLSHostnameOverride orderer.example.com -f ./channel-artifacts/ApproverMSPanchors.tx --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to update anchor peers for ApproverOrg..."
    exit 1
  fi
  echo "Anchor peers for ApproverOrg updated successfully."
}

# Function to deploy chaincode
function deployCC() {
  echo "Deploying chaincode '$CHAINCODE_NAME'..."
  export FABRIC_CFG_PATH=".."

  # Set environment variables for peer0.creator.example.com
  export CORE_PEER_TLS_ENABLED=true
  export CORE_PEER_LOCALMSPID="CreatorMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/creator.example.com/peers/peer0.creator.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/creator.example.com/users/Admin@creator.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:7051

  # Package chaincode
  set -x
  peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz --path ${CHAINCODE_PATH} --lang ${CHAINCODE_LANGUAGE} --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to package chaincode..."
    exit 1
  fi

  # Install chaincode on peer0.creator.example.com
  set -x
  peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to install chaincode on peer0.creator.example.com..."
    exit 1
  fi

  # Query installed chaincode to get package ID
  set -x
  PACKAGE_ID=$(peer lifecycle chaincode queryinstalled --output json | jq -r '.installed_chaincodes[] | select(.label=="'${CHAINCODE_NAME}_${CHAINCODE_VERSION}'") | .package_id')
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to query installed chaincode..."
    exit 1
  fi
  echo "Chaincode package ID: $PACKAGE_ID"

  # Approve chaincode for CreatorOrg
  set -x
  peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --package-id $PACKAGE_ID --sequence 1 --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to approve chaincode for CreatorOrg..."
    exit 1
  fi

  # Set environment variables for peer0.approver.example.com
  export CORE_PEER_LOCALMSPID="ApproverMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$(pwd)/organizations/peerOrganizations/approver.example.com/peers/peer0.approver.example.com/tls/ca.crt"
  export CORE_PEER_MSPCONFIGPATH="$(pwd)/organizations/peerOrganizations/approver.example.com/users/Admin@approver.example.com/msp"
  export CORE_PEER_ADDRESS=localhost:9051

  # Install chaincode on peer0.approver.example.com
  set -x
  peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to install chaincode on peer0.approver.example.com..."
    exit 1
  fi

  # Approve chaincode for ApproverOrg
  set -x
  peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --package-id $PACKAGE_ID --sequence 1 --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to approve chaincode for ApproverOrg..."
    exit 1
  fi

  # Check commit readiness
  set -x
  peer lifecycle chaincode checkcommitreadiness --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence 1 --output json
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Chaincode not ready for commit..."
    exit 1
  fi

  # Commit chaincode
  set -x
  peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence 1 --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "$(pwd)/organizations/peerOrganizations/creator.example.com/peers/peer0.creator.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "$(pwd)/organizations/peerOrganizations/approver.example.com/peers/peer0.approver.example.com/tls/ca.crt"
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to commit chaincode..."
    exit 1
  fi
  echo "Chaincode '$CHAINCODE_NAME' deployed successfully."

  # Initialize chaincode (optional, if your chaincode has an InitLedger function)
  echo "Initializing chaincode ledger..."
  set -x
  peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "$(pwd)/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlsca.example.com-cert.pem" -C $CHANNEL_NAME -n ${CHAINCODE_NAME} --peerAddresses localhost:7051 --tlsRootCertFiles "$(pwd)/organizations/peerOrganizations/creator.example.com/peers/peer0.creator.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "$(pwd)/organizations/peerOrganizations/approver.example.com/peers/peer0.approver.example.com/tls/ca.crt" -c '{"function":"InitLedger","Args":[]}'
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to initialize chaincode ledger..."
    exit 1
  fi
  echo "Chaincode ledger initialized successfully."
}

# Function to bring down the network
function networkDown() {
  echo "Bringing down Fabric network..."
  COMPOSE_FILES="-f docker-compose.yaml"
  if [ "$POSTGRES" == "true" ]; then
    COMPOSE_FILES="$COMPOSE_FILES -f ../../network/compose/compose-postgres.yaml"
  fi
  docker-compose $COMPOSE_FILES down --volumes --remove-orphans
  if [ $? -ne 0 ]; then
    echo "ERROR! Unable to bring down network..."
    exit 1
  fi
  # Clean up generated artifacts
  rm -Rf organizations/peerOrganizations organizations/ordererOrganizations system-genesis-block channel-artifacts
  rm -f ${CHAINCODE_NAME}.tar.gz
  echo "Fabric network brought down and artifacts removed."
}

# Parse command line arguments
MODE=$1
shift
while getopts "c:ccn:ccp:ccl:ccv:p" opt; do
  case "$opt" in
    c)
      CHANNEL_NAME=$OPTARG
      ;;
    ccn)
      CHAINCODE_NAME=$OPTARG
      ;;
    ccp)
      CHAINCODE_PATH=$OPTARG
      ;;
    ccl)
      CHAINCODE_LANGUAGE=$OPTARG
      ;;
    ccv)
      CHAINCODE_VERSION=$OPTARG
      ;;
    p)
      POSTGRES="true"
      ;;
    \?)
      printHelp
      exit 1
      ;;
  esac
done

# Execute the chosen mode
if [ "$MODE" == "up" ]; then
  networkUp
elif [ "$MODE" == "down" ]; then
  networkDown
elif [ "$MODE" == "createChannel" ]; then
  createChannel
elif [ "$MODE" == "deployCC" ]; then
  deployCC
else
  printHelp
  exit 1
fi
