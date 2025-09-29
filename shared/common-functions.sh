#!/bin/bash

# =============================================================================
# SHARED UTILITY FUNCTIONS
# =============================================================================
# Common functions used across all test modules
# Formatting, validation, command construction, etc.
# =============================================================================

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'  # No Color

# =============================================================================
# OUTPUT FORMATTING FUNCTIONS
# =============================================================================

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

# =============================================================================
# INPUT COLLECTION FUNCTIONS
# =============================================================================

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

# =============================================================================
# PREDEFINED DATA SETS FOR QUICK TESTING
# =============================================================================

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
            1) echo '["APPROVED","REJECTED"]'; break;;
            2) echo '["APPROVED","REJECTED","NEEDS_REVISION"]'; break;;
            3) echo '["APPROVED","REJECTED","NEEDS_REVISION","NEEDS_INFO"]'; break;;
            4) echo '["APPROVE","REJECT"]'; break;;
            5)
                echo -e "${BLUE}Enter custom decisions:${NC}" >&2
                echo -e "${YELLOW}(Enter comma-separated values, e.g., APPROVED,REJECTED,PENDING)${NC}" >&2
                read -p "> " custom_input
                if [ ! -z "$custom_input" ]; then
                    formatted_input=$(echo "$custom_input" | tr -d ' ' | sed 's/,/","/g; s/^/["/; s/$/"]/')
                    echo "$formatted_input"
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
            1) echo '["legal1","legal2","legal3"]'; break;;
            2) echo '["dev1","dev2","architect1"]'; break;;
            3) echo '["ceo","cfo","cto"]'; break;;
            4) echo '["finance1","finance2","controller"]'; break;;
            5) echo '["legal1","dev1","finance1"]'; break;;
            6)
                echo -e "${BLUE}Enter single approver username:${NC}" >&2
                read -p "> " single_user
                if [ ! -z "$single_user" ]; then
                    echo "[\"$single_user\"]"
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
                    echo "$formatted_input"
                    break
                else
                    echo -e "${RED}Please enter at least one approver.${NC}" >&2
                fi
                ;;
            *) echo -e "${RED}Invalid choice. Please select 1-7.${NC}" >&2;;
        esac
    done
}

# =============================================================================
# CHAINCODE COMMAND CONSTRUCTION
# =============================================================================

