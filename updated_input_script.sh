#!/bin/bash

# Define color codes for better visibility
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'  # No Color

# Function to display the menu of functions
display_menu() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Document Management Chaincode Menu  ${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${GREEN}📋 Document Lifecycle (Dual Submission System):${NC}"
    for i in $(seq 0 6); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${BLUE}📄 Document Information:${NC}"
    for i in $(seq 7 11); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${PURPLE}⚙️  Administrative Functions:${NC}"
    for i in $(seq 12 17); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${BLUE}🔍 CouchDB Rich Queries:${NC}"
    for i in $(seq 18 22); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${GREEN}🔄 Advanced Workflow Functions:${NC}"
    for i in $(seq 23 27); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${PURPLE}⏰ Deadline Management:${NC}"
    for i in $(seq 28 31); do
        echo -e "${YELLOW}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${RED}⚠️  Legacy Functions (Use Workflow Functions Instead):${NC}"
    for i in $(seq 32 $((${#FUNCTIONS[@]} - 1))); do
        echo -e "${RED}$((i + 1))${NC} - ${FUNCTIONS[$i]}"
    done
    echo ""
    echo -e "${RED}$(( ${#FUNCTIONS[@]} + 1 ))${NC} - Exit"
    echo ""
}

# Function to format and display JSON output with colors
format_json_output() {
    local output="$1"
    
    # Check if output contains JSON (starts with { or [)
    if [[ "$output" =~ ^\s*[\{\[] ]]; then
        # Pretty print JSON and add colors
        echo "$output" | jq . 2>/dev/null | sed \
            -e 's/"APPROVED"/"✅ APPROVED"/g' \
            -e 's/"REJECTED"/"❌ REJECTED"/g' \
            -e 's/"PENDING"/"🟡 PENDING"/g' \
            -e 's/"NEEDS_REVISION"/"🔄 NEEDS_REVISION"/g' \
            -e 's/"NEEDS_INFO"/"ℹ️  NEEDS_INFO"/g' \
            -e 's/"true"/🟢 true/g' \
            -e 's/"false"/🔴 false/g'
    else
        # Regular output, just add some status indicators
        echo "$output" | sed \
            -e 's/APPROVED/✅ APPROVED/g' \
            -e 's/REJECTED/❌ REJECTED/g' \
            -e 's/PENDING/🟡 PENDING/g' \
            -e 's/NEEDS_REVISION/🔄 NEEDS_REVISION/g' \
            -e 's/SUCCESS/✅ SUCCESS/g' \
            -e 's/ERROR/❌ ERROR/g'
    fi
}

# Function to execute the constructed command with enhanced output
execute_command() {
    local command="$1"
    echo -e "${GREEN}Executing command:${NC}"
    echo -e "${PURPLE}$command${NC}"
    echo ""
    
    # Execute and capture output
    local output
    output=$(eval "$command" 2>&1)
    local exit_code=$?
    
    echo -e "${GREEN}=== Command Output ===${NC}"
    if [ $exit_code -eq 0 ]; then
        format_json_output "$output"
    else
        echo -e "${RED}❌ Command failed with exit code: $exit_code${NC}"
        echo -e "${RED}$output${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}Command completed. Press Enter to continue...${NC}"
    read
}

# Function to collect array input and format it as escaped JSON string
collect_array_input() {
    local prompt="$1"
    # Use stderr for prompts so they don't get captured by $()
    echo -e "${BLUE}$prompt${NC}" >&2
    echo -e "${YELLOW}(Enter comma-separated values, e.g., user1,user2,user3)${NC}" >&2
    read -p "> " input
    # Remove spaces and format as a properly escaped JSON array string
    if [ -z "$input" ]; then
        echo "[]"
    else
        # Format as JSON array and escape quotes for JSON context
        formatted_input=$(echo "$input" | tr -d ' ' | sed 's/,/","/g; s/^/["/; s/$/"]/')
        escaped_input=$(echo "$formatted_input" | sed 's/"/\\"/g')
        echo "$escaped_input"
    fi
}

# Enhanced menu-driven input collection functions

# Predefined decision sets
DECISION_SETS=(
    "APPROVED,REJECTED"
    "APPROVED,REJECTED,NEEDS_REVISION"
    "APPROVED,REJECTED,NEEDS_REVISION,NEEDS_INFO"
    "APPROVE,REJECT"
)

# Predefined approver groups
APPROVER_GROUPS=(
    "legal1,legal2,legal3"
    "dev1,dev2,architect1"
    "ceo,cfo,cto"
    "finance1,finance2,controller"
    "legal1,dev1,finance1"
)

# Function to collect decision sets with menu
collect_decision_set() {
    echo -e "${GREEN}=== Choose Decision Set ===${NC}" >&2
    echo -e "${YELLOW}1${NC} - Standard (APPROVED, REJECTED)" >&2
    echo -e "${YELLOW}2${NC} - Extended (APPROVED, REJECTED, NEEDS_REVISION)" >&2
    echo -e "${YELLOW}3${NC} - Complete (APPROVED, REJECTED, NEEDS_REVISION, NEEDS_INFO)" >&2
    echo -e "${YELLOW}4${NC} - Binary (APPROVE, REJECT)" >&2
    echo -e "${YELLOW}5${NC} - Custom (Enter your own decisions)" >&2
    echo "" >&2
    
    while true; do
        read -p "Enter choice [1-5]: " choice
        case $choice in
            1) echo "[\\\"APPROVED\\\",\\\"REJECTED\\\"]"; break;;
            2) echo "[\\\"APPROVED\\\",\\\"REJECTED\\\",\\\"NEEDS_REVISION\\\"]"; break;;
            3) echo "[\\\"APPROVED\\\",\\\"REJECTED\\\",\\\"NEEDS_REVISION\\\",\\\"NEEDS_INFO\\\"]"; break;;
            4) echo "[\\\"APPROVE\\\",\\\"REJECT\\\"]"; break;;
            5) 
                echo -e "${BLUE}Enter custom decisions:${NC}" >&2
                echo -e "${YELLOW}(Enter comma-separated values, e.g., APPROVED,REJECTED,PENDING)${NC}" >&2
                read -p "> " custom_input
                if [ ! -z "$custom_input" ]; then
                    formatted_input=$(echo "$custom_input" | tr -d ' ' | sed 's/,/","/g; s/^/["/; s/$/"]/')
                    escaped_input=$(echo "$formatted_input" | sed 's/"/\\"/g')
                    echo "$escaped_input"
                    break
                else
                    echo -e "${RED}Please enter at least one decision.${NC}" >&2
                fi
                ;;
            *) echo -e "${RED}Invalid choice. Please select 1-5.${NC}" >&2;;
        esac
    done
}

