#!/usr/bin/env bash

function one_line_pem {
    echo "$(awk 'NF {sub(/\n/, ""); printf "%s\\n",$0;}' $1)"
}

function json_ccp {
    local PP=$(one_line_pem $4)
    local CP=$(one_line_pem $5)
    sed -e "s/${ORG}/$1/g" \
        -e "s/${P0PORT}/$2/g" \
        -e "s/${CAPORT}/$3/g" \
        -e "s#${PEERPEM}#$PP#g" \
        -e "s#${CAPEM}#$CP#g" \
        organizations/ccp-template.json
}

function yaml_ccp {
    local PP=$(one_line_pem $4)
    local CP=$(one_line_pem $5)
    sed -e "s/${ORG}/$1/g" \
        -e "s/${P0PORT}/$2/g" \
        -e "s/${CAPORT}/$3/g" \
        -e "s#${PEERPEM}#$PP#g" \
        -e "s#${CAPEM}#$CP#g" \
        organizations/ccp-template.yaml | sed -e 's/\n/\n          /g'
}

ORG=is
P0PORT=7051
CAPORT=7054
PEERPEM=organizations/peerOrganizations/is.example.com/peers/peer0.is.example.com/tls/ca.crt
CAPEM=organizations/peerOrganizations/is.example.com/ca/ca.is.example.com-cert.pem

echo "$(json_ccp $ORG $P0PORT $CAPORT $PEERPEM $CAPEM)" > organizations/peerOrganizations/is.example.com/connection-is.json
echo "$(yaml_ccp $ORG $P0PORT $CAPORT $PEERPEM $CAPEM)" > organizations/peerOrganizations/is.example.com/connection-is.yaml

ORG=cs
P0PORT=9051
CAPORT=8054
PEERPEM=organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem
CAPEM=organizations/peerOrganizations/cs.example.com/ca/ca.cs.example.com-cert.pem

echo "$(json_ccp $ORG $P0PORT $CAPORT $PEERPEM $CAPEM)" > organizations/peerOrganizations/cs.example.com/connection-cs.json
echo "$(yaml_ccp $ORG $P0PORT $CAPORT $PEERPEM $CAPEM)" > organizations/peerOrganizations/cs.example.com/connection-cs.yaml
