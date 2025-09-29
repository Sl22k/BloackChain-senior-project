#!/bin/bash

# =============================================================================
# WORKFLOW MANAGEMENT MODULE
# =============================================================================
# Advanced workflow operations: Configure, Status, Stage Management
# Multi-stage workflow configuration and control
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# WORKFLOW MANAGEMENT MENU
# =============================================================================

display_workflow_menu() {
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}   🔄 WORKFLOW MANAGEMENT SYSTEM          ${NC}"
    echo -e "${GREEN}   Multi-Stage Approval Workflows         ${NC}"
    echo -e "${GREEN}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select a workflow management function:${NC}"
    echo ""
    echo -e "${YELLOW}🔧 Workflow Configuration:${NC}"
    echo -e "${YELLOW}1${NC} - ⚙️  Configure Document Workflow (Setup multi-stage workflows)"
    echo -e "${YELLOW}2${NC} - 📊 Get Workflow Status (Current workflow state details)"
    echo ""
    echo -e "${CYAN}🎯 Stage Management:${NC}"
    echo -e "${CYAN}3${NC} - ⬆️  Advance Document to Next Stage (Manual stage advancement)"
    echo -e "${CYAN}4${NC} - ⬅️  Return Document to Stage (Rollback to previous stage)"
    echo ""
    echo -e "${BLUE}📈 Workflow History & Analysis:${NC}"
    echo -e "${BLUE}5${NC} - 📈 Get Document Workflow History (Complete workflow history)"
    echo -e "${BLUE}6${NC} - 🎯 Get Version Workflow State (Version-specific workflow)"
    echo ""
    echo -e "${PURPLE}📊 Quick Status Queries:${NC}"
    echo -e "${PURPLE}7${NC} - 📊 Query Document Status (General status including workflow)"
    echo -e "${PURPLE}8${NC} - 🔍 Get All Documents (List documents for workflow management)"
    echo ""
    echo -e "${RED}9${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# WORKFLOW CONFIGURATION FUNCTIONS
# =============================================================================

