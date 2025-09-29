#!/bin/bash

# =============================================================================
# QUERIES & VERSIONING MODULE
# =============================================================================
# Phase 6 Enhanced query functions and version history management
# Separated from status queries for performance optimization
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# QUERIES & VERSIONING MENU
# =============================================================================

display_queries_menu() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}   📊 QUERIES & VERSION HISTORY          ${NC}"
    echo -e "${BLUE}   Phase 6 Enhanced Query System         ${NC}"
    echo -e "${BLUE}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select a query or versioning function:${NC}"
    echo ""
    echo -e "${YELLOW}📋 Document Status & Information:${NC}"
    echo -e "${YELLOW}1${NC} - 📊 Query Document Status (Primary status query)"
    echo -e "${YELLOW}2${NC} - 🔍 Get All Documents (List all documents)"
    echo -e "${YELLOW}3${NC} - 📝 Get Document Summary (Summary information)"
    echo -e "${YELLOW}4${NC} - ✅ Document Exists (Check existence)"
    echo ""
    echo -e "${CYAN}📚 Version History & Management (Phase 6):${NC}"
    echo -e "${CYAN}5${NC} - 📜 Get Document History (Complete version history) ✨"
    echo -e "${CYAN}6${NC} - 🎯 Get Document Version (Specific version details) ✨"
    echo -e "${CYAN}7${NC} - 🔄 Get Document Version Workflow (Version workflow state) ✨"
    echo ""
    echo -e "${GREEN}🔍 Advanced Workflow Queries:${NC}"
    echo -e "${GREEN}8${NC} - 🔄 Get Workflow Status (Current workflow state)"
    echo -e "${GREEN}9${NC} - 📈 Get Document Workflow History (Complete workflow history)"
    echo -e "${GREEN}10${NC} - 🎯 Get Version Workflow State (Version-specific workflow)"
    echo ""
    echo -e "${PURPLE}👤 User-Based Queries:${NC}"
    echo -e "${PURPLE}11${NC} - 👤 Get Documents by Editor (Documents you can edit)"
    echo -e "${PURPLE}12${NC} - ✅ Get Documents by Approver (Documents you can approve)"
    echo -e "${PURPLE}13${NC} - 🔔 Get Pending Approvals (Your pending approvals)"
    echo ""
    echo -e "${BLUE}🔧 Utility Functions:${NC}"
    echo -e "${BLUE}14${NC} - 🔍 Get Document ID by Hash (Find document by hash)"
    echo ""
    echo -e "${RED}15${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# DOCUMENT STATUS & INFORMATION FUNCTIONS
# =============================================================================

query_document_status() {
    echo -e "${BLUE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}🎯 PRIMARY: Complete document status & workflow information${NC}"
    echo -e "${BLUE}Optimized for fast status queries (Phase 6 performance improvement)${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "QueryDocumentStatus" "$id"
}

get_all_documents() {
    echo -e "${BLUE}=== 🔍 Get All Documents ===${NC}"
    echo -e "${YELLOW}List all documents in the system${NC}"
    echo ""

    execute_chaincode_function "GetAllDocuments"
}

get_document_summary() {
    echo -e "${BLUE}=== 📝 Get Document Summary ===${NC}"
    echo -e "${YELLOW}Get condensed document information${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "GetDocumentSummary" "$id"
}

document_exists() {
    echo -e "${BLUE}=== ✅ Document Exists ===${NC}"
    echo -e "${YELLOW}Check if document exists in the system${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "DocumentExists" "$id"
}

# =============================================================================
# VERSION HISTORY & MANAGEMENT FUNCTIONS (Phase 6)
# =============================================================================

get_document_history() {
    echo -e "${CYAN}=== 📜 Get Document History ===${NC}"
    echo -e "${YELLOW}🆕 Phase 6: Complete version history with performance optimization${NC}"
    echo -e "${BLUE}Separated from status queries for better performance${NC}"
    echo ""

    read -p "Enter document ID: " id

    echo ""
    echo -e "${GREEN}=== Version History Query ===${NC}"
    echo -e "${BLUE}Document:${NC} $id"
    echo -e "${BLUE}Query Type:${NC} Complete version history"
    echo ""

    execute_chaincode_function "GetDocumentHistory" "$id"
}

