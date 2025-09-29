#!/bin/bash

# =============================================================================
# ADVANCED QUERIES MODULE
# =============================================================================
# Advanced query functions from advanced_query_handlers.go
# Date ranges, statistics, sorting, and analytics
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# ADVANCED QUERIES MENU
# =============================================================================

display_advanced_queries_menu() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}   🔍 ADVANCED QUERIES & ANALYTICS        ${NC}"
    echo -e "${BLUE}   Comprehensive Search & Statistics      ${NC}"
    echo -e "${BLUE}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select an advanced query function:${NC}"
    echo ""
    echo -e "${YELLOW}📅 Date & Time Queries:${NC}"
    echo -e "${YELLOW}1${NC} - 📅 Get Documents by Date Range (Filter by creation date)"
    echo -e "${YELLOW}2${NC} - 🕒 Get Recent Documents (Latest documents with limit)"
    echo ""
    echo -e "${CYAN}👤 User & Sorting Queries:${NC}"
    echo -e "${CYAN}3${NC} - 👤 Get Documents by Uploader Sorted (Sorted by user)"
    echo -e "${CYAN}4${NC} - 🔔 Get Documents with Pending Approvals (System-wide pending)"
    echo ""
    echo -e "${PURPLE}📊 Statistics & Analytics:${NC}"
    echo -e "${PURPLE}5${NC} - 📊 Get Document Stats (System-wide statistics)"
    echo ""
    echo -e "${GREEN}🔧 Administrative Functions:${NC}"
    echo -e "${GREEN}6${NC} - ⚙️  Update Valid Decisions (Modify decision options)"
    echo ""
    echo -e "${BLUE}🔍 Legacy & Utility Functions:${NC}"
    echo -e "${BLUE}7${NC} - 📜 Get Doc History (Legacy version history)"
    echo -e "${BLUE}8${NC} - 🔍 Get Document ID by Hash (Find by hash)"
    echo ""
    echo -e "${RED}9${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# DATE & TIME QUERIES
# =============================================================================

