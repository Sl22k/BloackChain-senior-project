#!/bin/bash

# SPDX-License-Identifier: Apache-2.0

# This script is based on the registerEnroll.sh script in the fabric-samples repository.
# It has been modified to work with the custom IS and CS organizations.

. scripts/utils.sh

function createIS() {
  infoln "Enrolling the CA admin"
  mkdir -p organizations/peerOrganizations/is.example.com/
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/is.example.com/

  set -x
  fabric-ca-client enroll -u https://admin:adminpw@localhost:7054 --caname ca-is --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  echo 'NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-is.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-is.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-is.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-is.pem
    OrganizationalUnitIdentifier: orderer' > "${PWD}/organizations/peerOrganizations/is.example.com/msp/config.yaml"

  infoln "Registering peer0"
  set -x
  fabric-ca-client register --caname ca-is --id.name peer0 --id.secret peer0pw --id.type peer --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering peer1"
  set -x
  fabric-ca-client register --caname ca-is --id.name peer1 --id.secret peer1pw --id.type peer --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering user"
  set -x
  fabric-ca-client register --caname ca-is --id.name user1 --id.secret user1pw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering the org admin"
  set -x
  fabric-ca-client register --caname ca-is --id.name isadmin --id.secret isadminpw --id.type admin --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Generating the peer0 msp"
  set -x
  fabric-ca-client enroll -u https://peer0:peer0pw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/msp" --csr.hosts peer0.is.example.com --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/is.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/msp/config.yaml"

  infoln "Generating the peer0-tls certificates"
  set -x
  fabric-ca-client enroll -u https://peer0:peer0pw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls" --enrollment.profile tls --csr.hosts peer0.is.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/server.key"

  mkdir -p "${PWD}/organizations/peerOrganizations/is.example.com/msp/tlscacerts"
  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/is.example.com/msp/tlscacerts/ca.crt"

  mkdir -p organizations/peerOrganizations/is.example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/is/ca-cert.pem" organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem

  mkdir -p organizations/peerOrganizations/is.example.com/ca
  cp "${PWD}/organizations/fabric-ca/is/ca-cert.pem" organizations/peerOrganizations/is.example.com/ca/ca.is.example.com-cert.pem

  infoln "Generating the peer1 msp"
  set -x
  fabric-ca-client enroll -u https://peer1:peer1pw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/msp" --csr.hosts peer1.is.example.com --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/is.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/msp/config.yaml"

  infoln "Generating the peer1-tls certificates"
  set -x
  fabric-ca-client enroll -u https://peer1:peer1pw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls" --enrollment.profile tls --csr.hosts peer1.is.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/is.example.com/peers/peer1.is.example.com/tls/server.key"

  mkdir -p organizations/peerOrganizations/is.example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/is/ca-cert.pem" organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem

  mkdir -p organizations/peerOrganizations/is.example.com/ca
  cp "${PWD}/organizations/fabric-ca/is/ca-cert.pem" organizations/peerOrganizations/is.example.com/ca/ca.is.example.com-cert.pem

  infoln "Generating the user msp"
  set -x
    fabric-ca-client enroll -u https://user1:user1pw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/users/User1@is.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

    cp "${PWD}/organizations/peerOrganizations/is.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/is.example.com/users/User1@is.example.com/msp/config.yaml"

  infoln "Generating the org admin msp"
  set -x
    fabric-ca-client enroll -u https://isadmin:isadminpw@localhost:7054 --caname ca-is -M "${PWD}/organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/is/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/is.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp/config.yaml"
}

