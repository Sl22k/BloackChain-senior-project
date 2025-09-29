#!/bin/bash

# =============================================================================
# DOCUMENT LIFECYCLE MODULE
# =============================================================================
# Core document operations: Submit, Approve, Versioning
# Unified workflow system from Phase 1-2 refactoring
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# DOCUMENT LIFECYCLE MENU
# =============================================================================

display_lifecycle_menu() {
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}   📋 DOCUMENT LIFECYCLE MANAGEMENT      ${NC}"
    echo -e "${GREEN}   Unified Workflow System               ${NC}"
    echo -e "${GREEN}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select a document lifecycle function:${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - 🆕 Initialize Ledger (Setup system)"
    echo -e "${YELLOW}2${NC} - 📝 Submit Document (Progressive workflow builder)"
    echo -e "${YELLOW}3${NC} - ✅ Approve Document (Unified approval system)"
    echo -e "${YELLOW}4${NC} - 🔄 Submit New Version (Enhanced versioning)"
    echo ""
    echo -e "${BLUE}5${NC} - 📊 Query Document Status"
    echo -e "${BLUE}6${NC} - 🔍 Get All Documents"
    echo ""
    echo -e "${RED}7${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# DOCUMENT LIFECYCLE FUNCTIONS
# =============================================================================

initialize_ledger() {
    echo -e "${GREEN}=== 🆕 Initialize Ledger ===${NC}"
    echo -e "${YELLOW}Setup the blockchain ledger with initial data${NC}"
    echo ""

    echo -e "${BLUE}This will initialize the ledger with sample data.${NC}"
    while true; do
        read -p "Proceed with ledger initialization? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Initialization cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "InitLedger"
}