# Function to collect approver groups with menu
collect_approver_group() {
    echo -e "${GREEN}=== Choose Approver Group ===${NC}" >&2
    echo -e "${YELLOW}1${NC} - Legal Team (legal1, legal2, legal3)" >&2
    echo -e "${YELLOW}2${NC} - Technical Team (dev1, dev2, architect1)" >&2
    echo -e "${YELLOW}3${NC} - Executive Team (ceo, cfo, cto)" >&2
    echo -e "${YELLOW}4${NC} - Finance Team (finance1, finance2, controller)" >&2
    echo -e "${YELLOW}5${NC} - Mixed Review (legal1, dev1, finance1)" >&2
    echo -e "${YELLOW}6${NC} - Single Approver (Choose one person)" >&2
    echo -e "${YELLOW}7${NC} - Custom (Enter your own approvers)" >&2
    echo "" >&2
    
    while true; do
        read -p "Enter choice [1-7]: " choice
        case $choice in
            1) echo "[\\\"legal1\\\",\\\"legal2\\\",\\\"legal3\\\"]"; break;;
            2) echo "[\\\"dev1\\\",\\\"dev2\\\",\\\"architect1\\\"]"; break;;
            3) echo "[\\\"ceo\\\",\\\"cfo\\\",\\\"cto\\\"]"; break;;
            4) echo "[\\\"finance1\\\",\\\"finance2\\\",\\\"controller\\\"]"; break;;
            5) echo "[\\\"legal1\\\",\\\"dev1\\\",\\\"finance1\\\"]"; break;;
            6) 
                echo -e "${BLUE}Enter single approver username:${NC}" >&2
                read -p "> " single_user
                if [ ! -z "$single_user" ]; then
                    echo "[\\\"$single_user\\\"]"
                    break
                else
                    echo -e "${RED}Please enter a username.${NC}" >&2
                fi
                ;;
            7) 
                echo -e "${BLUE}Enter custom approvers:${NC}" >&2
                echo -e "${YELLOW}(Enter comma-separated usernames, e.g., alice,bob,charlie)${NC}" >&2
                read -p "> " custom_input
                if [ ! -z "$custom_input" ]; then
                    formatted_input=$(echo "$custom_input" | tr -d ' ' | sed 's/,/","/g; s/^/["/; s/$/"]/')
                    escaped_input=$(echo "$formatted_input" | sed 's/"/\\"/g')
                    echo "$escaped_input"
                    break
                else
                    echo -e "${RED}Please enter at least one approver.${NC}" >&2
                fi
                ;;
            *) echo -e "${RED}Invalid choice. Please select 1-7.${NC}" >&2;;
        esac
    done
}

