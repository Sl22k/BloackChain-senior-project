#!/bin/bash

# =============================================================================
# HYPERLEDGER FABRIC DOCUMENT APPROVAL CHAINCODE TEST LAUNCHER
# =============================================================================
# Modular Test Script System - Phase 6 Production Ready
# Matches the 8-module chaincode architecture from refactoring session
# =============================================================================

# Load shared functions
LAUNCHER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${LAUNCHER_DIR}/shared/common-functions.sh"

# Define color codes for better visibility
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'  # No Color

# =============================================================================
# PEER ENVIRONMENT MANAGEMENT
# =============================================================================

# Function to display peer selection menu
display_peer_menu() {
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}   Fabric Peer Environment Setup       ${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
    echo -e "${GREEN}Select a peer to set environment variables:${NC}"
    echo ""
    echo -e "${YELLOW}IS Organization:${NC}"
    echo -e "${BLUE}1${NC} - IS Peer 0 (localhost:7051)"
    echo -e "${BLUE}2${NC} - IS Peer 1 (localhost:8051)"
    echo ""
    echo -e "${YELLOW}CS Organization:${NC}"
    echo -e "${BLUE}3${NC} - CS Peer 0 (localhost:9051)"
    echo -e "${BLUE}4${NC} - CS Peer 1 (localhost:10051)"
    echo -e "${BLUE}5${NC} - CS Peer 2 (localhost:11051)"
    echo ""
    echo -e "${RED}6${NC} - Exit"
    echo ""
}

# Function to set environment variables for selected peer
set_peer_env() {
    local peer_choice="$1"

    echo -e "${GREEN}Setting environment variables...${NC}"
    echo ""

    # Show common environment variables being set
    echo -e "${PURPLE}export PATH=\${PWD}/../../bin:\$PATH${NC}"
    echo -e "${PURPLE}export FABRIC_CFG_PATH=\$PWD/../../config/${NC}"

    # Set common environment variables - adjust paths for test-scripts subdirectory
    export PATH=${PWD}/../../bin:$PATH
    export FABRIC_CFG_PATH=${PWD}/../../config/

    case $peer_choice in
        1)
            # IS Organization - Peer 0
            echo -e "${PURPLE}export CORE_PEER_LOCALMSPID=ISMSP${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ENABLED=true${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ROOTCERT_FILE=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem${NC}"
            echo -e "${PURPLE}export CORE_PEER_MSPCONFIGPATH=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp${NC}"
            echo -e "${PURPLE}export CORE_PEER_ADDRESS=localhost:7051${NC}"

            export CORE_PEER_LOCALMSPID=ISMSP
            export CORE_PEER_TLS_ENABLED=true
            export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem
            export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp
            export CORE_PEER_ADDRESS=localhost:7051
            echo ""
            echo -e "${GREEN}✅ Environment set for IS Peer 0 (localhost:7051)${NC}"
            ;;
        2)
            # IS Organization - Peer 1
            echo -e "${PURPLE}export CORE_PEER_LOCALMSPID=ISMSP${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ENABLED=true${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ROOTCERT_FILE=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem${NC}"
            echo -e "${PURPLE}export CORE_PEER_MSPCONFIGPATH=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp${NC}"
            echo -e "${PURPLE}export CORE_PEER_ADDRESS=localhost:8051${NC}"

            export CORE_PEER_LOCALMSPID=ISMSP
            export CORE_PEER_TLS_ENABLED=true
            export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem
            export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp
            export CORE_PEER_ADDRESS=localhost:8051
            echo ""
            echo -e "${GREEN}✅ Environment set for IS Peer 1 (localhost:8051)${NC}"
            ;;
        3)
            # CS Organization - Peer 0
            echo -e "${PURPLE}export CORE_PEER_LOCALMSPID=CSMSP${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ENABLED=true${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ROOTCERT_FILE=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem${NC}"
            echo -e "${PURPLE}export CORE_PEER_MSPCONFIGPATH=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp${NC}"
            echo -e "${PURPLE}export CORE_PEER_ADDRESS=localhost:9051${NC}"

            export CORE_PEER_LOCALMSPID=CSMSP
            export CORE_PEER_TLS_ENABLED=true
            export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem
            export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp
            export CORE_PEER_ADDRESS=localhost:9051
            echo ""
            echo -e "${GREEN}✅ Environment set for CS Peer 0 (localhost:9051)${NC}"
            ;;
        4)
            # CS Organization - Peer 1
            echo -e "${PURPLE}export CORE_PEER_LOCALMSPID=CSMSP${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ENABLED=true${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ROOTCERT_FILE=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem${NC}"
            echo -e "${PURPLE}export CORE_PEER_MSPCONFIGPATH=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp${NC}"
            echo -e "${PURPLE}export CORE_PEER_ADDRESS=localhost:10051${NC}"

            export CORE_PEER_LOCALMSPID=CSMSP
            export CORE_PEER_TLS_ENABLED=true
            export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem
            export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp
            export CORE_PEER_ADDRESS=localhost:10051
            echo ""
            echo -e "${GREEN}✅ Environment set for CS Peer 1 (localhost:10051)${NC}"
            ;;
        5)
            # CS Organization - Peer 2
            echo -e "${PURPLE}export CORE_PEER_LOCALMSPID=CSMSP${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ENABLED=true${NC}"
            echo -e "${PURPLE}export CORE_PEER_TLS_ROOTCERT_FILE=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem${NC}"
            echo -e "${PURPLE}export CORE_PEER_MSPCONFIGPATH=/home/yousif-ubunto/my-fabric-project/network/organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp${NC}"
            echo -e "${PURPLE}export CORE_PEER_ADDRESS=localhost:11051${NC}"

            export CORE_PEER_LOCALMSPID=CSMSP
            export CORE_PEER_TLS_ENABLED=true
            export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem
            export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/cs.example.com/users/Admin@cs.example.com/msp
            export CORE_PEER_ADDRESS=localhost:11051
            echo ""
            echo -e "${GREEN}✅ Environment set for CS Peer 2 (localhost:11051)${NC}"
            ;;
        *)
            echo -e "${RED}❌ Invalid peer selection${NC}"
            return 1
            ;;
    esac

    echo -e "${BLUE}Current peer: $CORE_PEER_LOCALMSPID @ $CORE_PEER_ADDRESS${NC}"
    echo ""
    return 0
}