get_documents_by_date_range() {
    echo -e "${YELLOW}=== 📅 Get Documents by Date Range ===${NC}"
    echo -e "${BLUE}Filter documents by creation date range${NC}"
    echo ""

    echo -e "${GREEN}Enter date range (YYYY-MM-DD format):${NC}"
    read -p "Start date (YYYY-MM-DD): " startDate
    read -p "End date (YYYY-MM-DD): " endDate

    # Validate date format
    if ! [[ "$startDate" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || ! [[ "$endDate" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        echo -e "${RED}❌ Invalid date format. Please use YYYY-MM-DD${NC}"
        return
    fi

    echo ""
    echo -e "${GREEN}=== Query Preview ===${NC}"
    echo -e "${BLUE}Start Date:${NC} $startDate"
    echo -e "${BLUE}End Date:${NC} $endDate"
    echo -e "${BLUE}Query Type:${NC} Date range filter"
    echo ""

    execute_chaincode_function "GetDocumentsByDateRange" "$startDate" "$endDate"
}

get_recent_documents() {
    echo -e "${YELLOW}=== 🕒 Get Recent Documents ===${NC}"
    echo -e "${BLUE}Get the most recently modified documents${NC}"
    echo ""

    read -p "Enter limit (number of documents, default 10): " limit
    if [[ "$limit" == "" ]]; then
        limit="10"
    fi

    # Validate limit is a number
    if ! [[ "$limit" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}❌ Limit must be a positive number${NC}"
        return
    fi

    echo ""
    echo -e "${GREEN}=== Query Preview ===${NC}"
    echo -e "${BLUE}Limit:${NC} $limit documents"
    echo -e "${BLUE}Sort Order:${NC} Most recent first"
    echo ""

    execute_chaincode_function "GetRecentDocuments" "$limit"
}

# =============================================================================
# USER & SORTING QUERIES
# =============================================================================

get_documents_by_uploader_sorted() {
    echo -e "${CYAN}=== 👤 Get Documents by Uploader Sorted ===${NC}"
    echo -e "${BLUE}Get documents by specific uploader with sorting options${NC}"
    echo ""

    read -p "Enter uploader identity: " uploader
    echo ""
    echo -e "${GREEN}Sort Options:${NC}"
    echo -e "${YELLOW}1${NC} - Descending (newest first)"
    echo -e "${YELLOW}2${NC} - Ascending (oldest first)"
    echo ""
    read -p "Choose sort order [1-2]: " sort_choice

    case $sort_choice in
        1) sortOrder="desc";;
        2) sortOrder="asc";;
        *)
            echo -e "${YELLOW}Using default: descending${NC}"
            sortOrder="desc";;
    esac

    echo ""
    echo -e "${GREEN}=== Query Preview ===${NC}"
    echo -e "${BLUE}Uploader:${NC} $uploader"
    echo -e "${BLUE}Sort Order:${NC} $sortOrder"
    echo ""

    execute_chaincode_function "GetDocumentsByUploaderSorted" "$uploader" "$sortOrder"
}

get_documents_with_pending_approvals() {
    echo -e "${CYAN}=== 🔔 Get Documents with Pending Approvals ===${NC}"
    echo -e "${BLUE}System-wide search for documents needing approvals${NC}"
    echo ""

    echo -e "${GREEN}This will search all documents that have pending approvals.${NC}"
    echo ""

    execute_chaincode_function "GetDocumentsWithPendingApprovals"
}

# =============================================================================
# STATISTICS & ANALYTICS
# =============================================================================

get_document_stats() {
    echo -e "${PURPLE}=== 📊 Get Document Stats ===${NC}"
    echo -e "${BLUE}System-wide document statistics and analytics${NC}"
    echo ""

    echo -e "${GREEN}Generating comprehensive system statistics...${NC}"
    echo ""

    execute_chaincode_function "GetDocumentStats"
}

# =============================================================================
# ADMINISTRATIVE FUNCTIONS
# =============================================================================

update_valid_decisions() {
    echo -e "${GREEN}=== ⚙️  Update Valid Decisions (Sophisticated Interface) ===${NC}"
    echo -e "${BLUE}Modify the valid decision options for a document${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    # Get current document configuration
    echo -e "${BLUE}Fetching current document configuration...${NC}"
    current_doc_result=$(peer chaincode query -C mychannel -n documentApproval -c "{\"Function\":\"QueryDocumentStatus\",\"Args\":[\"$documentID\"]}" 2>/dev/null)

    if [[ $? -ne 0 || -z "$current_doc_result" ]]; then
        echo -e "${RED}Error: Could not fetch document $documentID${NC}"
        return
    fi

    # UX-friendly valid decisions selection
    echo ""
    echo -e "${GREEN}=== Select Valid Decisions for Document ===${NC}"
    echo -e "${YELLOW}Current valid decisions:${NC} $(echo "$current_doc_result" | jq -r '.ValidDecisions | join(", ")')"
    echo ""
    echo -e "${YELLOW}1${NC} - Keep current decisions unchanged"
    echo -e "${YELLOW}2${NC} - Standard (APPROVED, REJECTED)"
    echo -e "${YELLOW}3${NC} - Extended (APPROVED, REJECTED, NEEDS_REVISION)"
    echo -e "${YELLOW}4${NC} - Comprehensive (APPROVED, REJECTED, NEEDS_REVISION, ON_HOLD)"
    echo -e "${YELLOW}5${NC} - Approval Only (APPROVED) - Single decision type"
    echo -e "${YELLOW}6${NC} - Binary Plus (APPROVED, REJECTED, ABSTAIN)"
    echo -e "${YELLOW}7${NC} - Management Set (APPROVED, REJECTED, DEFER, ESCALATE)"
    echo -e "${YELLOW}8${NC} - Custom (enter your own)"
    echo ""

    while true; do
        read -p "Choice [1-8]: " decision_choice
        case $decision_choice in
            1)
                validDecisions=$(echo "$current_doc_result" | jq -c '.ValidDecisions')
                break;;
            2)
                validDecisions='["APPROVED","REJECTED"]'
                break;;
            3)
                validDecisions='["APPROVED","REJECTED","NEEDS_REVISION"]'
                break;;
            4)
                validDecisions='["APPROVED","REJECTED","NEEDS_REVISION","ON_HOLD"]'
                break;;
            5)
                validDecisions='["APPROVED"]'
                break;;
            6)
                validDecisions='["APPROVED","REJECTED","ABSTAIN"]'
                break;;
            7)
                validDecisions='["APPROVED","REJECTED","DEFER","ESCALATE"]'
                break;;
            8)
                read -p "Enter valid decisions (comma-separated): " custom_decisions
                if [[ -n "$custom_decisions" ]]; then
                    validDecisions="[\"$(echo "$custom_decisions" | tr -d ' ' | sed 's/,/","/g')\"]"
                    break
                else
                    echo -e "${RED}Please enter at least one valid decision${NC}"
                fi
                ;;
            *) echo -e "${RED}Invalid choice 1-8${NC}";;
        esac
    done

    if [[ -z "$validDecisions" || "$validDecisions" == "null" ]]; then
        validDecisions='["APPROVED","REJECTED"]'
    fi

    echo -e "${GREEN}Selected decisions:${NC} $(echo "$validDecisions" | tr -d '[]"' | sed 's/,/, /g')"

    echo ""
    echo -e "${GREEN}=== Valid Decisions Update Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo -e "${BLUE}Current Decisions:${NC} $(echo "$current_doc_result" | jq -r '.ValidDecisions | join(", ")')"
    echo -e "${BLUE}New Decisions:${NC} $(echo "$validDecisions" | tr -d '[]"' | sed 's/,/, /g')"
    echo ""
    echo -e "${PURPLE}Generated ValidDecisions JSON:${NC}"
    echo "$validDecisions" | jq .
    echo ""
    echo -e "${YELLOW}Note:${NC} This will create a new document version with updated decision options only"
    echo -e "${YELLOW}      Content hash and workflow stages remain unchanged${NC}"
    echo ""

    while true; do
        read -p "Proceed with this valid decisions update? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Valid decisions update cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    # Execute with correct signature: documentID, newDecisionsJSON, invokerId
    compactValidDecisions=$(echo "$validDecisions" | jq -c .)
    execute_chaincode_function "UpdateValidDecisions" "$documentID" "$compactValidDecisions" "$invokerId"
}

# =============================================================================
# LEGACY & UTILITY FUNCTIONS
# =============================================================================

get_doc_history() {
    echo -e "${BLUE}=== 📜 Get Doc History ===${NC}"
    echo -e "${YELLOW}Legacy version history function (different from GetDocumentHistory)${NC}"
    echo ""

    read -p "Enter document ID: " id

    execute_chaincode_function "GetDocHistory" "$id"
}

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
    display_advanced_queries_menu
    read -p "Enter your choice [1-9]: " choice

    case $choice in
        1)
            get_documents_by_date_range
            ;;
        2)
            get_recent_documents
            ;;
        3)
            get_documents_by_uploader_sorted
            ;;
        4)
            get_documents_with_pending_approvals
            ;;
        5)
            get_document_stats
            ;;
        6)
            update_valid_decisions
            ;;
        7)
            get_doc_history
            ;;
        8)
            get_document_id_by_hash
            ;;
        9)
            return_to_main_menu
            break
            ;;
        *)
            echo -e "${RED}Invalid choice. Please select 1-9.${NC}"
            echo ""
            ;;
    esac
done