submit_document() {
    echo -e "${GREEN}=== 📝 Submit New Document (Progressive Workflow Builder) ===${NC}"
    echo -e "${YELLOW}🌟 Unified submission with progressive disclosure${NC}"
    echo -e "${YELLOW}Start simple, add complexity as needed${NC}"
    echo ""

    # Document basic info
    read -p "Enter document ID: " id
    read -p "Enter document title: " title
    read -p "Enter document description: " description
    read -p "Enter document hash: " hash
    read -p "Enter uploader identity: " uploader

    # Decision set selection
    echo ""
    decisions=$(collect_decision_set)

    # Progressive workflow builder - start with stage 1
    echo ""
    echo -e "${GREEN}=== Approval Workflow (Progressive Builder) ===${NC}"
    echo -e "${YELLOW}Stage 1 (required) - Document Review${NC}"

    # First stage - simplified collection
    echo ""
    echo -e "${BLUE}Select approvers for this document:${NC}"
    stage1_approvers_json=$(collect_approver_group)

    echo ""
    read -p "Required approval count for Stage 1 (default: ALL): " stage1_required
    if [[ "$stage1_required" == "" || "$stage1_required" == "ALL" ]]; then
        # Count approvers for ALL logic
        approver_count=$(echo "$stage1_approvers_json" | grep -o '"[^"]*"' | wc -l)
        stage1_required="$approver_count"
    fi

    # Check if user wants to add more stages (progressive disclosure)
    echo ""
    echo -e "${PURPLE}💡 Progressive Disclosure: Add More Stages?${NC}"
    echo -e "${YELLOW}• Current: Single-stage workflow (simple)${NC}"
    echo -e "${YELLOW}• Option: Add more stages for complex approval process${NC}"
    echo ""

    read -p "Add additional stages? (y/n): " add_more_stages

    if [[ "$add_more_stages" == "y" || "$add_more_stages" == "Y" ]]; then
        # ADVANCED MODE: Build multi-stage workflow
        echo ""
        echo -e "${GREEN}=== Multi-Stage Workflow Builder ===${NC}"
        echo -e "${YELLOW}Building advanced workflow...${NC}"


        # First stage name input
        echo -e "${BLUE}--- Stage 1 ---${NC}"
        read -p "Enter name for Stage 1: " stage1_name
        if [[ "$stage1_name" == "" ]]; then
            stage1_name="Review and Approval"
        fi
        read -p "Enter description for Stage 1 (optional): " stage1_description
        read -p "Set deadline for Stage 1? (hours, empty for none): " stage1_deadline_hours
        if [[ "$stage1_deadline_hours" == "" || ! "$stage1_deadline_hours" =~ ^[0-9]+$ ]]; then
            stage1_deadline_hours=0
        fi

        # Extract approvers from Stage 1 for raw JSON building
        stage1_approvers_list=$(echo "$stage1_approvers_json" | sed 's/^\[//' | sed 's/\]$//')

        # Build workflow stages JSON without escaping (for raw JSON)
        workflow_stages=""
        escaped_stage1_name=$(echo "$stage1_name" | sed 's/"/\\"/g')
        escaped_stage1_description=$(echo "$stage1_description" | sed 's/"/\\"/g')

        stage1_json="{\"name\":\"$escaped_stage1_name\",\"description\":\"$escaped_stage1_description\",\"approvers\":[$stage1_approvers_list],\"requiredApprovals\":$stage1_required,\"autoAdvance\":true,\"deadlineHours\":$stage1_deadline_hours}"
        workflow_stages="$stage1_json"

        stage_count=1

        # Arrays to store stage details for preview
        stage_names=("$stage1_name")
        stage_descriptions=("$stage1_description")
        stage_approvers_jsons=("$stage1_approvers_json")
        stage_required_counts=("$stage1_required")
        stage_deadline_hours_array=("$stage1_deadline_hours")
        stage_auto_advances=("true")

        # Add additional stages
        while true; do
            stage_count=$((stage_count + 1))
            echo -e "${BLUE}--- Stage $stage_count ---${NC}"
            read -p "Enter stage name: " stageName
            read -p "Enter description for Stage $stage_count (optional): " stageDescription

            # Use enhanced approver group selection
            echo ""
            echo -e "${BLUE}Select approvers for stage $stage_count:${NC}"
            stage_approvers_json=$(collect_approver_group)
            stage_approvers_list=$(echo "$stage_approvers_json" | sed 's/^\[//' | sed 's/\]$//')

            echo ""
            read -p "Required approval count (default: ALL): " requiredCount
            if [[ "$requiredCount" == "" || "$requiredCount" == "ALL" ]]; then
                # Count approvers for ALL logic
                approver_count=$(echo "$stage_approvers_json" | grep -o '"[^"]*"' | wc -l)
                requiredCount="$approver_count"
            fi

            autoAdvance=$(collect_boolean_input "Auto-advance to next stage?")

            # Get deadline for this stage
            read -p "Set deadline for Stage $stage_count? (hours, empty for none): " stage_deadline_hours
            if [[ "$stage_deadline_hours" == "" || ! "$stage_deadline_hours" =~ ^[0-9]+$ ]]; then
                stage_deadline_hours=0
            fi

            # Build stage JSON without escaping (for raw JSON)
            escaped_stageName=$(echo "$stageName" | sed 's/"/\\"/g')
            escaped_stageDescription=$(echo "$stageDescription" | sed 's/"/\\"/g')
            stage_json="{\"name\":\"$escaped_stageName\",\"description\":\"$escaped_stageDescription\",\"approvers\":[$stage_approvers_list],\"requiredApprovals\":$requiredCount,\"autoAdvance\":$autoAdvance,\"deadlineHours\":$stage_deadline_hours}"

            workflow_stages="$workflow_stages,$stage_json"

            # Store stage details for preview
            stage_names+=("$stageName")
            stage_descriptions+=("$stageDescription")
            stage_approvers_jsons+=("$stage_approvers_json")
            stage_required_counts+=("$requiredCount")
            stage_deadline_hours_array+=("$stage_deadline_hours")
            stage_auto_advances+=("$autoAdvance")

            read -p "Add another stage? (y/n): " add_another
            if [[ "$add_another" != "y" && "$add_another" != "Y" ]]; then
                break
            fi
        done

        workflow_stages="[$workflow_stages]"

        # Advanced configuration preview
        echo ""
        echo -e "${GREEN}=== Multi-Stage Configuration Preview ===${NC}"
        echo -e "${BLUE}Document:${NC} $id ($title)"
        echo -e "${BLUE}Total Stages:${NC} $stage_count stages"
        echo -e "${BLUE}Workflow Type:${NC} Advanced multi-stage workflow"
        echo ""


        # Advanced mode arguments: Phase 6 simplified format
        args=("$id" "$title" "$description" "$hash" "$uploader" "$decisions" "$workflow_stages")

    else
        # SIMPLE MODE: Single-stage workflow
        echo -e "${GREEN}=== Simple Workflow Configuration ===${NC}"
        echo -e "${BLUE}Single-stage approval process${NC}"

        # Ask for stage name (optional)
        echo ""
        read -p "Enter stage name (default: 'Document Approval'): " stage1_name
        if [[ "$stage1_name" == "" ]]; then
            stage1_name="Document Approval"
        fi

        # Optional deadline configuration
        echo ""
        echo -e "${BLUE}=== Optional Deadline Configuration ===${NC}"
        read -p "Set deadline? Hours from now (empty for no deadline): " deadlineHours
        if [[ "$deadlineHours" == "" ]]; then
            deadlineHours=""
        fi

        # Build simple workflow JSON with custom stage name and deadline
        stage1_approvers_list=$(echo "$stage1_approvers_json" | sed 's/^\[//' | sed 's/\]$//')
        escaped_stage1_name=$(echo "$stage1_name" | sed 's/"/\\\"/g')

        # Convert deadlineHours to integer (0 if empty)
        deadline_hours_int=0
        if [[ "$deadlineHours" != "" && "$deadlineHours" =~ ^[0-9]+$ ]]; then
            deadline_hours_int="$deadlineHours"
        fi

        simple_workflow="[{\"name\":\"$escaped_stage1_name\",\"description\":\"\",\"approvers\":[$stage1_approvers_list],\"requiredApprovals\":$stage1_required,\"autoAdvance\":true,\"deadlineHours\":$deadline_hours_int}]"

        # Simple mode arguments: Phase 6 simplified format
        args=("$id" "$title" "$description" "$hash" "$uploader" "$decisions" "$simple_workflow")
    fi

    # Enhanced Final confirmation with detailed preview
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    📋 SUBMISSION PREVIEW                       ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}📄 Document Information:${NC}"
    echo -e "${BLUE}   • Document ID:${NC} $id"
    echo -e "${BLUE}   • Title:${NC} $title"
    echo -e "${BLUE}   • Description:${NC} $description"
    echo -e "${BLUE}   • Content Hash:${NC} $hash"
    echo -e "${BLUE}   • Uploader:${NC} $uploader"
    echo ""
    echo -e "${CYAN}⚖️  Decision Framework:${NC}"
    echo -e "${BLUE}   • Valid Decisions:${NC} $(echo "$decisions" | tr -d '[]\"' | sed 's/,/, /g')"
    echo ""
    if [[ "$add_more_stages" == "y" || "$add_more_stages" == "Y" ]]; then
        echo -e "${CYAN}🔄 Workflow Configuration:${NC} ${YELLOW}Multi-Stage Process${NC}"
        echo -e "${BLUE}   • Total Stages:${NC} $stage_count"
        echo -e "${BLUE}   • Sequential Processing:${NC} Each stage must complete before next"
        echo ""
        echo -e "${CYAN}📋 Stage Details:${NC}"

        # Display ALL stages with complete details
        for i in "${!stage_names[@]}"; do
            stage_num=$((i + 1))
            stage_name="${stage_names[$i]}"
            stage_approvers_json="${stage_approvers_jsons[$i]}"
            stage_required="${stage_required_counts[$i]}"
            stage_deadline="${stage_deadline_hours_array[$i]}"
            stage_auto_advance="${stage_auto_advances[$i]}"

            # Format approvers
            stage_approvers_clean=$(echo "$stage_approvers_json" | tr -d '[]\"' | sed 's/,/, /g')
            stage_approvers_count=$(echo "$stage_approvers_json" | grep -o '"[^"]*"' | wc -l)

            # Determine connector
            if [[ $i -eq 0 ]]; then
                if [[ $stage_count -eq 1 ]]; then
                    connector="└─"
                else
                    connector="┌─"
                fi
            elif [[ $i -eq $((stage_count - 1)) ]]; then
                connector="└─"
            else
                connector="├─"
            fi

            # Display stage information
            echo -e "${BLUE}   $connector Stage $stage_num: \"$stage_name\"${NC}"
            echo -e "${BLUE}   │  • Approvers:${NC} $stage_approvers_clean ($stage_approvers_count people)"

            # Show required approvals with context
            if [[ "$stage_required" == "$stage_approvers_count" ]]; then
                echo -e "${BLUE}   │  • Required:${NC} $stage_required out of $stage_approvers_count approvals (ALL)"
            else
                echo -e "${BLUE}   │  • Required:${NC} $stage_required out of $stage_approvers_count approvals"
            fi

            # Show deadline information
            if [[ "$stage_deadline" != "0" && "$stage_deadline" != "" ]]; then
                echo -e "${BLUE}   │  • Deadline:${NC} $stage_deadline hours from submission"
            else
                echo -e "${BLUE}   │  • Deadline:${NC} No time limit"
            fi

            # Show auto-advance
            if [[ "$stage_auto_advance" == "true" ]]; then
                echo -e "${BLUE}   │  • Auto-advance:${NC} ✅ Enabled"
            else
                echo -e "${BLUE}   │  • Auto-advance:${NC} ❌ Disabled"
            fi

            # Add spacing except for last stage
            if [[ $i -lt $((stage_count - 1)) ]]; then
                echo -e "${BLUE}   │${NC}"
            fi
        done
    else
        echo -e "${CYAN}🔄 Workflow Configuration:${NC} ${YELLOW}Single-Stage Process${NC}"
        echo -e "${BLUE}   • Simple Approval:${NC} All approvals happen in one stage"
        echo ""
        echo -e "${CYAN}📋 Stage Details:${NC}"

        # Show detailed single-stage information
        approvers_clean=$(echo "$stage1_approvers_json" | tr -d '[]\"' | sed 's/,/, /g')
        approvers_count=$(echo "$stage1_approvers_json" | grep -o '"[^"]*"' | wc -l)

        echo -e "${BLUE}   └─ Stage 1: \"$stage1_name\"${NC}"
        echo -e "${BLUE}      • Approvers:${NC} $approvers_clean ($approvers_count people)"

        # Show required approvals with context
        if [[ "$stage1_required" == "$approvers_count" ]]; then
            echo -e "${BLUE}      • Required:${NC} $stage1_required out of $approvers_count approvals (ALL)"
        else
            echo -e "${BLUE}      • Required:${NC} $stage1_required out of $approvers_count approvals"
        fi

        # Show deadline information
        if [[ "$deadlineHours" != "" && "$deadlineHours" != "0" ]]; then
            echo -e "${BLUE}      • Deadline:${NC} $deadlineHours hours from submission"
        else
            echo -e "${BLUE}      • Deadline:${NC} No time limit"
        fi

        echo -e "${BLUE}      • Auto-advance:${NC} ✅ Enabled (document completes when approved)"
    fi
    echo ""
    echo -e "${PURPLE}🚀 Ready to submit this document to the blockchain?${NC}"
    echo ""

    while true; do
        read -p "Proceed with this configuration? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Configuration cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "SubmitDocument" "${args[@]}"
}

