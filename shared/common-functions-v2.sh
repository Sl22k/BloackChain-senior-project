#!/bin/bash

# =============================================================================
# PHASE 6 NATIVE COMMON FUNCTIONS
# =============================================================================
# Clean, direct implementation that matches chaincode exactly
# No legacy conversions, no parameter juggling - just pure Phase 6
# =============================================================================

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# =============================================================================
# CHAINCODE COMMAND EXECUTION - SIMPLIFIED
# =============================================================================

# Execute chaincode function with direct parameter mapping
execute_chaincode_function() {
    local func_name="$1"
    shift
    local args=("$@")

    # Build JSON args - no conversions, direct mapping
    local args_json=""
    if [ ${#args[@]} -eq 0 ]; then
        args_json="[]"
    else
        # Quote all arguments as strings, properly escaping JSON
        local json_args=()
        for arg in "${args[@]}"; do
            # Escape quotes and backslashes in the argument
            local escaped_arg=$(echo "$arg" | sed 's/\\/\\\\/g; s/"/\\"/g')
            json_args+=("\"$escaped_arg\"")
        done

        # Join with commas
        args_json="["
        for i in "${!json_args[@]}"; do
            if [ $i -gt 0 ]; then
                args_json+=","
            fi
            args_json+="${json_args[$i]}"
        done
        args_json+="]"
    fi

    # Determine if function needs invoke or query
    local command=""
    if is_invoke_function "$func_name"; then
        command="peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile \"\${PWD}/../organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem\" \\
-C mychannel -n documentApproval \\
--peerAddresses localhost:7051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:8051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem\" \\
--peerAddresses localhost:9051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:10051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
--peerAddresses localhost:11051 --tlsRootCertFiles \"\${PWD}/../organizations/peerOrganizations/cs.example.com/tlsca/tlsca.cs.example.com-cert.pem\" \\
-c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
    else
        command="peer chaincode query -C mychannel -n documentApproval -c '{\"Function\":\"$func_name\",\"Args\":$args_json}'"
    fi

    # Execute and format output
    execute_command_with_formatting "$func_name" "$command"
}

# Execute command with enhanced formatting
execute_command_with_formatting() {
    local func_name="$1"
    local command="$2"

    echo -e "${GREEN}📡 Executing $func_name${NC}"
    echo -e "${PURPLE}$command${NC}"
    echo ""

    local output
    output=$(eval "$command" 2>&1)
    local exit_code=$?

    echo -e "${GREEN}=== 📋 Response ===${NC}"
    if [ $exit_code -eq 0 ]; then
        # Try to format as JSON if possible
        if [[ "$output" =~ ^\s*[\{\[] ]]; then
            echo "$output" | jq . 2>/dev/null || echo "$output"
        else
            echo "$output"
        fi
    else
        echo -e "${RED}❌ Error (exit code: $exit_code)${NC}"
        echo -e "${RED}$output${NC}"
    fi

    echo ""
    echo -e "${GREEN}Press Enter to continue...${NC}"
    read
}

# =============================================================================
# FUNCTION TYPE DETECTION
# =============================================================================

# State-changing functions that require invoke
INVOKE_FUNCTIONS=(
    "InitLedger" "SubmitDocument" "ApproveDocument" "SubmitNewVersion"
    "ConfigureDocumentWorkflow" "ReturnDocumentToStage" "AdvanceDocumentToNextStage"
    "AddEditor" "RemoveEditor" "AddPrivilegedEditor" "RemovePrivilegedEditor"
    "UpdateDocumentApprovers" "UpdateValidDecisions"
    "RequestOwnershipTransfer" "AcceptOwnershipTransfer" "RejectOwnershipTransfer" "CancelOwnershipTransfer"
    "SetDocumentDeadline" "RemoveDocumentDeadline"
)

is_invoke_function() {
    local func_name="$1"
    for invoke_func in "${INVOKE_FUNCTIONS[@]}"; do
        [[ "$invoke_func" == "$func_name" ]] && return 0
    done
    return 1
}

# =============================================================================
# PHASE 6 NATIVE INPUT BUILDERS
# =============================================================================

# Build StageInput JSON for SubmitDocument
build_stage_input() {
    local name="$1"
    local description="$2"
    local approvers="$3"        # JSON array string like ["user1","user2"]
    local required_count="$4"   # Number or 0 for ALL
    local auto_advance="$5"     # true/false

    # Escape JSON strings properly
    local escaped_name=$(echo "$name" | sed 's/"/\\"/g')
    local escaped_desc=$(echo "$description" | sed 's/"/\\"/g')

    # Convert approvers to clean JSON array
    local approvers_clean
    if [[ "$approvers" =~ ^\[.*\]$ ]]; then
        approvers_clean="$approvers"
    else
        # Convert comma-separated to JSON array
        approvers_clean="[\"$(echo "$approvers" | sed 's/,/","/g')\"]"
    fi

    # Build StageInput with exact field names
    echo "{\"name\":\"$escaped_name\",\"description\":\"$escaped_desc\",\"approvers\":$approvers_clean,\"requiredApprovals\":$required_count,\"autoAdvance\":$auto_advance}"
}

# Build NewVersionState JSON for SubmitNewVersion
build_new_version_state() {
    local content_hash="$1"
    local approvers_list="$2"      # JSON array string
    local valid_decisions="$3"     # JSON array string
    local reset_approvals="$4"     # true/false
    local change_reason="$5"

    echo "{\"contentHash\":\"$content_hash\",\"approversList\":$approvers_list,\"validDecisions\":$valid_decisions,\"resetApprovals\":$reset_approvals,\"changeReason\":\"$change_reason\"}"
}

# =============================================================================
# SIMPLIFIED INPUT COLLECTION
# =============================================================================

# Collect yes/no input
get_boolean() {
    local prompt="$1"
    while true; do
        read -p "$prompt (y/n): " input
        case "$input" in
            [Yy]*) echo "true"; break;;
            [Nn]*) echo "false"; break;;
            *) echo -e "${RED}Please answer y or n${NC}" >&2;;
        esac
    done
}

# Collect JSON array from comma-separated input
get_array() {
    local prompt="$1"
    echo -e "${BLUE}$prompt${NC}" >&2
    echo -e "${YELLOW}(comma-separated, e.g: user1,user2,user3)${NC}" >&2
    read -p "> " input
    if [ -z "$input" ]; then
        echo "[]"
    else
        echo "[\"$(echo "$input" | tr -d ' ' | sed 's/,/","/g')\"]"
    fi
}

# Quick preset approver groups
get_approver_group() {
    echo -e "${GREEN}=== Select Approver Group ===${NC}" >&2
    echo -e "${YELLOW}1${NC} - Legal Team" >&2
    echo -e "${YELLOW}2${NC} - Technical Team" >&2
    echo -e "${YELLOW}3${NC} - Executive Team" >&2
    echo -e "${YELLOW}4${NC} - Custom (enter your own)" >&2
    echo "" >&2

    while true; do
        read -p "Choice [1-4]: " choice
        case $choice in
            1) echo "[\"legal1\",\"legal2\",\"legal3\"]"; break;;
            2) echo "[\"dev1\",\"dev2\",\"architect1\"]"; break;;
            3) echo "[\"ceo\",\"cfo\",\"cto\"]"; break;;
            4) get_array "Enter approvers"; break;;
            *) echo -e "${RED}Invalid choice 1-4${NC}" >&2;;
        esac
    done
}

# Quick preset decision sets
get_decision_set() {
    echo -e "${GREEN}=== Select Decision Set ===${NC}" >&2
    echo -e "${YELLOW}1${NC} - Standard (APPROVED, REJECTED)" >&2
    echo -e "${YELLOW}2${NC} - Extended (APPROVED, REJECTED, NEEDS_REVISION)" >&2
    echo -e "${YELLOW}3${NC} - Custom (enter your own)" >&2
    echo "" >&2

    while true; do
        read -p "Choice [1-3]: " choice
        case $choice in
            1) echo "[\"APPROVED\",\"REJECTED\"]"; break;;
            2) echo "[\"APPROVED\",\"REJECTED\",\"NEEDS_REVISION\"]"; break;;
            3) get_array "Enter decisions"; break;;
            *) echo -e "${RED}Invalid choice 1-3${NC}" >&2;;
        esac
    done
}

# =============================================================================
# MODULE RETURN
# =============================================================================

return_to_main_menu() {
    echo -e "${GREEN}Returning to main menu...${NC}"
    echo ""
}