# Function to show current environment status
show_current_env() {
    echo -e "${CYAN}Current Environment:${NC}"
    echo -e "${BLUE}  Organization: ${CORE_PEER_LOCALMSPID:-'Not Set'}${NC}"
    echo -e "${BLUE}  Peer Address: ${CORE_PEER_ADDRESS:-'Not Set'}${NC}"
    echo -e "${BLUE}  TLS Enabled: ${CORE_PEER_TLS_ENABLED:-'Not Set'}${NC}"
    echo ""
}

# =============================================================================
# MODULE MENU SYSTEM
# =============================================================================

# Function to display main module menu
display_main_menu() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Hyperledger Fabric Test Suite       ${NC}"
    echo -e "${GREEN}   Phase 6 Production Ready System     ${NC}"
    echo -e "${GREEN}========================================${NC}"
    show_current_env
    echo -e "${GREEN}🚀 Test Module Selection:${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - 📋 Document Lifecycle (Submit, Approve, Versioning)"
    echo -e "${YELLOW}2${NC} - 📊 Queries & Version History (Status, History, Versions)"
    echo -e "${YELLOW}3${NC} - 🔄 Ownership Transfer System (Request, Accept, Reject, Cancel) ✨"
    echo -e "${YELLOW}4${NC} - 🔄 Workflow Management (Configure, Status, Return to Stage)"
    echo -e "${YELLOW}5${NC} - 👥 Document Administration (Editors, Approvers, Permissions)"
    echo -e "${YELLOW}6${NC} - ⏰ Stage Deadline Management (Set, Get, Remove Stage Deadlines)"
    echo -e "${YELLOW}7${NC} - 🔍 Advanced Queries & Analytics (Search, Stats, Reports)"
    echo ""
    echo -e "${PURPLE}8${NC} - 🔧 Change Peer Environment"
    echo -e "${RED}9${NC} - 🚪 Exit"
    echo ""
}