approve_document() {
    echo -e "${GREEN}=== ✅ Unified Document Approval ===${NC}"
    echo -e "${YELLOW}🌟 Automatically detects document type (simple or workflow)${NC}"
    echo -e "${YELLOW}Works with both single-stage and multi-stage documents${NC}"
    echo ""

    read -p "Enter document ID: " id
    read -p "Enter approver identity: " approver
    read -p "Enter decision: " decision
    read -p "Enter comment (optional): " comment

    echo ""
    echo -e "${GREEN}=== Approval Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $id"
    echo -e "${BLUE}Approver:${NC} $approver"
    echo -e "${BLUE}Decision:${NC} $decision"
    echo -e "${BLUE}Comment:${NC} $comment"
    echo ""

    while true; do
        read -p "Proceed with this approval? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Approval cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "ApproveDocument" "$id" "$approver" "$decision" "$comment"
}

submit_new_version() {
    echo -e "${GREEN}=== 🔄 Submit New Document Version (Unified Model) ===${NC}"
    echo -e "${YELLOW}Stage-specific approver updates with unified workflow model${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter submitter/invoker ID: " submitter
    read -p "Enter new content hash: " contentHash
    read -p "Enter change reason (required): " changeReason

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
    read -p "Which stages to update (comma-separated, e.g., 1,3 or 'none'): " stages_to_update

    # Build stageUpdates JSON
    stage_updates="{}"
    if [[ "$stages_to_update" != "none" && -n "$stages_to_update" ]]; then
        for stage_num in $(echo $stages_to_update | tr ',' ' '); do
            echo ""
            echo -e "${GREEN}=== Stage $stage_num Updates ===${NC}"

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

    # UX-friendly valid decisions selection
    echo ""
    echo -e "${GREEN}=== Select Valid Decisions for New Version ===${NC}"
    echo -e "${YELLOW}Current valid decisions:${NC} $(echo "$current_doc_result" | jq -r '.ValidDecisions | join(", ")')"
    echo ""
    echo -e "${YELLOW}1${NC} - Keep current decisions unchanged"
    echo -e "${YELLOW}2${NC} - Standard (APPROVED, REJECTED)"
    echo -e "${YELLOW}3${NC} - Extended (APPROVED, REJECTED, NEEDS_REVISION)"
    echo -e "${YELLOW}4${NC} - Comprehensive (APPROVED, REJECTED, NEEDS_REVISION, ON_HOLD)"
    echo -e "${YELLOW}5${NC} - Custom (enter your own)"
    echo ""

    while true; do
        read -p "Choice [1-5]: " decision_choice
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
                read -p "Enter valid decisions (comma-separated): " custom_decisions
                if [[ -n "$custom_decisions" ]]; then
                    validDecisions="[\"$(echo "$custom_decisions" | tr -d ' ' | sed 's/,/","/g')\"]"
                    break
                else
                    echo -e "${RED}Please enter at least one valid decision${NC}"
                fi
                ;;
            *) echo -e "${RED}Invalid choice 1-5${NC}";;
        esac
    done

    if [[ -z "$validDecisions" || "$validDecisions" == "null" ]]; then
        validDecisions='["APPROVED","REJECTED"]'
    fi

    echo -e "${GREEN}Selected decisions:${NC} $(echo "$validDecisions" | tr -d '[]"' | sed 's/,/, /g')"

    # Reset Approvals Selection
    echo ""
    echo -e "${GREEN}=== Reset Approvals Setting ===${NC}"
    echo -e "${YELLOW}Current approvals will be:${NC}"
    echo ""
    echo -e "${YELLOW}1${NC} - Reset to PENDING (recommended for most changes)"
    echo -e "${YELLOW}2${NC} - Keep existing approvals (only for stage 1 documents)"
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
                echo -e "${YELLOW}Note:${NC} This only works for documents in stage 1"
                break;;
            *) echo -e "${RED}Invalid choice. Please select 1 or 2${NC}";;
        esac
    done

    # Build final JSON
    newVersionState=$(jq -n \
        --arg contentHash "$contentHash" \
        --argjson stageUpdates "$stage_updates" \
        --argjson validDecisions "$validDecisions" \
        --argjson resetApprovals "$resetApprovals" \
        --arg changeReason "$changeReason" \
        '{
            "contentHash": $contentHash,
            "stageUpdates": $stageUpdates,
            "validDecisions": $validDecisions,
            "resetApprovals": $resetApprovals,
            "changeReason": $changeReason
        }')

    echo ""
    echo -e "${GREEN}=== Version Update Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $documentID"
    echo -e "${BLUE}Submitter:${NC} $submitter"
    echo -e "${BLUE}Content Hash:${NC} $contentHash"
    echo -e "${BLUE}Change Reason:${NC} $changeReason"
    echo ""
    echo -e "${PURPLE}Generated NewVersionState JSON:${NC}"
    echo "$newVersionState" | jq .
    echo ""

    while true; do
        read -p "Proceed with this version update? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Version update cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    # Execute with correct signature
    compactVersionState=$(echo "$newVersionState" | jq -c .)
    execute_chaincode_function "SubmitNewVersion" "$documentID" "$submitter" "$compactVersionState"
}

query_document_status() {
    echo -e "${BLUE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}Get complete document status and workflow information${NC}"
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

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_lifecycle_menu
    read -p "Enter your choice [1-7]: " choice

    case $choice in
        1)
            initialize_ledger
            ;;
        2)
            submit_document
            ;;
        3)
            approve_document
            ;;
        4)
            submit_new_version
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