#!/bin/bash

# =============================================================================
# DOCUMENT LIFECYCLE MODULE - PHASE 6 NATIVE
# =============================================================================
# Clean, intuitive implementation that directly matches chaincode expectations
# No legacy conversions - pure Phase 6 architecture
# =============================================================================

# Source Phase 6 native functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions-v2.sh"

# =============================================================================
# DOCUMENT LIFECYCLE MENU
# =============================================================================

display_lifecycle_menu() {
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}   📋 DOCUMENT LIFECYCLE (PHASE 6)        ${NC}"
    echo -e "${GREEN}   Native Implementation                    ${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - 🆕 Initialize Ledger"
    echo -e "${YELLOW}2${NC} - 📝 Submit Document (Simple & Multi-stage)"
    echo -e "${YELLOW}3${NC} - ✅ Approve Document"
    echo -e "${YELLOW}4${NC} - 🔄 Submit New Version"
    echo -e "${YELLOW}5${NC} - 📊 Query Document Status"
    echo ""
    echo -e "${RED}6${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# PHASE 6 NATIVE FUNCTIONS
# =============================================================================

initialize_ledger() {
    echo -e "${GREEN}=== 🆕 Initialize Ledger ===${NC}"
    echo -e "${BLUE}Setup blockchain with sample data${NC}"
    echo ""

    local confirm
    confirm=$(get_boolean "Proceed with initialization?")
    if [[ "$confirm" == "false" ]]; then
        echo -e "${YELLOW}Cancelled${NC}"
        return
    fi

    execute_chaincode_function "InitLedger"
}

submit_document() {
    echo -e "${GREEN}=== 📝 Submit Document (Phase 6 Native) ===${NC}"
    echo -e "${BLUE}🎯 Direct mapping to StageInput format${NC}"
    echo ""

    # Basic document info
    read -p "Document ID: " id
    read -p "Title: " title
    read -p "Description: " description
    read -p "Content hash: " hash
    read -p "Your identity (uploader): " uploader

    # Decision set
    echo ""
    local decisions
    decisions=$(get_decision_set)

    # Workflow approach selection
    echo ""
    echo -e "${GREEN}=== Workflow Design ===${NC}"
    echo -e "${YELLOW}1${NC} - Single-stage approval (simple)"
    echo -e "${YELLOW}2${NC} - Multi-stage workflow (advanced)"
    echo ""

    local workflow_choice
    read -p "Choose workflow type [1-2]: " workflow_choice

    local stages_json=""

    case $workflow_choice in
        1)
            # SIMPLE: Single stage
            echo -e "${BLUE}=== Single Stage Setup ===${NC}"
            read -p "Stage name (default: 'Document Review'): " stage_name
            stage_name="${stage_name:-Document Review}"
            read -p "Stage description (optional): " stage_desc

            echo ""
            local approvers
            approvers=$(get_approver_group)

            read -p "Required approvals (number, or press Enter for ALL): " required
            if [[ "$required" == "ALL" || "$required" == "" ]]; then
                required="0"
            fi

            local auto_advance
            auto_advance=$(get_boolean "Auto-advance when complete?")

            # Build single stage
            local stage
            stage=$(build_stage_input "$stage_name" "$stage_desc" "$approvers" "$required" "$auto_advance")
            stages_json="[$stage]"
            ;;

        2)
            # ADVANCED: Multi-stage
            echo -e "${BLUE}=== Multi-Stage Workflow Builder ===${NC}"
            local stage_array=()

            local stage_num=1
            while true; do
                echo -e "${CYAN}--- Stage $stage_num ---${NC}"
                read -p "Stage name: " stage_name
                read -p "Stage description (optional): " stage_desc

                echo ""
                local approvers
                approvers=$(get_approver_group)

                read -p "Required approvals (number, or press Enter for ALL): " required
                if [[ "$required" == "ALL" || "$required" == "" ]]; then
                    required="0"
                fi

                local auto_advance
                auto_advance=$(get_boolean "Auto-advance to next stage?")

                # Build this stage
                local stage
                stage=$(build_stage_input "$stage_name" "$stage_desc" "$approvers" "$required" "$auto_advance")
                stage_array+=("$stage")

                echo ""
                local add_more
                add_more=$(get_boolean "Add another stage?")
                if [[ "$add_more" == "false" ]]; then
                    break
                fi
                stage_num=$((stage_num + 1))
            done

            # Combine stages into JSON array
            stages_json="["
            for i in "${!stage_array[@]}"; do
                if [ $i -gt 0 ]; then
                    stages_json+=","
                fi
                stages_json+="${stage_array[$i]}"
            done
            stages_json+="]"
            ;;

        *)
            echo -e "${RED}Invalid choice. Using single-stage default.${NC}"
            local default_stage
            default_stage=$(build_stage_input "Document Review" "" "[\"reviewer\"]" "0" "true")
            stages_json="[$default_stage]"
            ;;
    esac

    # Final confirmation
    echo ""
    echo -e "${GREEN}=== 📋 Submit Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $id ($title)"
    echo -e "${BLUE}Uploader:${NC} $uploader"
    echo -e "${BLUE}Decisions:${NC} $decisions"
    echo -e "${BLUE}Stages:${NC} $(echo "$stages_json" | jq -r 'length') stage(s)"
    echo ""

    local confirm
    confirm=$(get_boolean "Submit document?")
    if [[ "$confirm" == "false" ]]; then
        echo -e "${YELLOW}Submission cancelled${NC}"
        return
    fi

    # Execute with Phase 6 signature: (id, title, description, hash, uploader, validDecisionsJSON, stagesJSON)
    execute_chaincode_function "SubmitDocument" "$id" "$title" "$description" "$hash" "$uploader" "$decisions" "$stages_json"
}

