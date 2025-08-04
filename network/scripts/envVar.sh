#!/usr/bin/env bash
#
# Copyright IBM Corp All Rights Reserved
#
# SPDX-License-Identifier: Apache-2.0
#

# This is a collection of bash functions used by different scripts

# imports
# test network home var targets to test-network folder
# the reason we use a var here is to accommodate scenarios
# where execution occurs from folders outside of default as $PWD, such as the test-network/addOrg3 folder.
# For setting environment variables, simple relative paths like ".." could lead to unintended references
# due to how they interact with FABRIC_CFG_PATH. It's advised to specify paths more explicitly,
# such as using "../${PWD}", to ensure that Fabric's environment variables are pointing to the correct paths.
TEST_NETWORK_HOME=${TEST_NETWORK_HOME:-${PWD}}
. ${TEST_NETWORK_HOME}/scripts/utils.sh

export FABRIC_CFG_PATH=${TEST_NETWORK_HOME}/../config

export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${TEST_NETWORK_HOME}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_IS_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem
export PEER0_CS_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem
export PEER0_ORG3_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org3.example.com/tlsca/tlsca.org3.example.com-cert.pem

# Set environment variables for the peer org
setGlobals() {
  local orgName=$1
  local peerNum=$2
  infoln "Using organization ${orgName} and peer ${peerNum}"
  if [ "$orgName" = "is" ]; then
    export CORE_PEER_LOCALMSPID=ISMSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_IS_CA
    export CORE_PEER_MSPCONFIGPATH=${TEST_NETWORK_HOME}/organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp
    export CORE_PEER_ADDRESS=localhost:$((7051 + peerNum * 1000))
  elif [ "$orgName" = "cs" ]; then
    export CORE_PEER_LOCALMSPID=CSMSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_CS_CA
    export CORE_PEER_MSPCONFIGPATH=${TEST_NETWORK_HOME}/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp
    export CORE_PEER_ADDRESS=localhost:$((9051 + peerNum * 1000))
  else
    errorln "ORG Unknown"
  fi

  if [ "$VERBOSE" = "true" ]; then
    env | grep CORE
  fi
}

# parsePeerConnectionParameters $@
# Helper function that sets the peer connection parameters for a chaincode
# operation
parsePeerConnectionParameters() {
  PEER_CONN_PARMS=()
  PEERS=""
  while [ "$#" -gt 0 ]; do
    setGlobals $1 $2
    ORG_NAME=$1
    PEER=$CORE_PEER_ADDRESS
    ## Set peer addresses
    if [ -z "${PEERS}" ]
    then
	PEERS="$PEER"
    else
	PEERS="$PEERS $PEER"
    fi
    CA_VAR="PEER0_${ORG_NAME^^}_CA"
    PEER_CONN_PARMS=("${PEER_CONN_PARMS[@]}" --peerAddresses $CORE_PEER_ADDRESS --tlsRootCertFiles "${!CA_VAR}")
    # shift by two to get to the next organization and peer number
    shift 2
  done
}

verifyResult() {
  if [ $1 -ne 0 ]; then
    fatalln "$2"
  fi
}