# =============================================================================
# PEER ENVIRONMENT INITIALIZATION
# =============================================================================

echo -e "${GREEN}Welcome to the Document Management Chaincode Test Suite!${NC}"
echo -e "${YELLOW}Phase 6 Production Ready - Modular Architecture${NC}"
echo ""

# Check if environment variables are already set
if [[ -z "$CORE_PEER_LOCALMSPID" || -z "$CORE_PEER_ADDRESS" ]]; then
    echo -e "${YELLOW}No peer environment detected. Please select a peer to configure:${NC}"
    echo ""

    while true; do
        display_peer_menu
        read -p "Enter your choice [1-6]: " peer_choice

        case $peer_choice in
            [1-5])
                if set_peer_env "$peer_choice"; then
                    echo -e "${GREEN}Press Enter to continue to the main menu...${NC}"
                    read
                    break
                fi
                ;;
            6)
                echo -e "${YELLOW}Exiting script. Goodbye!${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}Invalid choice. Please select 1-6.${NC}"
                echo ""
                ;;
        esac
    done
else
    echo -e "${GREEN}Existing peer environment detected:${NC}"
    show_current_env

    read -p "Continue with current environment? [y/n]: " continue_choice
    if [[ "$continue_choice" != "y" && "$continue_choice" != "Y" ]]; then
        while true; do
            display_peer_menu
            read -p "Enter your choice [1-6]: " peer_choice

            case $peer_choice in
                [1-5])
                    if set_peer_env "$peer_choice"; then
                        echo -e "${GREEN}Press Enter to continue to the main menu...${NC}"
                        read
                        break
                    fi
                    ;;
                6)
                    echo -e "${YELLOW}Exiting script. Goodbye!${NC}"
                    exit 0
                    ;;
                *)
                    echo -e "${RED}Invalid choice. Please select 1-6.${NC}"
                    echo ""
                    ;;
            esac
        done
    fi
fi

# =============================================================================
# MAIN MENU LOOP
# =============================================================================

while true; do
    display_main_menu
    read -p "Enter your choice [1-9]: " choice

    case $choice in
        1)
            echo -e "${GREEN}Loading Document Lifecycle Module...${NC}"
            source "${LAUNCHER_DIR}/modules/document-lifecycle.sh"
            ;;
        2)
            echo -e "${GREEN}Loading Queries & Version History Module...${NC}"
            source "${LAUNCHER_DIR}/modules/queries-versioning.sh"
            ;;
        3)
            echo -e "${CYAN}Loading Ownership Transfer System Module...${NC}"
            source "${LAUNCHER_DIR}/modules/ownership-transfer.sh"
            ;;
        4)
            echo -e "${GREEN}Loading Workflow Management Module...${NC}"
            source "${LAUNCHER_DIR}/modules/workflow-management.sh"
            ;;
        5)
            echo -e "${PURPLE}Loading Document Administration Module...${NC}"
            source "${LAUNCHER_DIR}/modules/administration.sh"
            ;;
        6)
            echo -e "${PURPLE}Loading Deadline Management Module...${NC}"
            source "${LAUNCHER_DIR}/modules/deadline-management.sh"
            ;;
        7)
            echo -e "${BLUE}Loading Advanced Queries & Analytics Module...${NC}"
            source "${LAUNCHER_DIR}/modules/advanced-queries.sh"
            ;;
        8)
            echo ""
            while true; do
                display_peer_menu
                read -p "Enter your choice [1-6]: " peer_choice

                case $peer_choice in
                    [1-5])
                        if set_peer_env "$peer_choice"; then
                            echo -e "${GREEN}Press Enter to return to main menu...${NC}"
                            read
                            break
                        fi
                        ;;
                    6)
                        break
                        ;;
                    *)
                        echo -e "${RED}Invalid choice. Please select 1-6.${NC}"
                        echo ""
                        ;;
                esac
            done
            ;;
        9)
            echo -e "${GREEN}Exiting the test suite. Goodbye!${NC}"
            break
            ;;
        *)
            echo -e "${RED}Invalid choice. Please select 1-9.${NC}"
            echo ""
            ;;
    esac
done