# Document type quick setup combinations
collect_document_quick_setup() {
    echo -e "${GREEN}=== Document Type Quick Setup ===${NC}" >&2
    echo -e "${YELLOW}1${NC} - Contract Review (Legal + Finance teams, Extended decisions)" >&2
    echo -e "${YELLOW}2${NC} - Technical Spec (Tech team, Standard decisions)" >&2
    echo -e "${YELLOW}3${NC} - Policy Document (Executive team, Complete decisions)" >&2
    echo -e "${YELLOW}4${NC} - Budget Request (Finance + Executive, Standard decisions)" >&2
    echo -e "${YELLOW}5${NC} - Custom Setup (Choose everything manually)" >&2
    echo "" >&2
    
    while true; do
        read -p "Enter choice [1-5]: " choice
        case $choice in
            1) 
                echo "CONTRACT_REVIEW"
                echo "[\\\"legal1\\\",\\\"legal2\\\",\\\"finance1\\\",\\\"finance2\\\"]"
                echo "[\\\"APPROVED\\\",\\\"REJECTED\\\",\\\"NEEDS_REVISION\\\"]"
                break;;
            2) 
                echo "TECHNICAL_SPEC"
                echo "[\\\"dev1\\\",\\\"dev2\\\",\\\"architect1\\\"]"
                echo "[\\\"APPROVED\\\",\\\"REJECTED\\\"]"
                break;;
            3) 
                echo "POLICY_DOCUMENT"
                echo "[\\\"ceo\\\",\\\"cfo\\\",\\\"cto\\\"]"
                echo "[\\\"APPROVED\\\",\\\"REJECTED\\\",\\\"NEEDS_REVISION\\\",\\\"NEEDS_INFO\\\"]"
                break;;
            4) 
                echo "BUDGET_REQUEST"
                echo "[\\\"finance1\\\",\\\"finance2\\\",\\\"cfo\\\"]"
                echo "[\\\"APPROVED\\\",\\\"REJECTED\\\"]"
                break;;
            5) 
                echo "CUSTOM"
                break;;
            *) echo -e "${RED}Invalid choice. Please select 1-5.${NC}" >&2;;
        esac
    done
}

# Enhanced configuration preview
show_config_preview() {
    local doc_id="$1"
    local doc_type="$2"
    local approvers="$3"
    local decisions="$4"
    
    # Count approvers by parsing JSON-like string
    approver_count=$(echo "$approvers" | grep -o '"[^"]*"' | wc -l)
    decision_count=$(echo "$decisions" | grep -o '"[^"]*"' | wc -l)
    
    echo ""
    echo -e "${GREEN}=== Configuration Preview ===${NC}"
    echo -e "${BLUE}Document ID:${NC} $doc_id"
    echo -e "${BLUE}Document Type:${NC} $doc_type"
    echo -e "${BLUE}Approvers:${NC} $approvers ($approver_count users)"
    echo -e "${BLUE}Decisions:${NC} $decisions ($decision_count options)"
    echo -e "${BLUE}Auto-workflow:${NC} Single stage, ALL approvers required"
    echo ""
    
    while true; do
        read -p "Proceed with this configuration? [y/n]: " confirm
        case $confirm in
            [Yy]* ) return 0;;
            [Nn]* ) return 1;;
            * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
        esac
    done
}

# Function to collect yes/no input
collect_boolean_input() {
    local prompt="$1"
    while true; do
        read -p "$prompt (yes/no): " input
        case "$input" in
            [Yy]* | [Yy][Ee][Ss]* ) echo "true"; break;;
            [Nn]* | [Nn][Oo]* ) echo "false"; break;;
            * ) echo -e "${RED}Please answer yes or no.${NC}";;
        esac
    done
}

# Define chaincode functions (dummy implementations for local testing)
InitLedger() { echo "InitLedger executed"; }
SubmitDocument() { echo "SubmitDocument (unified) executed with args: $@"; }
SubmitNewVersion() { echo "SubmitNewVersion executed with args: $@"; }
ApproveDocument() { echo "ApproveDocument executed with args: $@"; }
QueryDocumentStatus() { echo "QueryDocumentStatus executed with args: $@"; }
AddEditor() { echo "AddEditor executed with args: $@"; }
RemoveEditor() { echo "RemoveEditor executed with args: $@"; }
AddPrivilegedEditor() { echo "AddPrivilegedEditor executed with args: $@"; }
RemovePrivilegedEditor() { echo "RemovePrivilegedEditor executed with args: $@"; }
UpdateDocumentApprovers() { echo "UpdateDocumentApprovers executed with args: $@"; }
UpdateValidDecisions() { echo "UpdateValidDecisions executed with args: $@"; }
GetDocumentIdByHash() { echo "GetDocumentIdByHash executed with args: $@"; }
DocumentExists() { echo "DocumentExists executed with args: $@"; }
GetDocHistory() { echo "GetDocHistory executed with args: $@"; }
GetAllDocuments() { echo "GetAllDocuments executed"; }
GetDocumentsByEditor() { echo "GetDocumentsByEditor executed with args: $@"; }
GetDocumentsByApprover() { echo "GetDocumentsByApprover executed with args: $@"; }
GetPendingApprovals() { echo "GetPendingApprovals executed with args: $@"; }
GetDocumentSummary() { echo "GetDocumentSummary executed with args: $@"; }

