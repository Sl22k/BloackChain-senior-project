#!/bin/bash

# =============================================================================
# ADMINISTRATION MODULE
# =============================================================================
# Document administration: Editors, Approvers, Permissions
# Administrative functions for document access control
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# ADMINISTRATION MENU
# =============================================================================

display_admin_menu() {
    echo -e "${PURPLE}============================================${NC}"
    echo -e "${PURPLE}   👥 DOCUMENT ADMINISTRATION            ${NC}"
    echo -e "${PURPLE}   Editors, Approvers & Permissions      ${NC}"
    echo -e "${PURPLE}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select an administration function:${NC}"
    echo ""
    echo -e "${YELLOW}👤 Editor Management:${NC}"
    echo -e "${YELLOW}1${NC} - ➕ Add Editor (Grant editing permissions)"
    echo -e "${YELLOW}2${NC} - ➖ Remove Editor (Revoke editing permissions)"
    echo -e "${YELLOW}3${NC} - ⭐ Add Privileged Editor (Grant privileged access)"
    echo -e "${YELLOW}4${NC} - 🚫 Remove Privileged Editor (Revoke privileged access)"
    echo ""
    echo -e "${CYAN}✅ Approver Management:${NC}"
    echo -e "${CYAN}5${NC} - 🔄 Update Document Approvers (Change approver list)"
    echo ""
    echo -e "${BLUE}📊 Administrative Queries:${NC}"
    echo -e "${BLUE}6${NC} - 👤 Get Documents by Editor (Documents user can edit)"
    echo -e "${BLUE}7${NC} - ✅ Get Documents by Approver (Documents user can approve)"
    echo -e "${BLUE}8${NC} - 🔔 Get Pending Approvals (Pending tasks for user)"
    echo ""
    echo -e "${GREEN}📋 Document Information:${NC}"
    echo -e "${GREEN}9${NC} - 📊 Query Document Status (Check document details)"
    echo -e "${GREEN}10${NC} - 📝 Get Document Summary (Summary information)"
    echo ""
    echo -e "${RED}11${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# EDITOR MANAGEMENT FUNCTIONS
# =============================================================================

add_editor() {
    echo -e "${YELLOW}=== ➕ Add Editor ===${NC}"
    echo -e "${BLUE}Grant editing permissions to a new user${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter new editor identity: " newEditor
    read -p "Enter your identity (invoker): " invokerId

    echo ""
    echo -e "${GREEN}=== Editor Addition Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}New Editor:${NC} $newEditor"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo ""

    while true; do
        read -p "Proceed with adding this editor? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Editor addition cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "AddEditor" "$documentID" "$newEditor" "$invokerId"
}

remove_editor() {
    echo -e "${YELLOW}=== ➖ Remove Editor ===${NC}"
    echo -e "${BLUE}Revoke editing permissions from a user${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter editor to remove: " editorToRemove
    read -p "Enter your identity (invoker): " invokerId

    echo ""
    echo -e "${GREEN}=== Editor Removal Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}Editor to Remove:${NC} $editorToRemove"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo ""

    while true; do
        read -p "Proceed with removing this editor? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Editor removal cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "RemoveEditor" "$documentID" "$editorToRemove" "$invokerId"
}

add_privileged_editor() {
    echo -e "${YELLOW}=== ⭐ Add Privileged Editor ===${NC}"
    echo -e "${BLUE}Grant privileged editing access (higher permissions)${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter new privileged editor identity: " newPrivilegedEditor
    read -p "Enter your identity (invoker): " invokerId

    echo ""
    echo -e "${GREEN}=== Privileged Editor Addition Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}New Privileged Editor:${NC} $newPrivilegedEditor"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo ""

    while true; do
        read -p "Proceed with adding this privileged editor? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Privileged editor addition cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "AddPrivilegedEditor" "$documentID" "$newPrivilegedEditor" "$invokerId"
}

remove_privileged_editor() {
    echo -e "${YELLOW}=== 🚫 Remove Privileged Editor ===${NC}"
    echo -e "${BLUE}Revoke privileged editing access${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter privileged editor to remove: " privilegedEditorToRemove
    read -p "Enter your identity (invoker): " invokerId

    echo ""
    echo -e "${GREEN}=== Privileged Editor Removal Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}Privileged Editor to Remove:${NC} $privilegedEditorToRemove"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo ""

    while true; do
        read -p "Proceed with removing this privileged editor? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Privileged editor removal cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "RemovePrivilegedEditor" "$documentID" "$privilegedEditorToRemove" "$invokerId"
}

# =============================================================================
# APPROVER MANAGEMENT FUNCTIONS
# =============================================================================

update_document_approvers() {
    echo -e "${CYAN}=== 🔄 Update Document Approvers (Sophisticated Stage-Based) ===${NC}"
    echo -e "${BLUE}Update approvers for specific workflow stages${NC}"
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

    # Extract stage information
    stage_count=$(echo "$current_doc_result" | jq '.Workflow.Stages | length')

    echo ""
    echo -e "${BLUE}Current document has $stage_count stage(s):${NC}"
    for i in $(seq 1 $stage_count); do
        stage_name=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((i-1))].StageName")
        stage_approvers=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((i-1))].Approvers | join(\",\")")
        stage_required=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((i-1))].RequiredCount")
        stage_auto=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((i-1))].AutoAdvance")
        echo -e "  ${YELLOW}Stage $i:${NC} $stage_name"
        echo -e "    Approvers: $stage_approvers"
        echo -e "    Required: $stage_required, Auto-advance: $stage_auto"
    done

    echo ""
    read -p "Which stages to update approvers for (comma-separated, e.g., 1,3 or 'all'): " stages_to_update

    # Build stageUpdates JSON
    stage_updates="{}"
    if [[ "$stages_to_update" == "all" ]]; then
        stages_to_update=$(seq -s, 1 $stage_count)
    fi

    if [[ "$stages_to_update" != "none" && -n "$stages_to_update" ]]; then
        for stage_num in $(echo $stages_to_update | tr ',' ' '); do
            echo ""
            echo -e "${GREEN}=== Stage $stage_num Approver Updates ===${NC}"

            # Get current stage info
            current_approvers=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((stage_num-1))].Approvers | join(\",\")")
            current_required=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((stage_num-1))].RequiredCount")
            current_auto=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((stage_num-1))].AutoAdvance")

            echo -e "${YELLOW}Current approvers:${NC} $current_approvers"
            echo -e "${YELLOW}Current required:${NC} $current_required"
            echo -e "${YELLOW}Current auto-advance:${NC} $current_auto"
            echo ""

            # UX-friendly approver selection
            echo -e "${GREEN}=== Select New Approvers for Stage $stage_num ===${NC}"
            echo -e "${YELLOW}1${NC} - Legal Team (legal1, legal2, legal3)"
            echo -e "${YELLOW}2${NC} - Technical Team (dev1, dev2, architect1)"
            echo -e "${YELLOW}3${NC} - Executive Team (ceo, cfo, cto)"
            echo -e "${YELLOW}4${NC} - Mixed Teams (combine multiple groups)"
            echo -e "${YELLOW}5${NC} - Keep current approvers unchanged"
            echo -e "${YELLOW}6${NC} - Custom (enter your own)"
            echo ""

            while true; do
                read -p "Choice [1-6]: " approver_choice
                case $approver_choice in
                    1) approvers_json='["legal1","legal2","legal3"]'; break;;
                    2) approvers_json='["dev1","dev2","architect1"]'; break;;
                    3) approvers_json='["ceo","cfo","cto"]'; break;;
                    4)
                        echo -e "${GREEN}=== Select Multiple Teams ===${NC}"
                        echo "Select which teams to include (space-separated numbers, e.g. '1 2' for Legal+Technical):"
                        echo -e "${YELLOW}1${NC} - Legal Team"
                        echo -e "${YELLOW}2${NC} - Technical Team"
                        echo -e "${YELLOW}3${NC} - Executive Team"
                        read -p "Teams to include: " team_selection

                        combined_approvers=""
                        for team in $team_selection; do
                            case $team in
                                1) combined_approvers="$combined_approvers,legal1,legal2,legal3";;
                                2) combined_approvers="$combined_approvers,dev1,dev2,architect1";;
                                3) combined_approvers="$combined_approvers,ceo,cfo,cto";;
                            esac
                        done
                        # Remove leading comma and convert to JSON
                        combined_approvers=$(echo "$combined_approvers" | sed 's/^,//')
                        if [[ -n "$combined_approvers" ]]; then
                            approvers_json="[\"$(echo "$combined_approvers" | sed 's/,/","/g')\"]"
                            break
                        else
                            echo -e "${RED}No valid teams selected, try again${NC}"
                        fi
                        ;;
                    5)
                        # Keep current approvers - convert from comma-separated to JSON
                        approvers_json="[\"$(echo "$current_approvers" | sed 's/,/","/g')\"]"
                        break;;
                    6)
                        read -p "Enter approvers (comma-separated): " custom_approvers
                        if [[ -n "$custom_approvers" ]]; then
                            approvers_json="[\"$(echo "$custom_approvers" | tr -d ' ' | sed 's/,/","/g')\"]"
                            break
                        else
                            echo -e "${RED}Please enter at least one approver${NC}"
                        fi
                        ;;
                    *) echo -e "${RED}Invalid choice 1-6${NC}";;
                esac
            done

            echo -e "${GREEN}Selected approvers:${NC} $(echo "$approvers_json" | tr -d '[]"' | sed 's/,/, /g')"
            echo ""

            read -p "Required approvals (0=all, current: $current_required): " required_count
            read -p "Auto-advance (true/false, current: $current_auto): " auto_advance

            # Build stage update object
            stage_updates=$(echo "$stage_updates" | jq --arg stage "$stage_num" \
                --argjson approvers "$approvers_json" \
                --argjson required "$required_count" \
                --argjson auto "$auto_advance" \
                '.[$stage] = {"approvers": $approvers, "requiredCount": ($required | tonumber), "autoAdvance": $auto}')
        done
    fi

    # Reset Approvals Selection
    echo ""
    echo -e "${GREEN}=== Reset Approvals Setting ===${NC}"
    echo -e "${YELLOW}Current approvals will be:${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - Reset to PENDING (recommended for approver changes)"
    echo -e "${YELLOW}2${NC} - Keep existing approvals (only for non-disruptive changes)"
    echo ""

    while true; do
        read -p "Choice [1-2]: " reset_choice
        case $reset_choice in
            1)
                resetApprovals="true"
                echo -e "${GREEN}Selected:${NC} Reset all approvals to PENDING"
                break;;
            2)
                resetApprovals="false"
                echo -e "${GREEN}Selected:${NC} Keep existing approvals"
                echo -e "${YELLOW}Note:${NC} Only works if no approvers are removed"
                break;;
            *) echo -e "${RED}Invalid choice. Please select 1 or 2${NC}";;
        esac
    done

    echo ""
    echo -e "${GREEN}=== Approver Update Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo -e "${BLUE}Reset Approvals:${NC} $resetApprovals"
    echo ""
    echo -e "${PURPLE}Generated StageUpdates JSON:${NC}"
    echo "$stage_updates" | jq .
    echo ""

    while true; do
        read -p "Proceed with this approver update? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Approver update cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    # Execute with correct signature: documentID, stageUpdatesJSON, invokerId, resetApprovals
    compactStageUpdates=$(echo "$stage_updates" | jq -c .)
    execute_chaincode_function "UpdateDocumentApprovers" "$documentID" "$compactStageUpdates" "$invokerId" "$resetApprovals"
}