approve_document() {
    echo -e "${GREEN}=== ✅ Approve Document ===${NC}"
    echo -e "${BLUE}Unified approval for all document types${NC}"
    echo ""

    read -p "Document ID: " id
    read -p "Your identity (approver): " approver
    read -p "Decision (APPROVED/REJECTED/etc): " decision
    read -p "Comment (optional): " comment

    echo ""
    echo -e "${GREEN}=== 📋 Approval Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $id"
    echo -e "${BLUE}Approver:${NC} $approver"
    echo -e "${BLUE}Decision:${NC} $decision"
    echo -e "${BLUE}Comment:${NC} $comment"
    echo ""

    local confirm
    confirm=$(get_boolean "Submit approval?")
    if [[ "$confirm" == "false" ]]; then
        echo -e "${YELLOW}Approval cancelled${NC}"
        return
    fi

    # Execute with Phase 6 signature: (id, approver, decision, comment)
    execute_chaincode_function "ApproveDocument" "$id" "$approver" "$decision" "$comment"
}

submit_new_version() {
    echo -e "${GREEN}=== 🔄 Submit New Version ===${NC}"
    echo -e "${BLUE}Enhanced versioning with NewVersionState${NC}"
    echo ""

    read -p "Document ID: " doc_id
    read -p "New content hash: " new_hash
    read -p "Change reason: " change_reason

    echo ""
    echo -e "${BLUE}=== Version Updates ===${NC}"

    local update_approvers
    update_approvers=$(get_boolean "Update approvers list?")

    local approvers_list="[]"
    if [[ "$update_approvers" == "true" ]]; then
        approvers_list=$(get_array "New approvers")
    fi

    local update_decisions
    update_decisions=$(get_boolean "Update valid decisions?")

    local decisions_list="[]"
    if [[ "$update_decisions" == "true" ]]; then
        decisions_list=$(get_decision_set)
    fi

    local reset_approvals
    reset_approvals=$(get_boolean "Reset all approvals?")

    # Build NewVersionState JSON
    local new_state
    new_state=$(build_new_version_state "$new_hash" "$approvers_list" "$decisions_list" "$reset_approvals" "$change_reason")

    echo ""
    echo -e "${GREEN}=== 📋 Version Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $doc_id"
    echo -e "${BLUE}New Hash:${NC} $new_hash"
    echo -e "${BLUE}Reason:${NC} $change_reason"
    echo -e "${BLUE}Reset Approvals:${NC} $reset_approvals"
    echo ""

    local confirm
    confirm=$(get_boolean "Submit new version?")
    if [[ "$confirm" == "false" ]]; then
        echo -e "${YELLOW}Version submission cancelled${NC}"
        return
    fi

    # Execute with Phase 6 signature: (documentID, newStateJSON)
    execute_chaincode_function "SubmitNewVersion" "$doc_id" "$new_state"
}

query_document_status() {
    echo -e "${BLUE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}Get complete document information${NC}"
    echo ""

    read -p "Document ID: " id

    # Execute with Phase 6 signature: (id)
    execute_chaincode_function "QueryDocumentStatus" "$id"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_lifecycle_menu
    read -p "Enter your choice [1-6]: " choice

    case $choice in
        1) initialize_ledger ;;
        2) submit_document ;;
        3) approve_document ;;
        4) submit_new_version ;;
        5) query_document_status ;;
        6) return_to_main_menu; break ;;
        *) echo -e "${RED}Invalid choice. Please select 1-6.${NC}"; echo "";;
    esac
done