# Rich Query Functions
GetDocumentsByDateRange() { echo "GetDocumentsByDateRange executed with args: $@"; }
GetRecentDocuments() { echo "GetRecentDocuments executed with args: $@"; }
GetDocumentsByUploaderSorted() { echo "GetDocumentsByUploaderSorted executed with args: $@"; }
GetDocumentsWithPendingApprovals() { echo "GetDocumentsWithPendingApprovals executed with args: $@"; }
GetDocumentStats() { echo "GetDocumentStats executed with args: $@"; }

# Workflow Functions  
ConfigureDocumentWorkflow() { echo "ConfigureDocumentWorkflow executed with args: $@"; }
GetWorkflowStatus() { echo "GetWorkflowStatus executed with args: $@"; }
ReturnDocumentToStage() { echo "ReturnDocumentToStage executed with args: $@"; }
# NEW: Workflow Audit Functions - FIX FOR BUG #2
GetDocumentWorkflowHistory() { echo "GetDocumentWorkflowHistory executed with args: $@"; }
GetVersionWorkflowState() { echo "GetVersionWorkflowState executed with args: $@"; }

# Deadline Functions - Embedded System
SetDocumentDeadline() { echo "SetDocumentDeadline executed with args: $@"; }
GetDeadlineStatus() { echo "GetDeadlineStatus executed with args: $@"; }
GetDocumentsWithUpcomingDeadlines() { echo "GetDocumentsWithUpcomingDeadlines executed with args: $@"; }
RemoveDocumentDeadline() { echo "RemoveDocumentDeadline executed with args: $@"; }

# Array of function names - organized by category
FUNCTIONS=(
    # ===== CORE DOCUMENT LIFECYCLE =====
    "InitLedger"                        # Initialize the system
    "SubmitDocument"                    # ✅ UNIFIED submission (progressive disclosure)
    "ApproveDocument"                   # ✅ UNIFIED approval (simple & workflow modes) 
    "SubmitNewVersion"                  # Submit new version of existing document
    
    # ===== DOCUMENT QUERIES =====
    "QueryDocumentStatus"               # 🎯 PRIMARY: Complete document status & workflow
    "DocumentExists"                    # Check if document exists
    "GetDocumentIdByHash"               # Find document by hash
    "GetDocHistory"                     # Get version history
    "GetAllDocuments"                   # List all documents
    "GetDocumentSummary"                # Get document summary
    
    # ===== WORKFLOW MANAGEMENT =====
    "ConfigureDocumentWorkflow"         # Set up multi-stage workflows
    "GetWorkflowStatus"                 # Get workflow state details
    "ReturnDocumentToStage"             # Return document to previous stage
    "GetDocumentWorkflowHistory"        # Get complete workflow history
    "GetVersionWorkflowState"           # Get workflow state for specific version
    
    # ===== DOCUMENT ADMINISTRATION =====
    "AddEditor"                         # Add document editor
    "RemoveEditor"                      # Remove document editor
    "AddPrivilegedEditor"               # Add privileged editor
    "RemovePrivilegedEditor"            # Remove privileged editor
    "UpdateDocumentApprovers"           # Update document approvers
    "UpdateValidDecisions"              # Update valid decisions
    
    # ===== DEADLINE MANAGEMENT =====
    "SetDocumentDeadline"               # Set document deadlines
    "GetDeadlineStatus"                 # Get real-time deadline status
    "GetDocumentsWithUpcomingDeadlines" # Query documents by deadline
    "RemoveDocumentDeadline"            # Remove document deadline
    
    # ===== ADVANCED QUERIES =====
    "GetDocumentsByDateRange"           # Query by date range
    "GetRecentDocuments"                # Get recently modified documents  
    "GetDocumentsByUploaderSorted"      # Query by uploader
    "GetDocumentsWithPendingApprovals"  # Find documents needing approval
    "GetDocumentStats"                  # Get system statistics
    "GetDocumentsByEditor"              # Get documents by editor
    "GetDocumentsByApprover"            # Get documents by approver  
    "GetPendingApprovals"               # Get pending approvals for user
)

# Functions that require invoke (state-changing operations)
INVOKE_FUNCTIONS=(
    # Core Lifecycle (state-changing)
    "InitLedger" 
    "SubmitDocument"                    # ✅ UNIFIED submission
    "ApproveDocument"                   # ✅ UNIFIED approval
    "SubmitNewVersion"
    
    # Workflow Management (state-changing) 
    "ConfigureDocumentWorkflow"
    "ReturnDocumentToStage"
    
    # Administration (state-changing)
    "AddEditor" 
    "RemoveEditor" 
    "AddPrivilegedEditor" 
    "RemovePrivilegedEditor"
    "UpdateDocumentApprovers" 
    "UpdateValidDecisions"
    
    # Deadline Management (state-changing)
    "SetDocumentDeadline"
    "RemoveDocumentDeadline"
)