get_document_version() {
    echo -e "${CYAN}=== 🎯 Get Document Version ===${NC}"
    echo -e "${YELLOW}🆕 Phase 6: Specific version lookup with detailed information${NC}"
    echo -e "${BLUE}Granular version access for detailed analysis${NC}"
    echo ""

    read -p "Enter document ID: " id
    read -p "Enter version number: " version

    echo ""
    echo -e "${GREEN}=== Version Lookup ===${NC}"
    echo -e "${BLUE}Document:${NC} $id"
    echo -e "${BLUE}Version:${NC} $version"
    echo -e "${BLUE}Query Type:${NC} Specific version details"
    echo ""

    execute_chaincode_function "GetDocumentVersion" "$id" "$version"
}

get_document_version_workflow() {
    echo -e "${CYAN}=== 🔄 Get Document Version Workflow ===${NC}"
    echo -e "${YELLOW}🆕 Phase 6: Workflow state for specific document version${NC}"
    echo -e "${BLUE}Version-specific workflow analysis${NC}"
    echo ""

    read -p "Enter document ID: " id
    read -p "Enter version number: " version

    echo ""
    echo -e "${GREEN}=== Version Workflow Query ===${NC}"
    echo -e "${BLUE}Document:${NC} $id"
    echo -e "${BLUE}Version:${NC} $version"
    echo -e "${BLUE}Query Type:${NC} Version workflow state"
    echo ""

    execute_chaincode_function "GetDocumentVersionWorkflow" "$id" "$version"
}

# =============================================================================
# ADVANCED WORKFLOW QUERIES
# =============================================================================

get_workflow_status() {
    echo -e "${GREEN}=== 🔄 Get Workflow Status ===${NC}"
    echo -e "${YELLOW}Current workflow state and stage information${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "GetWorkflowStatus" "$id"
}

get_document_workflow_history() {
    echo -e "${GREEN}=== 📈 Get Document Workflow History ===${NC}"
    echo -e "${YELLOW}Complete workflow progression across all versions${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "GetDocumentWorkflowHistory" "$id"
}

get_version_workflow_state() {
    echo -e "${GREEN}=== 🎯 Get Version Workflow State ===${NC}"
    echo -e "${YELLOW}Detailed workflow state for specific document version${NC}"
    echo ""

    read -p "Enter document ID: " id
    read -p "Enter version number: " version

    execute_chaincode_function "GetVersionWorkflowState" "$id" "$version"
}

# =============================================================================
# USER-BASED QUERIES
# =============================================================================

get_documents_by_editor() {
    echo -e "${PURPLE}=== 👤 Get Documents by Editor ===${NC}"
    echo -e "${YELLOW}Documents you can edit${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetDocumentsByEditor" "$userId"
}

get_documents_by_approver() {
    echo -e "${PURPLE}=== ✅ Get Documents by Approver ===${NC}"
    echo -e "${YELLOW}Documents you can approve${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetDocumentsByApprover" "$userId"
}

get_pending_approvals() {
    echo -e "${PURPLE}=== 🔔 Get Pending Approvals ===${NC}"
    echo -e "${YELLOW}Your pending approval tasks${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetPendingApprovals" "$userId"
}

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

get_document_id_by_hash() {
    echo -e "${BLUE}=== 🔍 Get Document ID by Hash ===${NC}"
    echo -e "${YELLOW}Find document by its content hash${NC}"
    echo ""

    read -p "Enter document hash: " hash

    execute_chaincode_function "GetDocumentIdByHash" "$hash"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_queries_menu
    read -p "Enter your choice [1-15]: " choice

    case $choice in
        1)
            query_document_status
            ;;
        2)
            get_all_documents
            ;;
        3)
            get_document_summary
            ;;
        4)
            document_exists
            ;;
        5)
            get_document_history
            ;;
        6)
            get_document_version
            ;;
        7)
            get_document_version_workflow
            ;;
        8)
            get_workflow_status
            ;;
        9)
            get_document_workflow_history
            ;;
        10)
            get_version_workflow_state
            ;;
        11)
            get_documents_by_editor
            ;;
        12)
            get_documents_by_approver
            ;;
        13)
            get_pending_approvals
            ;;
        14)
            get_document_id_by_hash
            ;;
        15)
            return_to_main_menu
            break
            ;;
        *)
            echo -e "${RED}Invalid choice. Please select 1-15.${NC}"
            echo ""
            ;;
    esac
done