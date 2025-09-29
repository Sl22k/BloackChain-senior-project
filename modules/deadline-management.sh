#!/bin/bash

# =============================================================================
# DEADLINE MANAGEMENT MODULE
# =============================================================================
# Deadline system: Set, Get, Remove deadlines for document approvals
# Embedded deadline tracking with RFC3339 timestamp consistency
# =============================================================================

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../shared/common-functions.sh"

# =============================================================================
# DEADLINE MANAGEMENT MENU
# =============================================================================

display_deadline_menu() {
    echo -e "${PURPLE}============================================${NC}"
    echo -e "${PURPLE}   ⏰ STAGE DEADLINE MANAGEMENT          ${NC}"
    echo -e "${PURPLE}   Unified Stage-Specific Deadlines      ${NC}"
    echo -e "${PURPLE}============================================${NC}"
    show_current_env
    echo -e "${GREEN}Select a deadline management function:${NC}"
    echo ""
    echo -e "${YELLOW}📅 Stage Deadline Operations:${NC}"
    echo -e "${YELLOW}1${NC} - ⏰ Set Stage Deadlines (Set deadlines for specific stages)"
    echo -e "${YELLOW}2${NC} - 📊 Get Deadline Status (Check all stage deadlines)"
    echo -e "${YELLOW}3${NC} - 🔍 Get Documents with Upcoming Deadlines (Search by timeframe)"
    echo -e "${YELLOW}4${NC} - 🗑️  Remove Stage Deadlines (Remove deadlines from stages)"
    echo ""
    echo -e "${BLUE}📋 Document Information:${NC}"
    echo -e "${BLUE}5${NC} - 📊 Query Document Status (Full document status)"
    echo -e "${BLUE}6${NC} - 🔍 Get All Documents (List all documents)"
    echo ""
    echo -e "${RED}7${NC} - 🔙 Return to Main Menu"
    echo ""
}

# =============================================================================
# DEADLINE OPERATIONS
# =============================================================================