configure_document_workflow() {
    echo -e "${GREEN}=== ⚙️  Enhanced Workflow Configuration ===${NC}"
    echo -e "${BLUE}🎯 Complete workflow reconfiguration with maximum flexibility${NC}"
    echo -e "${YELLOW}Add, remove, modify stages with intelligent approval preservation${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    # Load and display current workflow configuration
    echo ""
    echo -e "${BLUE}🔍 Loading current workflow configuration...${NC}"

    local current_doc_result
    current_doc_result=$(execute_chaincode_query "QueryDocumentStatus" "$documentID" 2>/dev/null)

    if [ $? -ne 0 ] || [ -z "$current_doc_result" ]; then
        echo -e "${RED}❌ Failed to load document $documentID${NC}"
        echo -e "${YELLOW}Please verify the document ID and try again.${NC}"
        return
    fi

    # Check authorization by examining uploader and privileged editors
    local uploader
    uploader=$(echo "$current_doc_result" | jq -r '.Uploader // "unknown"')
    local privileged_editors
    privileged_editors=$(echo "$current_doc_result" | jq -r '.PrivilegedEditors[]? // empty' | tr '\n' ' ')

    if [[ "$uploader" != "$invokerId" ]] && [[ ! " $privileged_editors " =~ " $invokerId " ]]; then
        echo -e "${RED}❌ Authorization Failed${NC}"
        echo -e "${YELLOW}Only the document uploader ($uploader) or privileged editors can configure workflows.${NC}"
        echo -e "${YELLOW}Privileged editors: ${privileged_editors:-"none"}${NC}"
        return
    fi

    # Display comprehensive current workflow preview
    echo -e "${GREEN}=== 📋 CURRENT WORKFLOW CONFIGURATION ===${NC}"
    echo -e "${BLUE}📄 Document:${NC} $documentID ($(echo "$current_doc_result" | jq -r '.Title // "Untitled"'))"
    echo -e "${BLUE}👤 Uploader:${NC} $uploader"
    echo -e "${BLUE}🎯 Current Stage:${NC} $(echo "$current_doc_result" | jq -r '.Workflow.CurrentStage // 0')"
    echo -e "${BLUE}✅ Completed Stages:${NC} $(echo "$current_doc_result" | jq -r '.Workflow.CompletedStages | length') stage(s)"

    # Extract and display current stages
    local stage_count
    stage_count=$(echo "$current_doc_result" | jq '.Workflow.Stages | length')
    echo -e "${BLUE}📊 Total Stages:${NC} $stage_count"
    echo ""

    if [[ "$stage_count" -eq 0 ]]; then
        echo -e "${RED}❌ No workflow configured for this document${NC}"
        echo -e "${YELLOW}Use SubmitDocument to create initial workflow first.${NC}"
        return
    fi

    # Display each stage in detail
    echo -e "${CYAN}=== CURRENT STAGES DETAILED VIEW ===${NC}"
    for ((i=0; i<stage_count; i++)); do
        local stage_num=$((i+1))
        local stage_name=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$i].StageName // \"Stage $stage_num\"")
        local stage_desc=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$i].Description // \"\"")
        local stage_approvers=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$i].Approvers | join(\", \")")
        local required_count=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$i].RequiredCount // 0")
        local auto_advance=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$i].AutoAdvance // false")

        # Check for stage deadline
        local stage_deadline=""
        local deadline_info=$(echo "$current_doc_result" | jq -r ".DeadlineConfig.StageDeadlines.\"stage_$stage_num\" // \"\"")
        if [[ -n "$deadline_info" && "$deadline_info" != "null" ]]; then
            stage_deadline=" 🕒 Deadline: $deadline_info"
        fi

        echo -e "${YELLOW}📍 Stage $stage_num:${NC} $stage_name"
        if [[ -n "$stage_desc" ]]; then
            echo -e "   ${BLUE}📝 Description:${NC} $stage_desc"
        fi
        echo -e "   ${BLUE}👥 Approvers:${NC} $stage_approvers"
        echo -e "   ${BLUE}✅ Required:${NC} $([ "$required_count" -eq 0 ] && echo "ALL" || echo "$required_count")"
        echo -e "   ${BLUE}⚡ Auto-advance:${NC} $auto_advance$stage_deadline"
        echo ""
    done

    # Ask for configuration reason
    echo -e "${GREEN}=== 📝 Configuration Details ===${NC}"
    read -p "Enter reason for workflow reconfiguration: " config_reason
    if [[ -z "$config_reason" ]]; then
        echo -e "${RED}❌ Configuration reason is required${NC}"
        return
    fi

    # Initialize configuration request structure
    local stage_operations="{}"
    local new_stages="[]"
    local stages_to_remove="[]"
    local deadline_updates="{}"
    local keep_existing_approvals="true"

    # Stage modification menu
    echo ""
    echo -e "${GREEN}=== 🔧 STAGE MODIFICATION OPTIONS ===${NC}"
    echo -e "${YELLOW}Choose what you want to do with the current workflow:${NC}"
    echo ""
    echo -e "${BLUE}1${NC} - 📝 Modify existing stages (change approvers, settings)"
    echo -e "${BLUE}2${NC} - ➕ Add new stages"
    echo -e "${BLUE}3${NC} - ➖ Remove stages"
    echo -e "${BLUE}4${NC} - 🕒 Update deadlines"
    echo -e "${BLUE}5${NC} - 🔄 Complete workflow reconfiguration (all options)"
    echo -e "${BLUE}6${NC} - ❌ Cancel configuration"
    echo ""

    while true; do
        read -p "Enter your choice [1-6]: " modification_choice
        case $modification_choice in
            1)
                echo -e "${BLUE}=== 📝 Modify Existing Stages ===${NC}"
                stage_operations=$(configure_stage_modifications "$current_doc_result" "$stage_count")
                break
                ;;
            2)
                echo -e "${BLUE}=== ➕ Add New Stages ===${NC}"
                new_stages=$(configure_new_stages)
                break
                ;;
            3)
                echo -e "${BLUE}=== ➖ Remove Stages ===${NC}"
                stages_to_remove=$(configure_stage_removals "$stage_count")
                break
                ;;
            4)
                echo -e "${BLUE}=== 🕒 Update Deadlines ===${NC}"
                deadline_updates=$(configure_deadline_updates "$stage_count" "$deadline_status")
                break
                ;;
            5)
                echo -e "${BLUE}=== 🔄 Complete Workflow Reconfiguration ===${NC}"
                echo -e "${YELLOW}This allows you to modify, add, remove stages and update deadlines in one operation${NC}"

                # Stage modifications
                echo ""
                echo -e "${CYAN}--- Step 1: Modify Existing Stages ---${NC}"
                local modify_existing
                modify_existing=$(collect_boolean_input "Do you want to modify any existing stages?")
                if [[ "$modify_existing" == "true" ]]; then
                    stage_operations=$(configure_stage_modifications "$current_doc_result" "$stage_count")
                fi

                # Add new stages
                echo ""
                echo -e "${CYAN}--- Step 2: Add New Stages ---${NC}"
                local add_new
                add_new=$(collect_boolean_input "Do you want to add new stages?")
                if [[ "$add_new" == "true" ]]; then
                    new_stages=$(configure_new_stages)
                fi

                # Remove stages
                echo ""
                echo -e "${CYAN}--- Step 3: Remove Stages ---${NC}"
                local remove_stages
                remove_stages=$(collect_boolean_input "Do you want to remove any stages?")
                if [[ "$remove_stages" == "true" ]]; then
                    stages_to_remove=$(configure_stage_removals "$stage_count")
                fi

                # Update deadlines
                echo ""
                echo -e "${CYAN}--- Step 4: Configure Deadlines ---${NC}"
                local update_deadlines
                update_deadlines=$(collect_boolean_input "Do you want to update stage deadlines?")
                if [[ "$update_deadlines" == "true" ]]; then
                    deadline_updates=$(configure_deadline_updates "$stage_count" "$deadline_status")
                fi
                break
                ;;
            6)
                echo -e "${YELLOW}Configuration cancelled. Returning to menu...${NC}"
                return
                ;;
            *)
                echo -e "${RED}Invalid choice. Please select 1-6.${NC}"
                ;;
        esac
    done

    # Approval preservation choice
    echo ""
    echo -e "${GREEN}=== 🔄 Approval Preservation ===${NC}"
    echo -e "${YELLOW}Do you want to keep existing approvals where possible?${NC}"
    echo -e "${BLUE}• Yes: Preserve approvals for unchanged stages${NC}"
    echo -e "${BLUE}• No: Reset all approvals to pending${NC}"
    keep_existing_approvals=$(collect_boolean_input "Keep existing approvals where possible?")

    # Build configuration JSON (single line to avoid newline issues)
    local configuration_json="{\"stageOperations\":$stage_operations,\"newStages\":$new_stages,\"stagesToRemove\":$stages_to_remove,\"keepExistingApprovals\":$keep_existing_approvals,\"configurationReason\":\"$config_reason\",\"deadlineUpdates\":$deadline_updates}"

    # Final configuration preview
    echo ""
    echo -e "${GREEN}=== 📋 FINAL CONFIGURATION PREVIEW ===${NC}"
    echo -e "${BLUE}📄 Document:${NC} $documentID"
    echo -e "${BLUE}👤 Configured by:${NC} $invokerId"
    echo -e "${BLUE}📝 Reason:${NC} $config_reason"
    echo -e "${BLUE}🔄 Keep approvals:${NC} $keep_existing_approvals"
    echo ""
    echo -e "${YELLOW}Configuration summary:${NC}"

    # Show operation counts
    local mod_count=$(echo "$stage_operations" | jq 'length // 0')
    local new_count=$(echo "$new_stages" | jq 'length // 0')
    local remove_count=$(echo "$stages_to_remove" | jq 'length // 0')
    local deadline_count=$(echo "$deadline_updates" | jq 'length // 0')

    echo -e "${BLUE}• Modified stages:${NC} $mod_count"
    echo -e "${BLUE}• New stages:${NC} $new_count"
    echo -e "${BLUE}• Removed stages:${NC} $remove_count"
    echo -e "${BLUE}• Deadline updates:${NC} $deadline_count"
    echo ""

    while true; do
        read -p "Proceed with workflow reconfiguration? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Configuration cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    # Execute the configuration
    echo -e "${GREEN}🔄 Executing workflow reconfiguration...${NC}"
    execute_chaincode_function "ConfigureDocumentWorkflow" "$documentID" "$invokerId" "$configuration_json"
}