# =============================================================================
# ADMINISTRATIVE QUERIES
# =============================================================================

get_documents_by_editor() {
    echo -e "${BLUE}=== 👤 Get Documents by Editor ===${NC}"
    echo -e "${YELLOW}Find all documents a user can edit${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetDocumentsByEditor" "$userId"
}

get_documents_by_approver() {
    echo -e "${BLUE}=== ✅ Get Documents by Approver ===${NC}"
    echo -e "${YELLOW}Find all documents a user can approve${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetDocumentsByApprover" "$userId"
}

get_pending_approvals() {
    echo -e "${BLUE}=== 🔔 Get Pending Approvals ===${NC}"
    echo -e "${YELLOW}Get all pending approval tasks for a user${NC}"
    echo ""

    read -p "Enter user identity: " userId

    execute_chaincode_function "GetPendingApprovals" "$userId"
}

# =============================================================================
# DOCUMENT INFORMATION
# =============================================================================

query_document_status() {
    echo -e "${GREEN}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}Get complete document information and status${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "QueryDocumentStatus" "$documentID"
}

get_document_summary() {
    echo -e "${GREEN}=== 📝 Get Document Summary ===${NC}"
    echo -e "${YELLOW}Get condensed document information${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "GetDocumentSummary" "$documentID"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_admin_menu
    read -p "Enter your choice [1-11]: " choice

    case $choice in
        1)
            add_editor
            ;;
        2)
            remove_editor
            ;;
        3)
            add_privileged_editor
            ;;
        4)
            remove_privileged_editor
            ;;
        5)
            update_document_approvers
            ;;
        6)
            get_documents_by_editor
            ;;
        7)
            get_documents_by_approver
            ;;
        8)
            get_pending_approvals
            ;;
        9)
            query_document_status
            ;;
        10)
            get_document_summary
            ;;
        11)
            return_to_main_menu
            break
            ;;
        *)
            echo -e "${RED}Invalid choice. Please select 1-11.${NC}"
            echo ""
            ;;
    esac
done