set_stage_deadlines() {
    echo -e "${YELLOW}=== ⏰ Set Stage Deadlines ===${NC}"
    echo -e "${BLUE}Set deadlines for specific workflow stages${NC}"
    echo ""

    # 1. User Input
    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    # 2. Script Queries & Shows Preview
    echo ""
    echo -e "${GREEN}=== Loading Document Information ===${NC}"

    # First get document status to show current state
    local doc_status
    doc_status=$(execute_chaincode_query "QueryDocumentStatus" "$documentID" 2>/dev/null)

    if [ $? -ne 0 ] || [ -z "$doc_status" ]; then
        echo -e "${RED}❌ Failed to load document $documentID${NC}"
        echo -e "${YELLOW}Please verify the document ID exists and try again.${NC}"
        return
    fi

    # Parse and display comprehensive document overview
    echo -e "${CYAN}=== DOCUMENT: $documentID ===${NC}"

    # Extract basic document info
    local title=$(echo "$doc_status" | jq -r '.Title // "Untitled"' 2>/dev/null)
    local uploader=$(echo "$doc_status" | jq -r '.Uploader // "unknown"' 2>/dev/null)
    local current_stage=$(echo "$doc_status" | jq -r '.Workflow.CurrentStage // 0' 2>/dev/null)
    local total_stages=$(echo "$doc_status" | jq '.Workflow.Stages | length' 2>/dev/null)
    local status=$(echo "$doc_status" | jq -r '.Status // "unknown"' 2>/dev/null)

    echo -e "${BLUE}📄 Title:${NC} $title"
    echo -e "${BLUE}👤 Uploader:${NC} $uploader"
    echo -e "${BLUE}📊 Status:${NC} $status"
    echo -e "${BLUE}🎯 Current Stage:${NC} $current_stage of $total_stages"

    # Get deadline status for stage deadline information
    local deadline_status
    deadline_status=$(execute_chaincode_query "GetDeadlineStatus" "$documentID" 2>/dev/null)

    echo ""
    echo -e "${GREEN}=== DETAILED STAGE BREAKDOWN ===${NC}"

    # Display each stage with comprehensive information
    for ((i=0; i<total_stages; i++)); do
        local stage_num=$((i+1))
        local stage_name=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].StageName // \"Stage $stage_num\"" 2>/dev/null)
        local stage_desc=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].Description // \"\"" 2>/dev/null)
        local stage_approvers=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].Approvers | join(\", \")" 2>/dev/null)
        local required_count=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].RequiredCount // 0" 2>/dev/null)
        local auto_advance=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].AutoAdvance // false" 2>/dev/null)

        # Get approval information for this stage
        local stage_approvals=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].StageApprovals // {}" 2>/dev/null)
        local approval_count=0
        local approval_status=""

        # Count approvals and build status string
        if [[ "$stage_approvals" != "{}" && "$stage_approvals" != "null" ]]; then
            # Get approvers who have responded
            local approved_users=$(echo "$stage_approvals" | jq -r 'to_entries[] | select(.value == "APPROVED") | .key' 2>/dev/null)
            local rejected_users=$(echo "$stage_approvals" | jq -r 'to_entries[] | select(.value == "REJECTED") | .key' 2>/dev/null)

            # Count approvals
            if [[ -n "$approved_users" ]]; then
                approval_count=$(echo "$approved_users" | wc -l)
                approval_status="✅ $approval_count approved"
                if [[ -n "$rejected_users" ]]; then
                    local rejection_count=$(echo "$rejected_users" | wc -l)
                    approval_status="$approval_status, ❌ $rejection_count rejected"
                fi
            elif [[ -n "$rejected_users" ]]; then
                local rejection_count=$(echo "$rejected_users" | wc -l)
                approval_status="❌ $rejection_count rejected"
            else
                approval_status="🟡 Pending"
            fi
        else
            approval_status="🟡 Pending"
        fi

        # Get deadline for this stage
        local stage_deadline=""
        local deadline_info=""
        if [ $? -eq 0 ] && [ -n "$deadline_status" ]; then
            deadline_info=$(echo "$deadline_status" | jq -r ".StageStatuses.\"stage_$stage_num\".Deadline // \"\"" 2>/dev/null)
            if [[ -n "$deadline_info" && "$deadline_info" != "null" && "$deadline_info" != "" ]]; then
                local deadline_status_val=$(echo "$deadline_status" | jq -r ".StageStatuses.\"stage_$stage_num\".Status // \"\"" 2>/dev/null)
                case "$deadline_status_val" in
                    "OVERDUE") stage_deadline="🔴 OVERDUE: $deadline_info";;
                    "WARNING") stage_deadline="🟡 WARNING: $deadline_info";;
                    "ACTIVE") stage_deadline="🟢 ACTIVE: $deadline_info";;
                    "COMPLETED") stage_deadline="✅ COMPLETED: $deadline_info";;
                    *) stage_deadline="⏰ $deadline_info";;
                esac
            else
                stage_deadline="⏸️  No deadline set"
            fi
        else
            stage_deadline="⏸️  No deadline set"
        fi

        # Determine if this is the current stage
        local stage_indicator=""
        if [[ "$stage_num" -eq "$current_stage" ]]; then
            stage_indicator="👉 "
        else
            stage_indicator="   "
        fi

        # Display stage information
        echo -e "${stage_indicator}${YELLOW}📍 Stage $stage_num:${NC} $stage_name"
        if [[ -n "$stage_desc" && "$stage_desc" != "null" ]]; then
            echo -e "   ${BLUE}📝 Description:${NC} $stage_desc"
        fi
        echo -e "   ${BLUE}👥 Approvers:${NC} $stage_approvers"
        echo -e "   ${BLUE}✅ Required:${NC} $([ "$required_count" -eq 0 ] && echo "ALL" || echo "$required_count") | ${BLUE}⚡ Auto-advance:${NC} $auto_advance"
        echo -e "   ${BLUE}📊 Status:${NC} $approval_status"
        echo -e "   ${BLUE}⏰ Deadline:${NC} $stage_deadline"
        echo ""
    done

    echo -e "${GREEN}Document loaded successfully - Ready for deadline configuration${NC}"
    echo ""

    # 3. User Selection
    echo -e "${YELLOW}Which stages need deadline updates?${NC}"
    echo -e "${BLUE}Enter stage numbers separated by commas (e.g., 1,2,3):${NC}"
    read -p "Stage numbers: " stage_input

    # Parse comma-separated stage numbers
    IFS=',' read -ra STAGES <<< "$stage_input"

    # 4. Individual Deadline Input
    echo ""
    echo -e "${GREEN}=== Stage Deadline Configuration ===${NC}"

    declare -A stage_deadlines
    for stage in "${STAGES[@]}"; do
        # Trim whitespace
        stage=$(echo "$stage" | xargs)

        if ! [[ "$stage" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}❌ Invalid stage number: $stage${NC}"
            return
        fi

        echo -e "${YELLOW}Stage $stage:${NC}"
        read -p "New deadline (hours from now): " hours

        if ! [[ "$hours" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}❌ Hours must be a positive number${NC}"
            return
        fi

        stage_deadlines[$stage]=$hours
        echo -e "${GREEN}✅ Stage $stage: ${hours}h deadline${NC}"
        echo ""
    done

    # 5. Build JSON & Preview
    local json_string="{"
    local first=true
    for stage in "${!stage_deadlines[@]}"; do
        if [ "$first" = true ]; then
            first=false
        else
            json_string+=","
        fi
        json_string+="\"$stage\":${stage_deadlines[$stage]}"
    done
    json_string+="}"

    echo -e "${GREEN}=== Deadline Configuration Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo -e "${BLUE}Stage Deadlines JSON:${NC} $json_string"
    echo ""

    for stage in "${!stage_deadlines[@]}"; do
        echo -e "${CYAN}📅 Stage $stage: ${stage_deadlines[$stage]} hours from now${NC}"
    done
    echo ""

    while true; do
        read -p "Proceed with setting these stage deadlines? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Stage deadline setting cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    echo -e "${GREEN}⚙️ Executing: SetStageDeadlines${NC}"
    execute_chaincode_function "SetStageDeadlines" "$documentID" "$json_string" "$invokerId"
}

get_deadline_status() {
    echo -e "${YELLOW}=== 📊 Get Deadline Status ===${NC}"
    echo -e "${BLUE}Get real-time deadline status and information${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "GetDeadlineStatus" "$documentID"
}

get_documents_with_upcoming_deadlines() {
    echo -e "${YELLOW}=== 🔍 Get Documents with Upcoming Deadlines ===${NC}"
    echo -e "${BLUE}Search documents by deadline timeframe${NC}"
    echo ""

    echo -e "${GREEN}Deadline Search Configuration:${NC}"
    echo -e "${YELLOW}Enter the timeframe to look ahead for upcoming deadlines${NC}"
    read -p "Hours ahead to check (default 24): " hoursAhead

    if [[ "$hoursAhead" == "" ]]; then
        hoursAhead="24"
    fi

    # Validate hours ahead
    if ! [[ "$hoursAhead" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}❌ Hours ahead must be a positive number${NC}"
        return
    fi

    echo ""
    echo -e "${GREEN}=== Search Preview ===${NC}"
    echo -e "${BLUE}Search Timeframe:${NC} Next $hoursAhead hours"
    echo -e "${BLUE}Query Type:${NC} Documents with upcoming deadlines"
    echo ""

    execute_chaincode_function "GetDocumentsWithUpcomingDeadlines" "$hoursAhead"
}

remove_stage_deadlines() {
    echo -e "${YELLOW}=== 🗑️  Remove Stage Deadlines ===${NC}"
    echo -e "${BLUE}Remove deadlines from specific workflow stages${NC}"
    echo ""

    # User Input
    read -p "Enter document ID: " documentID
    read -p "Enter your identity (invoker): " invokerId

    # Show current deadline status first
    echo ""
    echo -e "${GREEN}=== Current Deadline Status ===${NC}"

    local deadline_status
    deadline_status=$(execute_chaincode_query "GetDeadlineStatus" "$documentID" 2>/dev/null)

    if [ $? -ne 0 ] || [ -z "$deadline_status" ]; then
        echo -e "${RED}❌ Failed to load deadline status for document $documentID${NC}"
        echo -e "${YELLOW}Please verify the document ID exists and try again.${NC}"
        return
    fi

    # Get document status for comprehensive information
    local doc_status
    doc_status=$(execute_chaincode_query "QueryDocumentStatus" "$documentID" 2>/dev/null)

    if [ $? -ne 0 ] || [ -z "$doc_status" ]; then
        echo -e "${RED}❌ Failed to load document information${NC}"
        echo -e "${YELLOW}Please verify the document ID exists and try again.${NC}"
        return
    fi

    # Parse and display comprehensive document overview
    echo -e "${CYAN}=== DOCUMENT: $documentID ===${NC}"

    # Extract basic document info
    local title=$(echo "$doc_status" | jq -r '.Title // "Untitled"' 2>/dev/null)
    local uploader=$(echo "$doc_status" | jq -r '.Uploader // "unknown"' 2>/dev/null)
    local current_stage=$(echo "$doc_status" | jq -r '.Workflow.CurrentStage // 0' 2>/dev/null)
    local total_stages=$(echo "$doc_status" | jq '.Workflow.Stages | length' 2>/dev/null)
    local status=$(echo "$doc_status" | jq -r '.Status // "unknown"' 2>/dev/null)

    echo -e "${BLUE}📄 Title:${NC} $title"
    echo -e "${BLUE}👤 Uploader:${NC} $uploader"
    echo -e "${BLUE}📊 Status:${NC} $status"
    echo -e "${BLUE}🎯 Current Stage:${NC} $current_stage of $total_stages"

    echo ""
    echo -e "${GREEN}=== CURRENT DEADLINES TO REMOVE ===${NC}"

    # Show which stages currently have deadlines with full detail
    local stage_count=0
    local stages_with_deadlines=()

    # Display each stage with deadline information
    for ((i=0; i<total_stages; i++)); do
        local stage_num=$((i+1))
        local stage_name=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].StageName // \"Stage $stage_num\"" 2>/dev/null)
        local stage_approvers=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].Approvers | join(\", \")" 2>/dev/null)

        # Get deadline for this stage
        local stage_deadline=""
        local deadline_info=""
        local has_deadline=false

        deadline_info=$(echo "$deadline_status" | jq -r ".StageStatuses.\"stage_$stage_num\".Deadline // \"\"" 2>/dev/null)
        if [[ -n "$deadline_info" && "$deadline_info" != "null" && "$deadline_info" != "" ]]; then
            has_deadline=true
            stages_with_deadlines+=("$stage_num")
            stage_count=$((stage_count + 1))

            local deadline_status_val=$(echo "$deadline_status" | jq -r ".StageStatuses.\"stage_$stage_num\".Status // \"\"" 2>/dev/null)
            case "$deadline_status_val" in
                "OVERDUE") stage_deadline="🔴 OVERDUE: $deadline_info";;
                "WARNING") stage_deadline="🟡 WARNING: $deadline_info";;
                "ACTIVE") stage_deadline="🟢 ACTIVE: $deadline_info";;
                "COMPLETED") stage_deadline="✅ COMPLETED: $deadline_info";;
                *) stage_deadline="⏰ $deadline_info";;
            esac
        else
            stage_deadline="⏸️  No deadline set"
        fi

        # Only show stages with deadlines prominently
        if [[ "$has_deadline" == "true" ]]; then
            # Determine if this is the current stage
            local stage_indicator=""
            if [[ "$stage_num" -eq "$current_stage" ]]; then
                stage_indicator="👉 "
            else
                stage_indicator="   "
            fi

            echo -e "${stage_indicator}${YELLOW}📍 Stage $stage_num:${NC} $stage_name"
            echo -e "   ${BLUE}👥 Approvers:${NC} $stage_approvers"
            echo -e "   ${RED}🗑️  Current Deadline:${NC} $stage_deadline"
            echo ""
        fi
    done

    if [[ $stage_count -eq 0 ]]; then
        echo -e "${YELLOW}⚠️  No stages currently have deadlines set${NC}"
        echo -e "${BLUE}Nothing to remove. Use 'Set Stage Deadlines' to add deadlines first.${NC}"
        echo ""
        echo -e "${CYAN}=== ALL STAGES (for reference) ===${NC}"

        # Show all stages without deadlines for reference
        for ((i=0; i<total_stages; i++)); do
            local stage_num=$((i+1))
            local stage_name=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].StageName // \"Stage $stage_num\"" 2>/dev/null)
            local stage_approvers=$(echo "$doc_status" | jq -r ".Workflow.Stages[$i].Approvers | join(\", \")" 2>/dev/null)

            echo -e "   ${YELLOW}📍 Stage $stage_num:${NC} $stage_name"
            echo -e "   ${BLUE}👥 Approvers:${NC} $stage_approvers"
            echo -e "   ${BLUE}⏰ Deadline:${NC} ⏸️  No deadline set"
            echo ""
        done
        return
    fi

    echo -e "${GREEN}✅ Found $stage_count stage(s) with deadlines available for removal${NC}"
    echo ""

    # User Selection
    echo -e "${YELLOW}Which stage deadlines do you want to remove?${NC}"
    echo -e "${BLUE}Enter stage numbers separated by commas (e.g., 1,3,5):${NC}"
    read -p "Stage numbers to remove: " stage_input

    # Parse comma-separated stage numbers
    IFS=',' read -ra STAGES <<< "$stage_input"

    # Validate and build JSON array
    local valid_stages=()
    for stage in "${STAGES[@]}"; do
        # Trim whitespace
        stage=$(echo "$stage" | xargs)

        if ! [[ "$stage" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}❌ Invalid stage number: $stage${NC}"
            return
        fi

        valid_stages+=("$stage")
    done

    # Build JSON array: ["1", "3", "5"]
    local json_array="["
    local first=true
    for stage in "${valid_stages[@]}"; do
        if [ "$first" = true ]; then
            first=false
        else
            json_array+=","
        fi
        json_array+="\"$stage\""
    done
    json_array+="]"

    echo ""
    echo -e "${GREEN}=== Deadline Removal Preview ===${NC}"
    echo -e "${BLUE}Document:${NC} $documentID"
    echo -e "${BLUE}Invoker:${NC} $invokerId"
    echo -e "${BLUE}Stages JSON:${NC} $json_array"
    echo ""

    for stage in "${valid_stages[@]}"; do
        echo -e "${RED}🗑️  Stage $stage: Remove deadline${NC}"
    done
    echo ""

    while true; do
        read -p "Proceed with removing these stage deadlines? [y/n]: " confirm
        case $confirm in
            [Yy]* ) break;;
            [Nn]* )
                echo -e "${YELLOW}Stage deadline removal cancelled. Returning to menu...${NC}"
                return;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done

    echo -e "${GREEN}⚙️ Executing: RemoveStageDeadlines${NC}"
    execute_chaincode_function "RemoveStageDeadlines" "$documentID" "$json_array" "$invokerId"
}

# =============================================================================
# DOCUMENT INFORMATION
# =============================================================================

query_document_status() {
    echo -e "${BLUE}=== 📊 Query Document Status ===${NC}"
    echo -e "${YELLOW}Get complete document status including deadline information${NC}"
    echo ""

    read -p "Enter document ID: " documentID

    execute_chaincode_function "QueryDocumentStatus" "$documentID"
}

get_all_documents() {
    echo -e "${BLUE}=== 🔍 Get All Documents ===${NC}"
    echo -e "${YELLOW}List all documents for deadline management${NC}"
    echo ""

    execute_chaincode_function "GetAllDocuments"
}

# =============================================================================
# MODULE MAIN LOOP
# =============================================================================

while true; do
    display_deadline_menu
    read -p "Enter your choice [1-7]: " choice

    case $choice in
        1)
            set_stage_deadlines
            ;;
        2)
            get_deadline_status
            ;;
        3)
            get_documents_with_upcoming_deadlines
            ;;
        4)
            remove_stage_deadlines
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