get_workflow_status() {
    echo -e "${YELLOW}=== 📊 Get Workflow Status ===${NC}"
    echo -e "${BLUE}Detailed workflow state and stage information${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "GetWorkflowStatus" "$documentID"
}

# =============================================================================
# STAGE MANAGEMENT FUNCTIONS
# =============================================================================

advance_document_to_next_stage() {
    echo -e "${CYAN}=== ⬆️  Advance Document to Next Stage ===${NC}"
    echo -e "${BLUE}Manually advance document to the next workflow stage${NC}"
    echo -e "${YELLOW}Use this when AutoAdvance is disabled or for manual control${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    echo ""
    echo -e "${GREEN}=== Stage Advancement Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo -e "${BLUE}Action:${NC} Advance to next stage"
    echo ""

    while true; do
        read -p "Proceed with stage advancement? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Stage advancement cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    execute_chaincode_function "AdvanceDocumentToNextStage" "$documentID" "$invokerId"
}

return_document_to_stage() {
    echo -e "${CYAN}=== ⬅️  Enhanced Return Document to Stage ===${NC}"
    echo -e "${BLUE}Return document to previous workflow stage with deadline handling${NC}"
    echo ""

    # 1. Get basic input
    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    # 2. Load and display current document status
    echo ""
    echo -e "${GREEN}🔍 Loading document workflow status...${NC}"

    local doc_status
    doc_status=$(execute_chaincode_query "QueryDocumentStatus" "$documentID" 2>/dev/null)

    if [ $? -ne 0 ] || [ -z "$doc_status" ]; then
        echo -e "${RED}❌ Failed to load document $documentID${NC}"
        echo -e "${YELLOW}Please verify the document ID and try again.${NC}"
        return
    fi

    echo -e "${CYAN}=== CURRENT DOCUMENT WORKFLOW STATUS ===${NC}"
    echo -e "${BLUE}📄 Document ID:${NC} $documentID"
    echo ""


    # Parse and display document workflow information
    local current_stage=$(echo "$doc_status" | jq -r '.Workflow.CurrentStage // "N/A"')
    local title=$(echo "$doc_status" | jq -r '.Title // "N/A"')
    local status=$(echo "$doc_status" | jq -r '.WorkflowProgress.StatusSummary // "N/A"')
    local total_stages=$(echo "$doc_status" | jq -r '.Workflow.Stages | length // 0')

    echo -e "${BLUE}📋 Title:${NC} $title"
    echo -e "${BLUE}🎯 Current Stage:${NC} $current_stage/$total_stages"
    echo -e "${BLUE}📊 Status:${NC} $status"

    # Display stage information
    if [ "$total_stages" -gt 0 ]; then
        echo ""
        echo -e "${CYAN}🔄 Workflow Stages:${NC}"
        for ((i=1; i<=total_stages; i++)); do
            local stage_name=$(echo "$doc_status" | jq -r ".Workflow.Stages[$((i-1))].StageName // \"Stage $i\"")
            local required_approvals=$(echo "$doc_status" | jq -r ".Workflow.Stages[$((i-1))].RequiredCount // 1")
            local approvers=$(echo "$doc_status" | jq -r ".Workflow.Stages[$((i-1))].Approvers[]?" | tr '\n' ',' | sed 's/,$//')
            local stage_approvals=$(echo "$doc_status" | jq -r ".Workflow.Stages[$((i-1))].StageApprovals // {}")

            # Count approvals for this stage
            local approved_count=0
            if [ "$stage_approvals" != "{}" ] && [ "$stage_approvals" != "null" ]; then
                approved_count=$(echo "$stage_approvals" | jq '. | to_entries | map(select(.value.Status == "APPROVED")) | length')
            fi

            local stage_marker="  "
            if [ "$i" -eq "$current_stage" ]; then
                stage_marker="${YELLOW}➤${NC} "
            fi

            echo -e "${stage_marker}${BLUE}Stage $i:${NC} $stage_name ${GREEN}($approved_count/$required_approvals approved)${NC}"
            if [ -n "$approvers" ]; then
                echo -e "    ${GRAY}Approvers: $approvers${NC}"
            fi
        done
    fi

    # 3. Get deadline status to show deadline info
    local deadline_status
    deadline_status=$(execute_chaincode_query "GetDeadlineStatus" "$documentID" 2>/dev/null)

    if [ $? -eq 0 ] && [ -n "$deadline_status" ]; then
        echo ""
        echo -e "${CYAN}⏰ Stage Deadlines:${NC}"


        # Use DeadlineStatus.StageStatuses instead of .stageDeadlines
        local deadlines=$(echo "$deadline_status" | jq -c '.StageStatuses // {}')
        if [ "$deadlines" != "{}" ] && [ "$deadlines" != "null" ]; then
            echo "$deadlines" | jq -r 'to_entries[] | "  Stage \(.value.StageNumber): \(.value.Status) (Due: \(.value.Deadline // "Not set"))"'
        else
            echo -e "${GRAY}  No active deadlines${NC}"
        fi
    else
        echo ""
        echo -e "${YELLOW}⏰ No deadline information available${NC}"
    fi
    echo ""

    # 4. Stage selection with clear 1-based numbering
    echo -e "${YELLOW}Which stage to return to?${NC}"
    echo -e "${BLUE}Enter target stage number (1-based, e.g., 1 for first stage):${NC}"
    read -p "Target stage (1, 2, 3, etc.): " targetStage

    # Validate stage number
    if ! [[ "$targetStage" =~ ^[1-9][0-9]*$ ]]; then
        echo -e "${RED}❌ Invalid stage number. Must be a positive integer (1, 2, 3, etc.)${NC}"
        return
    fi

    read -p "Enter reason for rollback: " reason

    # 5. First attempt - check for deadline conflicts
    echo ""
    echo -e "${GREEN}🔄 Checking for deadline conflicts...${NC}"

    # Try rollback without deadline options to detect conflicts
    local initial_result
    initial_result=$(execute_chaincode_function "ReturnDocumentToStage" "$documentID" "$invokerId" "$targetStage" "$reason" "" 2>&1)
    local initial_exit_code=$?

    if [ $initial_exit_code -eq 0 ]; then
        # Success - no conflicts
        echo -e "${GREEN}✅ Rollback completed successfully!${NC}"
        echo -e "${BLUE}No deadline conflicts detected.${NC}"
        echo ""
        echo -e "${CYAN}=== ROLLBACK RESULTS ===${NC}"
        echo "$initial_result"
        return
    fi

    # Check if error is about deadline conflicts
    if echo "$initial_result" | grep -q "deadline conflicts"; then
        echo -e "${YELLOW}⚠️ DEADLINE CONFLICTS DETECTED${NC}"
        echo ""

        # 6. Deadline conflict resolution
        echo -e "${GREEN}CONFLICT RESOLUTION OPTIONS:${NC}"
        echo -e "${YELLOW}a${NC} - Cancel rollback"
        echo -e "${YELLOW}b${NC} - Remove conflicting deadlines (stages will have no deadlines)"
        echo -e "${YELLOW}c${NC} - Set new deadlines for affected stages"
        echo ""

        while true; do
            read -p "Choose resolution option (a/b/c): " resolution
            case $resolution in
                [Aa]* )
                    echo -e "${YELLOW}Rollback cancelled. Returning to menu...${NC}"
                    return;;
                [Bb]* )
                    # Remove conflicts option
                    local deadline_options='{"removeConflicts": true, "newDeadlines": {}}'
                    break;;
                [Cc]* )
                    # Set new deadlines option
                    echo ""
                    echo -e "${GREEN}Setting new deadlines for affected stages:${NC}"
                    echo -e "${BLUE}Enter deadline hours for each stage (or 'skip' to not set):${NC}"

                    declare -A new_deadlines
                    local json_deadlines="{"
                    local first=true

                    for ((i=targetStage; i<=5; i++)); do  # Assume max 5 stages
                        read -p "Stage $i deadline (hours from now, or 'skip'): " hours

                        if [[ "$hours" != "skip" ]] && [[ "$hours" =~ ^[1-9][0-9]*$ ]]; then
                            if [ "$first" = true ]; then
                                first=false
                            else
                                json_deadlines+=","
                            fi
                            json_deadlines+="\"$i\":$hours"
                        fi
                    done
                    json_deadlines+="}"

                    local deadline_options="{\"removeConflicts\": false, \"newDeadlines\": $json_deadlines}"
                    break;;
                * )
                    echo -e "${RED}Please choose a, b, or c.${NC}";;
            esac
        done

        # 7. Execute rollback with conflict resolution
        echo ""
        echo -e "${GREEN}🔄 Executing rollback with conflict resolution...${NC}"

        # Use properly formatted deadline options
        execute_chaincode_function "ReturnDocumentToStage" "$documentID" "$invokerId" "$targetStage" "$reason" "$deadline_options"

    else
        # Other error - display it
        echo -e "${RED}❌ Rollback failed:${NC}"
        echo "$initial_result"
    fi
}