function createCS() {
  infoln "Enrolling the CA admin"
  mkdir -p organizations/peerOrganizations/cs.example.com/
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/cs.example.com/

  set -x
  fabric-ca-client enroll -u https://admin:adminpw@localhost:8054 --caname ca-cs --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  echo 'NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-cs.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-cs.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-cs.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-cs.pem
    OrganizationalUnitIdentifier: orderer' > "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml"

  infoln "Registering peer0"
  set -x
  fabric-ca-client register --caname ca-cs --id.name peer0 --id.secret peer0pw --id.type peer --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering peer1"
  set -x
  fabric-ca-client register --caname ca-cs --id.name peer1 --id.secret peer1pw --id.type peer --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering peer2"
  set -x
  fabric-ca-client register --caname ca-cs --id.name peer2 --id.secret peer2pw --id.type peer --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering user"
  set -x
  fabric-ca-client register --caname ca-cs --id.name user1 --id.secret user1pw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering the org admin"
  set -x
  fabric-ca-client register --caname ca-cs --id.name csadmin --id.secret csadminpw --id.type admin --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Generating the peer0 msp"
  set -x
  fabric-ca-client enroll -u https://peer0:peer0pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/msp" --csr.hosts peer0.cs.example.com --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/msp/config.yaml"

  infoln "Generating the peer0-tls certificates"
  set -x
  fabric-ca-client enroll -u https://peer0:peer0pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls" --enrollment.profile tls --csr.hosts peer0.cs.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/server.key"

  mkdir -p "${PWD}/organizations/peerOrganizations/cs.example.com/msp/tlscacerts"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer0.cs.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/msp/tlscacerts/ca.crt"

  mkdir -p organizations/peerOrganizations/cs.example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem

  mkdir -p organizations/peerOrganizations/cs.example.com/ca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/ca/ca.cs.example.com-cert.pem

  infoln "Generating the peer1 msp"
  set -x
  fabric-ca-client enroll -u https://peer1:peer1pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/msp" --csr.hosts peer1.cs.example.com --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/msp/config.yaml"

  infoln "Generating the peer1-tls certificates"
  set -x
  fabric-ca-client enroll -u https://peer1:peer1pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls" --enrollment.profile tls --csr.hosts peer1.cs.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer1.cs.example.com/tls/server.key"

  mkdir -p organizations/peerOrganizations/cs.example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem

  mkdir -p organizations/peerOrganizations/cs.example.com/ca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/ca/ca.cs.example.com-cert.pem

  infoln "Generating the peer2 msp"
  set -x
  fabric-ca-client enroll -u https://peer2:peer2pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/msp" --csr.hosts peer2.cs.example.com --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/msp/config.yaml"

  infoln "Generating the peer2-tls certificates"
  set -x
  fabric-ca-client enroll -u https://peer2:peer2pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls" --enrollment.profile tls --csr.hosts peer2.cs.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/cs.example.com/peers/peer2.cs.example.com/tls/server.key"

  mkdir -p organizations/peerOrganizations/cs.example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem

  mkdir -p organizations/peerOrganizations/cs.example.com/ca
  cp "${PWD}/organizations/fabric-ca/cs/ca-cert.pem" organizations/peerOrganizations/cs.example.com/ca/ca.cs.example.com-cert.pem

  infoln "Generating the user msp"
  set -x
  fabric-ca-client enroll -u https://user1:user1pw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/users/User1@cs.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/cs.example.com/users/User1@cs.example.com/msp/config.yaml"

  infoln "Generating the org admin msp"
  set -x
  fabric-ca-client enroll -u https://csadmin:csadminpw@localhost:8054 --caname ca-cs -M "${PWD}/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/cs/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/peerOrganizations/cs.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp/config.yaml"
}

function createOrderer() {
  infoln "Enrolling the CA admin"
  mkdir -p organizations/ordererOrganizations/example.com
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/ordererOrganizations/example.com

  set -x
  fabric-ca-client enroll -u https://admin:adminpw@localhost:9054 --caname ca-orderer --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  echo 'NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-9054-ca-orderer.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-9054-ca-orderer.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-9054-ca-orderer.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-9054-ca-orderer.pem
    OrganizationalUnitIdentifier: orderer' > "${PWD}/organizations/ordererOrganizations/example.com/msp/config.yaml"

  infoln "Registering orderer"
  set -x
  fabric-ca-client register --caname ca-orderer --id.name orderer --id.secret ordererpw --id.type orderer --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Registering the orderer admin"
  set -x
  fabric-ca-client register --caname ca-orderer --id.name ordererAdmin --id.secret ordererAdminpw --id.type admin --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  infoln "Generating the orderer msp"
  set -x
  fabric-ca-client enroll -u https://orderer:ordererpw@localhost:9054 --caname ca-orderer -M "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp" --csr.hosts orderer.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/ordererOrganizations/example.com/msp/config.yaml" "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/config.yaml"

  infoln "Generating the orderer-tls certificates"
  set -x
  fabric-ca-client enroll -u https://orderer:ordererpw@localhost:9054 --caname ca-orderer -M "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls" --enrollment.profile tls --csr.hosts orderer.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlscacerts/"* "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"
  cp "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/signcerts/"* "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt"
  cp "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/keystore/"* "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.key"

  mkdir -p organizations/ordererOrganizations/example.com/tlsca
  cp "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem" organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem

  mkdir -p organizations/ordererOrganizations/example.com/ca
  cp "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem" organizations/ordererOrganizations/example.com/ca/ca.example.com-cert.pem

  mkdir -p "${PWD}/organizations/ordererOrganizations/example.com/msp/tlscacerts"
  cp "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/tlscacerts/"* "${PWD}/organizations/ordererOrganizations/example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

  infoln "Generating the admin msp"
  set -x
  fabric-ca-client enroll -u https://ordererAdmin:ordererAdminpw@localhost:9054 --caname ca-orderer -M "${PWD}/organizations/ordererOrganizations/example.com/users/Admin@example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/ordererOrg/ca-cert.pem"
  { set +x; } 2>/dev/null

  cp "${PWD}/organizations/ordererOrganizations/example.com/msp/config.yaml" "${PWD}/organizations/ordererOrganizations/example.com/users/Admin@example.com/msp/config.yaml"
}