# Function to check if a function requires invoke
is_invoke_function() {
    local func_name="$1"
    for invoke_func in "${INVOKE_FUNCTIONS[@]}"; do
        if [[ "$invoke_func" == "$func_name" ]]; then
            return 0
        fi
    done
    return 1
}

# Functions that return plain strings (not JSON) - don't use jq with these
STRING_FUNCTIONS=(
    "GetDocumentIdByHash"
    "DocumentExists"
)

# Function to check if a function returns string (not JSON)
is_string_function() {
    local func_name="$1"
    for string_func in "${STRING_FUNCTIONS[@]}"; do
        if [[ "$string_func" == "$func_name" ]]; then
            return 0
        fi
    done
    return 1
}

# Main loop
while true; do
    #clear
    display_menu
    read -p "Enter the function number or name: " choice

    # Exit option
    if [[ "$choice" == "$(( ${#FUNCTIONS[@]} + 1 ))" || "$choice" == "exit" ]]; then
        echo -e "${GREEN}Exiting the script. Goodbye!${NC}"
        break
    fi

    # Determine function name
    if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#FUNCTIONS[@]} )); then
        func_name="${FUNCTIONS[$((choice - 1))]}"
    else
        func_name="$choice"
    fi

    # Check if function is valid
    if declare -f "$func_name" > /dev/null; then
        echo ""
        echo -e "${GREEN}Selected function: $func_name${NC}"
        echo -e "${YELLOW}Please provide the required parameters:${NC}"
        echo ""
        
        args=()

        # Collect args based on function
        case "$func_name" in
            "InitLedger")
                echo -e "${BLUE}No parameters required for InitLedger${NC}"
                ;;
                
            "SubmitDocument")
                echo -e "${GREEN}=== Submit New Document (Progressive Workflow Builder) ===${NC}"
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
                    stage1_required="0"
                fi
                
                # Check if user wants to add more stages (progressive disclosure)
                echo ""
                echo -e "${PURPLE}💡 Progressive Disclosure: Add More Stages?${NC}"
                echo -e "${YELLOW}• Current: Single-stage workflow (simple)${NC}"
                echo -e "${YELLOW}• Option: Add more stages for complex approval process${NC}"
                echo ""
                
                read -p "Add new stage? (y/n): " add_more_stages
                
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
                    
                    # Extract approvers from Stage 1 for raw JSON building
                    stage1_approvers_list=$(echo "$stage1_approvers_json" | sed 's/^\[\\\"/\"/' | sed 's/\\\"\]$/\"/' | sed 's/\\\",\\\"/\",\"/g')
                    
                    # Build workflow stages JSON without escaping (for raw JSON)
                    workflow_stages=""
                    escaped_stage1_name=$(echo "$stage1_name" | sed 's/"/\\"/g')
                    escaped_stage1_description=$(echo "$stage1_description" | sed 's/"/\\"/g')
                    stage1_json="{\"StageNumber\":1,\"StageName\":\"$escaped_stage1_name\",\"Approvers\":[$stage1_approvers_list],\"RequiredCount\":$stage1_required,\"AutoAdvance\":true,\"Description\":\"$escaped_stage1_description\"}"
                    workflow_stages="$stage1_json"
                    
                    stage_count=1
                    
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
                        stage_approvers_list=$(echo "$stage_approvers_json" | sed 's/^\[\\\"/\"/' | sed 's/\\\"\]$/\"/' | sed 's/\\\",\\\"/\",\"/g')
                        
                        echo ""
                        read -p "Required approval count (default: ALL): " requiredCount
                        if [[ "$requiredCount" == "" || "$requiredCount" == "ALL" ]]; then
                            requiredCount="0"
                        fi
                        
                        autoAdvance=$(collect_boolean_input "Auto-advance to next stage? (default: n)")
                        if [[ "$autoAdvance" == "" ]]; then
                            autoAdvance="false"
                        fi
                        
                        # Build stage JSON without escaping (for raw JSON)
                        escaped_stageName=$(echo "$stageName" | sed 's/"/\\"/g')
                        escaped_stageDescription=$(echo "$stageDescription" | sed 's/"/\\"/g')
                        stage_json="{\"StageNumber\":$stage_count,\"StageName\":\"$escaped_stageName\",\"Approvers\":[$stage_approvers_list],\"RequiredCount\":$requiredCount,\"AutoAdvance\":$autoAdvance,\"Description\":\"$escaped_stageDescription\"}"
                        
                        workflow_stages="$workflow_stages,$stage_json"
                        
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
                    
                    # Stage deadline configuration (optional)
                    echo -e "${GREEN}=== Stage Deadline Configuration (Optional) ===${NC}"
                    echo -e "${YELLOW}Set deadlines per stage (comma-separated hours):${NC}"
                    echo -e "${BLUE}Example: "24,48" for 24hrs Stage1, 48hrs Stage2${NC}"
                    echo ""
                    read -p "Stage deadlines (hours, comma-separated, empty for none): " stageDeadlineHours
                    if [[ "$stageDeadlineHours" == "" ]]; then
                        stageDeadlineHours=""
                    fi
                    
                    # Advanced mode arguments: validDecisionsJSON, approversJSON, workflowJSON, deadlineHours, stageDeadlineHours
                    # For the workflow parameter, we need to pass the JSON as-is, not as a quoted string
                    args+=("$id" "$title" "$description" "$hash" "$uploader" "$decisions" "" "$workflow_stages" "" "$stageDeadlineHours")
                    
                else
                    # SIMPLE MODE: Single-stage workflow
                    echo -e "${GREEN}=== Simple Workflow Configuration ===${NC}"
                    echo -e "${BLUE}Single-stage approval process${NC}"
                    
                    # Optional deadline configuration
                    echo ""
                    echo -e "${BLUE}=== Optional Deadline Configuration ===${NC}"
                    read -p "Set deadline? Hours from now (empty for no deadline): " deadlineHours
                    if [[ "$deadlineHours" == "" ]]; then
                        deadlineHours=""
                    fi
                    
                    # Simple mode arguments: validDecisionsJSON, approversJSON, workflowJSON, deadlineHours, stageDeadlineHours
                    args+=("$id" "$title" "$description" "$hash" "$uploader" "$decisions" "$stage1_approvers_json" "" "$deadlineHours" "")
                fi
                
                # Final confirmation
                echo ""
                echo -e "${GREEN}=== Configuration Summary ===${NC}"
                echo -e "${BLUE}Document:${NC} $id ($title)"
                echo -e "${BLUE}Approvers:${NC} $(echo "$stage1_approvers_json" | tr -d '[]' | sed 's/,/, /g')"
                if [[ "$add_more_stages" == "y" || "$add_more_stages" == "Y" ]]; then
                    echo -e "${BLUE}Mode:${NC} Multi-stage workflow ($stage_count stages)"
                else
                    echo -e "${BLUE}Mode:${NC} Simple single-stage workflow"
                fi
                echo ""
                
                while true; do
                    read -p "Proceed with this configuration? [y/n]: " confirm
                    case $confirm in
                        [Yy]* ) break;;
                        [Nn]* ) 
                            echo -e "${YELLOW}Configuration cancelled. Returning to menu...${NC}"
                            continue 2;;
                        * ) echo -e "${RED}Please answer yes (y) or no (n).${NC}";;
                    esac
                done
                ;;
                
            "SubmitNewVersion")
                read -p "Enter document ID: " id
                read -p "Enter document hash: " hash
                read -p "Enter submitter identity: " submitter
                resetApprovals=$(collect_boolean_input "Reset existing approvals?")
                args+=("$id" "$hash" "$submitter" "$resetApprovals")
                ;;
                
            "ApproveDocument")
                echo -e "${GREEN}=== Unified Document Approval ===${NC}"
                echo -e "${YELLOW}🌟 Automatically detects document type (simple or workflow)${NC}"
                echo -e "${YELLOW}Works with both single-stage and multi-stage documents${NC}"
                echo ""
                read -p "Enter document ID: " id
                read -p "Enter approver identity: " approver
                read -p "Enter decision: " decision
                read -p "Enter comment (optional): " comment
                args+=("$id" "$approver" "$decision" "$comment")
                ;;
                
            "QueryDocumentStatus" | "DocumentExists" | "GetDocHistory" | "GetDocumentSummary")
                read -p "Enter document ID: " id
                args+=("$id")
                ;;
                
            "AddEditor")
                read -p "Enter document ID: " id
                read -p "Enter new editor identity: " newEditor
                read -p "Enter invoker identity: " invokerId
                args+=("$id" "$newEditor" "$invokerId")
                ;;
                
            "RemoveEditor")
                read -p "Enter document ID: " id
                read -p "Enter editor to remove: " editorToRemove
                read -p "Enter invoker identity: " invokerId
                args+=("$id" "$editorToRemove" "$invokerId")
                ;;
                
            "AddPrivilegedEditor")
                read -p "Enter document ID: " id
                read -p "Enter new privileged editor identity: " newPrivilegedEditor
                read -p "Enter invoker identity: " invokerId
                args+=("$id" "$newPrivilegedEditor" "$invokerId")
                ;;
                
            "RemovePrivilegedEditor")
                read -p "Enter document ID: " id
                read -p "Enter privileged editor to remove: " privilegedEditorToRemove
                read -p "Enter invoker identity: " invokerId
                args+=("$id" "$privilegedEditorToRemove" "$invokerId")
                ;;
                
            "UpdateDocumentApprovers")
                read -p "Enter document ID: " documentID
                newApprovers=$(collect_array_input "Enter new approvers list:")
                read -p "Enter invoker identity: " invokerId
                args+=("$documentID" "$newApprovers" "$invokerId")
                ;;
                
            "UpdateValidDecisions")
                read -p "Enter document ID: " documentID
                newDecisions=$(collect_array_input "Enter new valid decisions:")
                read -p "Enter invoker identity: " invokerId
                args+=("$documentID" "$newDecisions" "$invokerId")
                ;;
                
            "GetDocumentIdByHash")
                read -p "Enter document hash: " hash
                args+=("$hash")
                ;;
                
            "GetDocumentsByEditor" | "GetDocumentsByApprover" | "GetPendingApprovals")
                read -p "Enter user identity: " userId
                args+=("$userId")
                ;;
                
            "GetAllDocuments")
                echo -e "${BLUE}No parameters required for GetAllDocuments${NC}"
                ;;
                
            "GetDocumentsByDateRange")
                read -p "Enter start date (YYYY-MM-DD): " startDate
                read -p "Enter end date (YYYY-MM-DD): " endDate
                args+=("$startDate" "$endDate")
                ;;
                
            "GetRecentDocuments")
                read -p "Enter limit (number): " limit
                args+=("$limit")
                ;;
                
            "GetDocumentsByUploaderSorted")
                read -p "Enter uploader identity: " uploader
                read -p "Enter sort order (asc/desc, default desc): " sortOrder
                if [[ "$sortOrder" == "" ]]; then
                    sortOrder="desc"
                fi
                args+=("$uploader" "$sortOrder")
                ;;
                
            "GetDocumentsWithPendingApprovals")
                echo -e "${BLUE}No parameters required for GetDocumentsWithPendingApprovals${NC}"
                ;;
                
            "GetDocumentStats")
                echo -e "${BLUE}No parameters required for GetDocumentStats${NC}"
                ;;
                
            "ConfigureDocumentWorkflow")
                echo -e "${PURPLE}=== Advanced Workflow Configuration ===${NC}"
                echo -e "${YELLOW}Note: All documents automatically get a single-stage workflow.${NC}"
                echo -e "${YELLOW}Use this function to create complex multi-stage workflows.${NC}"
                echo ""
                read -p "Enter document ID: " documentID
                read -p "Enter invoker identity: " invokerId
                
                echo -e "${YELLOW}Now let's create workflow stages (minimum 1 stage):${NC}"
                stages_json="["
                stage_count=0
                
                while true; do
                    stage_count=$((stage_count + 1))
                    echo -e "${BLUE}--- Stage $stage_count ---${NC}"
                    read -p "Enter stage name: " stageName
                    echo "Enter approvers for this stage (press Enter on empty line when done):"
                    stage_approvers=""
                    approver_count=0
                    while true; do
                        approver_count=$((approver_count + 1))
                        read -p "Approver $approver_count: " approver
                        if [[ "$approver" == "" ]]; then
                            break
                        fi
                        if [[ "$stage_approvers" == "" ]]; then
                            stage_approvers="\"$approver\""
                        else
                            stage_approvers="$stage_approvers,\"$approver\""
                        fi
                    done
                    
                    read -p "Required approval count (default: ALL): " requiredCount
                    if [[ "$requiredCount" == "" || "$requiredCount" == "ALL" ]]; then
                        requiredCount=0
                    fi
                    
                    read -p "Auto-advance to next stage? (y/n, default: y): " autoAdvance
                    if [[ "$autoAdvance" == "n" || "$autoAdvance" == "no" ]]; then
                        autoAdvance="false"
                    else
                        autoAdvance="true"
                    fi
                    
                    # Build stage JSON with correct field names and required fields
                    stage_json="{\"StageNumber\":$stage_count,\"StageName\":\"$stageName\",\"Approvers\":[$stage_approvers],\"RequiredCount\":$requiredCount,\"AutoAdvance\":$autoAdvance,\"ParallelStages\":[],\"Description\":\"\"}"
                    
                    if [[ "$stage_count" == "1" ]]; then
                        stages_json="$stages_json$stage_json"
                    else
                        stages_json="$stages_json,$stage_json"
                    fi
                    
                    read -p "Add another stage? (y/n): " addAnother
                    if [[ "$addAnother" != "y" && "$addAnother" != "yes" ]]; then
                        break
                    fi
                done
                
                stages_json="$stages_json]"
                args+=("$documentID" "$invokerId" "$stages_json")
                ;;
                
            "GetWorkflowStatus")
                read -p "Enter document ID: " documentID
                args+=("$documentID")
                ;;
                
            "ReturnDocumentToStage")
                echo -e "${PURPLE}=== Return Document to Previous Stage ===${NC}"
                read -p "Enter document ID: " documentID
                read -p "Enter invoker identity: " invokerId
                read -p "Enter target stage index (0-based): " targetStage
                read -p "Enter reason for rollback: " reason
                args+=("$documentID" "$invokerId" "$targetStage" "$reason")
                ;;
                
            "GetDocumentWorkflowHistory")
                echo -e "${GREEN}🆕 NEW: Get Complete Workflow History Across All Versions${NC}"
                echo -e "${YELLOW}Shows workflow progression for all versions of a document${NC}"
                read -p "Enter document ID: " documentID
                args+=("$documentID")
                ;;
                
            "GetVersionWorkflowState")
                echo -e "${GREEN}🆕 NEW: Get Workflow State for Specific Version${NC}"
                echo -e "${YELLOW}Shows detailed workflow state for a specific document version${NC}"
                read -p "Enter document ID: " documentID
                read -p "Enter version number: " version
                args+=("$documentID" "$version")
                ;;
                
            "SetDocumentDeadline")
                echo -e "${PURPLE}=== Set Document Deadline ===${NC}"
                echo -e "${YELLOW}Set deadline for document approval (embedded system)${NC}"
                read -p "Enter document ID: " documentID
                read -p "Enter invoker identity: " invokerId  
                read -p "Deadline hours from now: " deadlineHours
                args+=("$documentID" "$invokerId" "$deadlineHours")
                ;;
                
            "GetDeadlineStatus")
                echo -e "${BLUE}=== Get Deadline Status ===${NC}"
                read -p "Enter document ID: " documentID
                args+=("$documentID")
                ;;
                
            "GetDocumentsWithUpcomingDeadlines")
                echo -e "${BLUE}=== Get Documents With Upcoming Deadlines ===${NC}"
                read -p "Hours ahead to check (default 24): " hoursAhead
                if [[ "$hoursAhead" == "" ]]; then
                    hoursAhead="24"
                fi
                args+=("$hoursAhead")
                ;;
                
            "RemoveDocumentDeadline")
                echo -e "${PURPLE}=== Remove Document Deadline ===${NC}"
                read -p "Enter document ID: " documentID
                read -p "Enter invoker identity: " invokerId
                args+=("$documentID" "$invokerId")
                ;;
        esac

        # Convert args array to JSON string
        if [ ${#args[@]} -eq 0 ]; then
            args_json="[]"
        else
            # Special handling for functions with JSON parameters
            if [[ "$func_name" == "ConfigureDocumentWorkflow" ]]; then
                # For ConfigureDocumentWorkflow: [documentID, invokerId, stages_json]
                # The third parameter (stages_json) needs to be escaped as a string
                escaped_stages=$(echo "${args[2]}" | sed 's/"/\\"/g')
                args_json="[\"${args[0]}\",\"${args[1]}\",\"$escaped_stages\"]"
            elif [[ "$func_name" == "SubmitDocument" && ${#args[@]} -eq 10 && "${args[7]}" != "" ]]; then
                # For SubmitDocument with workflow JSON (advanced mode)
                # args[7] contains the workflow JSON that should be escaped as a string parameter
                escaped_workflow=$(echo "${args[7]}" | sed 's/"/\\"/g')
                args_json="[\"${args[0]}\",\"${args[1]}\",\"${args[2]}\",\"${args[3]}\",\"${args[4]}\",\"${args[5]}\",\"${args[6]}\",\"$escaped_workflow\",\"${args[8]}\",\"${args[9]}\"]"
            else
                # Regular parameter handling - quote all parameters
                args_json=$(printf '"%s",' "${args[@]}")
                args_json="[${args_json%,}]"
            fi
        fi

        # Build chaincode command based on function type
        if is_invoke_function "$func_name"; then
            # Invoke command for state-changing operations
            command="peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile \"\${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem\" \\
-C mychannel -n documentApproval \\
--peerAddresses localhost:7051 --tlsRootCertFiles \"\${PWD}/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:8051 --tlsRootCertFiles \"\${PWD}/organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:9051 --tlsRootCertFiles \"\${PWD}/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:10051 --tlsRootCertFiles \"\${PWD}/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:11051 --tlsRootCertFiles \"\${PWD}/organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
-c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
        else
            # Query command for read-only operations
            if is_string_function "$func_name"; then
                # Don't use jq for functions that return plain strings
                command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
            else
                # Use jq for functions that return JSON
                command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}' | jq ."
            fi
        fi

        # Execute the command
        echo ""
        execute_command "$command"
    else
        echo -e "${RED}Function '$func_name' not found.${NC}"
        echo -e "${GREEN}Press Enter to continue...${NC}"
        read
    fi
done