# =============================================================================
# WORKFLOW HISTORY & ANALYSIS
# =============================================================================

get_document_workflow_history() {
    echo -e "${BLUE}=== 📈 Get Document Workflow History ===${NC}"
    echo -e "${YELLOW}Complete workflow progression across all versions${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "GetDocumentWorkflowHistory" "$documentID"
}

get_version_workflow_state() {
    echo -e "${BLUE}=== 🎯 Get Version Workflow State ===${NC}"
    echo -e "${YELLOW}Detailed workflow state for specific document version${NC}"
    echo ""

    read -p "Enter document ID: " documentID
    read -p "Enter version number: " version

    execute_chaincode_function "GetVersionWorkflowState" "$documentID" "$version"
}

# =============================================================================
# QUICK STATUS QUERIES
# =============================================================================

query_document_status() {
    echo -e "${PURPLE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}General document status including workflow information${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "QueryDocumentStatus" "$documentID"
}

get_all_documents() {
    echo -e "${PURPLE}=== 🔍 Get All Documents ===${NC}"
    echo -e "${YELLOW}List all documents for workflow management${NC}"
    echo ""

    execute_chaincode_function "GetAllDocuments"
}

# =============================================================================
# ENHANCED WORKFLOW CONFIGURATION HELPER FUNCTIONS
# =============================================================================

