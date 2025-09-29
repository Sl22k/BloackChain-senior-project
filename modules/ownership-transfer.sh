#!/bin/bash

# =============================================================================
# OWNERSHIP TRANSFER MODULE
# =============================================================================
# Phase 6 Advanced Feature: Complete document ownership transfer system
# Four-function architecture: Request/Accept/Reject/Cancel
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# OWNERSHIP TRANSFER MENU
# =============================================================================

display_ownership_menu() {
    echo -e "${CYAN}============================================${NC}"
    echo -e "${CYAN}   🔄 OWNERSHIP TRANSFER SYSTEM           ${NC}"
    echo -e "${CYAN}   Phase 6 Advanced Features ✨           ${NC}"
    echo -e "${CYAN}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select an ownership transfer function:${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - 📝 Request Ownership Transfer (2-step approval process)"
    echo -e "${YELLOW}2${NC} - ✅ Accept Ownership Transfer (Complete the transfer)"
    echo -e "${YELLOW}3${NC} - ❌ Reject Ownership Transfer (Decline with message)"
    echo -e "${YELLOW}4${NC} - 🚫 Cancel Ownership Transfer (Cancel your own request)"
    echo ""
    echo -e "${BLUE}5${NC} - 📊 Query Document Status (Check transfer status)"
    echo -e "${BLUE}6${NC} - 🔍 Get All Documents (List documents for transfer)"
    echo ""
    echo -e "${RED}7${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# OWNERSHIP TRANSFER FUNCTIONS
# =============================================================================

request_ownership_transfer() {
    echo -e "${GREEN}=== 🆕 Request Document Ownership Transfer ===${NC}"
    echo -e "${YELLOW}Phase 6 Feature: Two-step approval process with user-defined expiration${NC}"
    echo -e "${BLUE}Use Case: Transfer document to new owner with custom expiration period${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter new owner identity: " newOwner
    read -p "Enter transfer message/reason: " message
    read -p "Enter your identity (current owner/privileged editor): " invokerId

    echo ""
    echo -e "${BLUE}Set expiration period:${NC}"
    echo -e "${YELLOW}Valid range: 1-365 days${NC}"
    read -p "Expiration days (default 7): " expirationDays
    if [[ "$expirationDays" == "" ]]; then
        expirationDays="7"
    fi

    echo ""
    echo -e "${GREEN}=== Configuration Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}New Owner:${NC} $newOwner"
    echo -e "${BLUE}Current Owner:${NC} $invokerId"
    echo -e "${BLUE}Transfer Message:${NC} $message"
    echo -e "${BLUE}Expiration:${NC} $expirationDays days"
    echo ""

    while true; do
        read -p "Proceed with this ownership transfer request? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Request cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "RequestOwnershipTransfer" "$documentID" "$newOwner" "$message" "$invokerId" "$expirationDays"
}

accept_ownership_transfer() {
    echo -e "${GREEN}=== ✅ Accept Ownership Transfer Request ===${NC}"
    echo -e "${YELLOW}Phase 6 Feature: Accept pending ownership transfer${NC}"
    echo -e "${BLUE}Use Case: Accept a transfer request made to you${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter your identity (new owner): " invokerId
    echo ""

    echo -e "${BLUE}Editor Management Options:${NC}"
    echo -e "${YELLOW}• Reset editors: Only you will be an editor${NC}"
    echo -e "${YELLOW}• Keep editors: Existing editors remain${NC}"
    echo ""

    resetEditors=$(collect_boolean_input "Reset editor list to only include you?")

    echo ""
    echo -e "${GREEN}=== Configuration Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}New Owner:${NC} $invokerId"
    echo -e "${BLUE}Reset Editors:${NC} $resetEditors"
    echo ""

    while true; do
        read -p "Proceed with accepting this transfer? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Accept cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "AcceptOwnershipTransfer" "$documentID" "$invokerId" "$resetEditors"
}

reject_ownership_transfer() {
    echo -e "${RED}=== ❌ Reject Ownership Transfer Request ===${NC}"
    echo -e "${YELLOW}Phase 6 Feature: Reject pending ownership transfer${NC}"
    echo -e "${BLUE}Use Case: Decline a transfer request made to you${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter rejection reason/message: " rejectionMessage
    read -p "Enter your identity (new owner): " invokerId

    echo ""
    echo -e "${GREEN}=== Configuration Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}Your Identity:${NC} $invokerId"
    echo -e "${BLUE}Rejection Message:${NC} $rejectionMessage"
    echo ""

    while true; do
        read -p "Proceed with rejecting this transfer? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Rejection cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "RejectOwnershipTransfer" "$documentID" "$rejectionMessage" "$invokerId"
}

cancel_ownership_transfer() {
    echo -e "${YELLOW}=== 🚫 Cancel Ownership Transfer Request ===${NC}"
    echo -e "${YELLOW}Phase 6 Feature: Cancel your own transfer request${NC}"
    echo -e "${BLUE}Use Case: Cancel a transfer request you previously made${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter your identity (current owner/requester): " invokerId

    echo ""
    echo -e "${GREEN}=== Configuration Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}Your Identity:${NC} $invokerId"
    echo ""

    while true; do
        read -p "Proceed with cancelling this transfer request? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Cancellation cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "CancelOwnershipTransfer" "$documentID" "$invokerId"
}

query_document_status() {
    echo -e "${BLUE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}Check document status including any pending ownership transfers${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "QueryDocumentStatus" "$documentID"
}

get_all_documents() {
    echo -e "${BLUE}=== 🔍 Get All Documents ===${NC}"
    echo -e "${YELLOW}List all documents to see which ones you can transfer${NC}"
    echo ""

    execute_chaincode_function "GetAllDocuments"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_ownership_menu
    read -p "Enter your choice [1-7]: " choice

    case $choice in
        1)
            request_ownership_transfer
            ;;
        2)
            accept_ownership_transfer
            ;;
        3)
            reject_ownership_transfer
            ;;
        4)
            cancel_ownership_transfer
            ;;
        5)
            query_document_status
            ;;
        6)
            get_all_documents
            ;;
        7)
            return_to_main_menu
            break
            ;;
        *)
            echo -e "${RED}Invalid choice. Please select 1-7.${NC}"
            echo ""
            ;;
    esac
done