# Functions that require invoke (state-changing operations)
INVOKE_FUNCTIONS=(
    # Core Lifecycle (state-changing)
    "InitLedger"
    "SubmitDocument"
    "ApproveDocument"
    "SubmitNewVersion"

    # Workflow Management (state-changing)
    "ConfigureDocumentWorkflow"
    "AdvanceDocumentToNextStage"
    "ReturnDocumentToStage"

    # Administration (state-changing)
    "AddEditor"
    "RemoveEditor"
    "AddPrivilegedEditor"
    "RemovePrivilegedEditor"
    "UpdateDocumentApprovers"
    "UpdateValidDecisions"

    # Ownership Transfer (state-changing)
    "RequestOwnershipTransfer"
    "AcceptOwnershipTransfer"
    "RejectOwnershipTransfer"
    "CancelOwnershipTransfer"

    # Deadline Management (state-changing)
    "SetStageDeadlines"
    "RemoveStageDeadlines"
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

# Function to build and execute chaincode commands
execute_chaincode_function() {
    local func_name="$1"
    shift
    local args=("$@")

    # Convert args array to JSON string
    if [ ${#args[@]} -eq 0 ]; then
        args_json="[]"
    else
        # Special handling for functions with JSON parameters
        if [[ "$func_name" == "ConfigureDocumentWorkflow" ]]; then
            # For ConfigureDocumentWorkflow: [documentID, invokerId, configurationJSON]
            # The third parameter (configurationJSON) needs to be escaped as a string
            escaped_config=$(echo "${args[2]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"${args[1]}\",\"$escaped_config\"]"
        elif [[ "$func_name" == "SetStageDeadlines" ]]; then
            # For SetStageDeadlines: [documentID, stageDeadlinesJSON, invokerId]
            # The second parameter (stageDeadlinesJSON) needs to be escaped as a string
            escaped_deadlines=$(echo "${args[1]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"$escaped_deadlines\",\"${args[2]}\"]"
        elif [[ "$func_name" == "RemoveStageDeadlines" ]]; then
            # For RemoveStageDeadlines: [documentID, stageNumbersJSON, invokerId]
            # The second parameter (stageNumbersJSON) needs to be escaped as a string
            escaped_stages=$(echo "${args[1]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"$escaped_stages\",\"${args[2]}\"]"
        elif [[ "$func_name" == "ReturnDocumentToStage" ]]; then
            # For ReturnDocumentToStage: [documentID, invokerId, targetStage(string), comment, deadlineOptionsJSON]
            # targetStage must be passed as string for JSON Args compatibility
            # The fifth parameter (deadlineOptionsJSON) might be empty or JSON, needs to be escaped as string
            if [[ ${#args[@]} -eq 5 && -n "${args[4]}" ]]; then
                escaped_deadline_options=$(echo "${args[4]}" | sed 's/"/\\"/g')
                args_json="[\"${args[0]}\",\"${args[1]}\",\"${args[2]}\",\"${args[3]}\",\"$escaped_deadline_options\"]"
            else
                # Handle case where deadlineOptionsJSON is empty or not provided
                args_json="[\"${args[0]}\",\"${args[1]}\",\"${args[2]}\",\"${args[3]}\",\"\"]"
            fi
        elif [[ "$func_name" == "SubmitDocument" ]]; then
            # Phase 6 SubmitDocument: (id, title, description, hash, uploader, validDecisionsJSON, stagesJSON)
            # Only 7 parameters, stagesJSON format is different
            if [[ ${#args[@]} -eq 7 ]]; then
                # Direct 7-parameter call - convert JSON parameters to escaped strings
                escaped_decisions=$(echo "${args[5]}" | sed 's/"/\\"/g')
                escaped_stages=$(echo "${args[6]}" | sed 's/"/\\"/g')
                args_json="[\"${args[0]}\",\"${args[1]}\",\"${args[2]}\",\"${args[3]}\",\"${args[4]}\",\"$escaped_decisions\",\"$escaped_stages\"]"
            else
                # Legacy 10-parameter conversion - convert to new 7-parameter format
                id="${args[0]}"
                title="${args[1]}"
                description="${args[2]}"
                hash="${args[3]}"
                uploader="${args[4]}"
                validDecisions="${args[5]}"
                # Convert the old approver/workflow format to new stagesJSON format
                if [[ "${args[7]}" != "" ]]; then
                    # Advanced mode: use the complex workflow stages
                    stagesJSON="${args[7]}"
                elif [[ "${args[6]}" != "" ]]; then
                    # Simple mode: single stage from approvers
                    approvers_clean=$(echo "${args[6]}" | sed 's/\\\"/"/g')
                    stagesJSON="[{\"name\":\"Document Approval\",\"description\":\"\",\"approvers\":$approvers_clean,\"requiredApprovals\":0,\"autoAdvance\":true}]"
                else
                    # Fallback: default single stage
                    stagesJSON="[{\"name\":\"Document Approval\",\"description\":\"\",\"approvers\":[],\"requiredApprovals\":0,\"autoAdvance\":true}]"
                fi
                escaped_stages=$(echo "$stagesJSON" | sed 's/"/\\"/g')
                args_json="[\"$id\",\"$title\",\"$description\",\"$hash\",\"$uploader\",\"$validDecisions\",\"$escaped_stages\"]"
            fi
        elif [[ "$func_name" == "SubmitNewVersion" && ${#args[@]} -eq 3 ]]; then
            # SubmitNewVersion takes (documentID, submitter, newVersionStateJSON)
            # The third parameter needs to be escaped as a JSON string
            escaped_new_state=$(echo "${args[2]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"${args[1]}\",\"$escaped_new_state\"]"
        elif [[ "$func_name" == "SubmitNewVersion" && ${#args[@]} -eq 2 ]]; then
            # Legacy SubmitNewVersion takes (documentID, newStateJSON)
            # The second parameter needs to be escaped as a JSON string
            escaped_new_state=$(echo "${args[1]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"$escaped_new_state\"]"
        elif [[ "$func_name" == "UpdateValidDecisions" ]]; then
            # UpdateValidDecisions takes (documentID, newDecisionsJSON, invokerId)
            # The second parameter (newDecisionsJSON) needs to be escaped as a JSON string
            escaped_decisions=$(echo "${args[1]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"$escaped_decisions\",\"${args[2]}\"]"
        elif [[ "$func_name" == "UpdateDocumentApprovers" ]]; then
            # UpdateDocumentApprovers takes (documentID, stageUpdatesJSON, invokerId, resetApprovals)
            # The second parameter (stageUpdatesJSON) needs to be escaped as a JSON string
            # The fourth parameter (resetApprovals) needs to be quoted as a string
            escaped_stage_updates=$(echo "${args[1]}" | sed 's/"/\\"/g')
            args_json="[\"${args[0]}\",\"$escaped_stage_updates\",\"${args[2]}\",\"${args[3]}\"]"
        else
            # Regular parameter handling - quote all parameters
            args_json=$(printf '"%s",' "${args[@]}")
            args_json="[${args_json%,}]"
        fi
    fi

    # Build chaincode command based on function type
    if is_invoke_function "$func_name"; then
        # Invoke command for state-changing operations
        command="peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile \"\${PWD}/../organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem\" \\
-C mychannel -n documentApproval \\
--peerAddresses localhost:7051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:8051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:9051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:10051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:11051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
-c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
    else
        # Query command for read-only operations
        if is_string_function "$func_name"; then
            # Don't use jq for functions that return plain strings
            command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
        else
            # Use jq for functions that return JSON - don't apply formatting here, let format_json_output handle it
            command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
        fi
    fi

    # Execute the command
    execute_command "$command"
}

# Function to execute chaincode queries (read-only, returns result without user interaction)
execute_chaincode_query() {
    local func_name="$1"
    shift
    local args=("$@")

    # Convert args array to JSON string
    if [ ${#args[@]} -eq 0 ]; then
        args_json="[]"
    else
        # Regular parameter handling - quote all parameters
        args_json=$(printf '"%s",' "${args[@]}")
        args_json="[${args_json%,}]"
    fi

    # Build query command
    local command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"

    # Execute and return result (no user interaction)
    eval "$command" 2>/dev/null
}

# =============================================================================
# MODULE RETURN FUNCTIONS
# =============================================================================

# Function to return to main menu
return_to_main_menu() {
    echo -e "${GREEN}Returning to main menu...${NC}"
    echo ""
}