# Configure modifications to existing stages
configure_stage_modifications() {
    local current_doc_result="$1"
    local stage_count="$2"
    local operations="{}"

    echo -e "${YELLOW}Select which stages to modify:${NC}" >&2

    for ((i=1; i<=stage_count; i++)); do
        local stage_name=$(echo "$current_doc_result" | jq -r ".Workflow.Stages[$((i-1))].StageName // \"Stage $i\"")
        echo -e "${BLUE}$i${NC} - $stage_name" >&2
    done
    echo "" >&2

    local selected_stages=()
    while true; do
        read -p "Enter stage number to modify (or 'done' to finish): " stage_input
        if [[ "$stage_input" == "done" ]]; then
            break
        fi

        if [[ "$stage_input" =~ ^[1-9][0-9]*$ ]] && [[ "$stage_input" -le "$stage_count" ]]; then
            selected_stages+=("$stage_input")
            echo -e "${GREEN}✅ Stage $stage_input selected for modification${NC}" >&2
        else
            echo -e "${RED}❌ Invalid stage number. Please enter 1-$stage_count or 'done'${NC}" >&2
        fi
    done

    if [[ ${#selected_stages[@]} -eq 0 ]]; then
        echo "{}"
        return
    fi

    # Configure each selected stage
    local ops_json="{"
    local first=true

    for stage_num in "${selected_stages[@]}"; do
        local current_stage_data=$(echo "$current_doc_result" | jq ".Workflow.Stages[$((stage_num-1))]")
        local current_name=$(echo "$current_stage_data" | jq -r '.StageName // ""')
        local current_desc=$(echo "$current_stage_data" | jq -r '.Description // ""')
        local current_approvers=$(echo "$current_stage_data" | jq -r '.Approvers | join(", ")')
        local current_required=$(echo "$current_stage_data" | jq -r '.RequiredCount // 0')
        local current_advance=$(echo "$current_stage_data" | jq -r '.AutoAdvance // false')

        echo "" >&2
        echo -e "${CYAN}=== Configuring Stage $stage_num: $current_name ===${NC}" >&2
        echo -e "${YELLOW}Current settings:${NC}" >&2
        echo -e "  Name: $current_name" >&2
        echo -e "  Description: $current_desc" >&2
        echo -e "  Approvers: $current_approvers" >&2
        echo -e "  Required: $([ "$current_required" -eq 0 ] && echo "ALL" || echo "$current_required")" >&2
        echo -e "  Auto-advance: $current_advance" >&2
        echo "" >&2

        # Get new stage configuration
        read -p "New stage name (Enter to keep current): " new_name
        if [[ -z "$new_name" ]]; then
            new_name="$current_name"
        fi

        read -p "New description (Enter to keep current): " new_desc
        if [[ -z "$new_desc" ]]; then
            new_desc="$current_desc"
        fi

        echo "" >&2
        echo -e "${GREEN}Configure approvers for this stage:${NC}" >&2
        local change_approvers
        change_approvers=$(collect_boolean_input "Change approvers list?")

        local new_approvers="[]"
        if [[ "$change_approvers" == "true" ]]; then
            new_approvers=$(get_team_based_approvers)

            # Check if approver selection was successful
            if [[ $? -ne 0 || -z "$new_approvers" || "$new_approvers" == "[]" ]]; then
                echo -e "${RED}❌ No approvers selected. Keeping current approvers.${NC}" >&2
                new_approvers=$(echo "$current_stage_data" | jq '.Approvers')
            else
                echo -e "${BLUE}New approvers:${NC} $new_approvers" >&2
            fi
        else
            new_approvers=$(echo "$current_stage_data" | jq '.Approvers')
            echo -e "${BLUE}Keeping current approvers:${NC} $current_approvers" >&2
        fi

        echo "" >&2
        read -p "Required approvals (number, or Enter for ALL): " new_required
        if [[ -z "$new_required" || "$new_required" == "ALL" ]]; then
            new_required="0"
        elif ! [[ "$new_required" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}❌ Invalid number. Using current value ($current_required).${NC}" >&2
            new_required="$current_required"
        fi

        local new_advance
        new_advance=$(collect_boolean_input "Auto-advance to next stage?")

        # Escape strings for JSON
        local escaped_name=$(echo "$new_name" | sed 's/"/\\"/g')
        local escaped_desc=$(echo "$new_desc" | sed 's/"/\\"/g')

        # Build stage operation JSON (clean, no extra output)
        local stage_operation="{\"action\":\"update\",\"updatedStage\":{\"StageNumber\":$stage_num,\"StageName\":\"$escaped_name\",\"Description\":\"$escaped_desc\",\"Approvers\":$new_approvers,\"RequiredCount\":$new_required,\"AutoAdvance\":$new_advance,\"StageApprovals\":{}}}"

        # Add to operations JSON
        if [[ "$first" == "true" ]]; then
            first=false
        else
            ops_json+=","
        fi
        ops_json+="\"$stage_num\": $stage_operation"
    done

    ops_json+="}"
    echo "$ops_json"
}

# Configure new stages to add
configure_new_stages() {
    local stage_array=()

    echo -e "${YELLOW}Add new stages to the workflow:${NC}" >&2

    local stage_num=1
    while true; do
        echo "" >&2
        echo -e "${CYAN}--- New Stage $stage_num ---${NC}" >&2

        read -p "Stage name: " stage_name
        if [[ -z "$stage_name" ]]; then
            echo -e "${RED}❌ Stage name is required${NC}" >&2
            continue
        fi

        read -p "Stage description (optional): " stage_desc

        echo "" >&2
        echo -e "${GREEN}Configure approvers for this new stage:${NC}" >&2
        local approvers
        approvers=$(get_team_based_approvers)

        # Check if approver selection was successful
        if [[ $? -ne 0 || -z "$approvers" || "$approvers" == "[]" ]]; then
            echo -e "${RED}❌ No approvers selected. Stage creation cancelled.${NC}" >&2
            continue
        fi

        echo -e "${BLUE}Selected approvers:${NC} $approvers" >&2
        echo "" >&2

        read -p "Required approvals (number, or Enter for ALL): " required
        if [[ -z "$required" || "$required" == "ALL" ]]; then
            required="0"
        elif ! [[ "$required" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}❌ Invalid number. Using ALL instead.${NC}" >&2
            required="0"
        fi

        echo "" >&2
        local auto_advance
        auto_advance=$(collect_boolean_input "Auto-advance to next stage?")

        # Escape strings for JSON
        local escaped_name=$(echo "$stage_name" | sed 's/"/\\"/g')
        local escaped_desc=$(echo "$stage_desc" | sed 's/"/\\"/g')

        # Build new stage JSON (clean, no extra output)
        local new_stage="{\"StageNumber\":0,\"StageName\":\"$escaped_name\",\"Description\":\"$escaped_desc\",\"Approvers\":$approvers,\"RequiredCount\":$required,\"AutoAdvance\":$auto_advance,\"StageApprovals\":{}}"

        stage_array+=("$new_stage")

        echo "" >&2
        local add_more
        add_more=$(collect_boolean_input "Add another stage?")
        if [[ "$add_more" == "false" ]]; then
            break
        fi
        stage_num=$((stage_num + 1))
    done

    if [[ ${#stage_array[@]} -eq 0 ]]; then
        echo "[]"
        return
    fi

    # Combine stages into clean JSON array
    local stages_json="["
    for i in "${!stage_array[@]}"; do
        if [ $i -gt 0 ]; then
            stages_json+=","
        fi
        stages_json+="${stage_array[$i]}"
    done
    stages_json+="]"

    echo "$stages_json"
}

# Configure which stages to remove
configure_stage_removals() {
    local stage_count="$1"
    local remove_array=()

    echo -e "${YELLOW}Select stages to remove:${NC}" >&2
    echo -e "${RED}⚠️  Warning: Removed stages cannot be recovered${NC}" >&2
    echo "" >&2

    for ((i=1; i<=stage_count; i++)); do
        echo -e "${BLUE}$i${NC} - Stage $i" >&2
    done
    echo "" >&2

    while true; do
        read -p "Enter stage number to remove (or 'done' to finish): " stage_input
        if [[ "$stage_input" == "done" ]]; then
            break
        fi

        if [[ "$stage_input" =~ ^[1-9][0-9]*$ ]] && [[ "$stage_input" -le "$stage_count" ]]; then
            # Check if already selected
            local already_selected=false
            for selected in "${remove_array[@]}"; do
                if [[ "$selected" == "$stage_input" ]]; then
                    already_selected=true
                    break
                fi
            done

            if [[ "$already_selected" == "false" ]]; then
                remove_array+=("$stage_input")
                echo -e "${RED}❌ Stage $stage_input selected for removal${NC}" >&2
            else
                echo -e "${YELLOW}⚠️  Stage $stage_input already selected${NC}" >&2
            fi
        else
            echo -e "${RED}❌ Invalid stage number. Please enter 1-$stage_count or 'done'${NC}" >&2
        fi
    done

    if [[ ${#remove_array[@]} -eq 0 ]]; then
        echo "[]"
        return
    fi

    # Build JSON array
    local remove_json="["
    for i in "${!remove_array[@]}"; do
        if [ $i -gt 0 ]; then
            remove_json+=","
        fi
        remove_json+="${remove_array[$i]}"
    done
    remove_json+="]"

    echo "$remove_json"
}

# Configure deadline updates
configure_deadline_updates() {
    local stage_count="$1"
    local deadline_status="$2"  # Optional: current deadline status JSON

    echo -e "${YELLOW}Configure stage deadlines:${NC}" >&2
    echo "" >&2

    local updates_json="{"
    local first=true

    for ((i=1; i<=stage_count; i++)); do
        # Show current deadline info if available
        local current_deadline=""
        if [[ -n "$deadline_status" ]]; then
            current_deadline=$(echo "$deadline_status" | jq -r ".StageStatuses.stage_$i.Deadline // \"Not set\"" 2>/dev/null || echo "Not set")
            if [[ "$current_deadline" != "Not set" && "$current_deadline" != "null" ]]; then
                echo -e "${BLUE}Stage $i deadline configuration:${NC} ${CYAN}(Current: $current_deadline)${NC}" >&2
            else
                echo -e "${BLUE}Stage $i deadline configuration:${NC} ${PURPLE}(Current: No deadline set)${NC}" >&2
            fi
        else
            echo -e "${BLUE}Stage $i deadline configuration:${NC}" >&2
        fi
        local set_deadline
        set_deadline=$(collect_boolean_input "Set deadline for stage $i?")

        if [[ "$set_deadline" == "true" ]]; then
            read -p "Hours from now for stage $i deadline: " hours
            if [[ "$hours" =~ ^[1-9][0-9]*$ ]]; then
                if [[ "$first" == "true" ]]; then
                    first=false
                else
                    updates_json+=","
                fi
                updates_json+="\"$i\": $hours"
                echo -e "${GREEN}✅ Stage $i deadline set to $hours hours from now${NC}" >&2
            else
                echo -e "${RED}❌ Invalid hours value${NC}" >&2
            fi
        fi
        echo "" >&2
    done

    updates_json+="}"
    echo "$updates_json"
}

# Get team-based approvers with enhanced UX (replicating SubmitNewVersion pattern)
get_team_based_approvers() {
    echo -e "${GREEN}Select approver teams/individuals:${NC}" >&2
    echo -e "${BLUE}1${NC} - 🏛️  Legal Team (legal1, legal2, legal3)" >&2
    echo -e "${BLUE}2${NC} - 💻 Development Team (dev1, dev2, dev3)" >&2
    echo -e "${BLUE}3${NC} - 🔧 DevOps Team (devops1, devops2)" >&2
    echo -e "${BLUE}4${NC} - 👔 Management Team (manager1, manager2, manager3)" >&2
    echo -e "${BLUE}5${NC} - 🔐 Security Team (security1, security2)" >&2
    echo -e "${BLUE}6${NC} - 📋 Custom approvers (manual entry)" >&2
    echo "" >&2

    local selected_approvers=()

    while true; do
        read -p "Select team(s) [1-6] (or 'done' to finish): " team_choice

        if [[ "$team_choice" == "done" ]]; then
            break
        fi

        case $team_choice in
            1)
                # DETERMINISM FIX: Add team members in sorted order
                selected_approvers+=("legal1" "legal2" "legal3")
                echo -e "${GREEN}✅ Legal team added${NC}" >&2
                ;;
            2)
                # DETERMINISM FIX: Add team members in sorted order
                selected_approvers+=("dev1" "dev2" "dev3")
                echo -e "${GREEN}✅ Development team added${NC}" >&2
                ;;
            3)
                # DETERMINISM FIX: Add team members in sorted order
                selected_approvers+=("devops1" "devops2")
                echo -e "${GREEN}✅ DevOps team added${NC}" >&2
                ;;
            4)
                # DETERMINISM FIX: Add team members in sorted order
                selected_approvers+=("manager1" "manager2" "manager3")
                echo -e "${GREEN}✅ Management team added${NC}" >&2
                ;;
            5)
                # DETERMINISM FIX: Add team members in sorted order
                selected_approvers+=("security1" "security2")
                echo -e "${GREEN}✅ Security team added${NC}" >&2
                ;;
            6)
                echo "Enter custom approvers (comma-separated):" >&2
                read -p "Approvers: " custom_approvers
                IFS=',' read -ra ADDR <<< "$custom_approvers"
                for approver in "${ADDR[@]}"; do
                    trimmed=$(echo "$approver" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
                    if [[ -n "$trimmed" ]]; then
                        selected_approvers+=("$trimmed")
                    fi
                done
                echo -e "${GREEN}✅ Custom approvers added${NC}" >&2
                ;;
            *)
                echo -e "${RED}❌ Invalid choice. Please select 1-6 or 'done'${NC}" >&2
                ;;
        esac
    done

    # DETERMINISM FIX: Remove duplicates and sort for consistent order across peers
    local unique_approvers=($(printf "%s\n" "${selected_approvers[@]}" | sort -u))

    if [[ ${#unique_approvers[@]} -eq 0 ]]; then
        echo -e "${RED}❌ At least one approver is required${NC}" >&2
        return 1
    fi

    # DETERMINISM FIX: Build JSON array with sorted approvers for consistent output
    local approvers_json="["
    for i in "${!unique_approvers[@]}"; do
        if [ $i -gt 0 ]; then
            approvers_json+=","
        fi
        approvers_json+="\"${unique_approvers[$i]}\""
    done
    approvers_json+="]"

    echo "$approvers_json"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_workflow_menu
    read -p "Enter your choice [1-9]: " choice

    case $choice in
        1)
            configure_document_workflow
            ;;
        2)
            get_workflow_status
            ;;
        3)
            advance_document_to_next_stage
            ;;
        4)
            return_document_to_stage
            ;;
        5)
            get_document_workflow_history
            ;;
        6)
            get_version_workflow_state
            ;;
        7)
            query_document_status
            ;;
        8)
            get_all_documents
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
