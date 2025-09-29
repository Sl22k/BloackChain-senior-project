# Test Script Sharing Guide

## Files to Share with Your Partner

To successfully run `test-launcher.sh`, your partner needs these files and folders:

### 📁 **Core Script Files** (Required)
```
test-scripts/
├── test-launcher.sh                    # Main entry point
├── shared/
│   ├── common-functions.sh            # Core chaincode interaction functions
│   └── common-functions-v2.sh         # Additional utility functions
└── modules/
    ├── document-lifecycle.sh           # Document creation, submission, approval
    ├── queries-versioning.sh           # Document queries and version history
    ├── ownership-transfer.sh           # Document ownership management
    ├── workflow-management.sh          # Workflow configuration and rollback
    ├── administration.sh               # Document administration tools
    ├── deadline-management.sh          # Deadline configuration and monitoring
    └── advanced-queries.sh             # Advanced analytics and batch operations
```

### 🔧 **External Dependencies** (Must be installed/configured)

#### 1. **Hyperledger Fabric Binaries**
- Location: `../../bin/` (relative to test-scripts directory)
- Required files:
  - `peer` (Fabric peer CLI)
  - Other Fabric binaries if used

#### 2. **Fabric Configuration**
- Location: `../../config/` (relative to test-scripts directory)
- Contains Fabric client configuration files

#### 3. **TLS Certificates & MSP**
- Organization certificates and MSP directories
- Required for peer authentication

#### 4. **System Dependencies**
- `jq` - JSON processing tool (install: `sudo apt-get install jq`)
- `bash` - Bash shell (version 4.0+)
- Standard Unix tools: `grep`, `sed`, `awk`, `head`, `tail`

### 🔧 **What Your Partner Needs to Modify**

#### 1. **Path Configuration in test-launcher.sh**
Your partner needs to update these paths (lines 60-139):

```bash
# Current paths (your system)
export PATH=${PWD}/../../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../../config/

# TLS Certificate paths (update for their system)
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/../organizations/peerOrganizations/is.example.com/tlsca/tlsca.is.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=${PWD}/../organizations/peerOrganizations/is.example.com/users/Admin@is.example.com/msp
```

#### 2. **Network Configuration**
- Peer addresses (currently: localhost:7051, 8051, 9051, 10051, 11051)
- Orderer address (currently: localhost:7050)
- Channel name (currently: mychannel)
- Chaincode name (currently: documentApproval)

#### 3. **Certificate Paths**
Update paths to match their Fabric network structure:
- Orderer TLS CA certificates
- Peer TLS CA certificates
- MSP configuration paths
- User certificate paths

### 📋 **Setup Instructions for Your Partner**

#### Step 1: Install Dependencies
```bash
# Install jq for JSON processing
sudo apt-get update && sudo apt-get install jq

# Verify other tools are available
which bash grep sed awk head tail
```

#### Step 2: Setup Hyperledger Fabric
```bash
# Download Fabric binaries (if not already done)
curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.4 1.5.5

# Or copy existing binaries to the expected location
```

#### Step 3: Configure Paths
1. Edit `test-launcher.sh`
2. Update all hardcoded paths to match their environment
3. Update network addresses and ports
4. Update organization and certificate paths

#### Step 4: Verify Network Access
```bash
# Test peer connectivity
peer version

# Test chaincode query (example)
peer chaincode query -C mychannel -n documentApproval -c '{"Function":"GetAllDocuments","Args":[]}'
```

### 🚨 **Critical Path Updates Required**

Your partner must search and replace these paths throughout the scripts:

#### Replace Your Absolute Paths:
```bash
# Find all occurrences of your specific paths
grep -r "/home/yousif-ubunto/my-fabric-project" test-scripts/

# Replace with their paths or use relative paths
```

#### Key Paths to Update:
1. **Fabric binaries**: `../../bin/`
2. **Fabric config**: `../../config/`
3. **TLS certificates**: `../organizations/`
4. **MSP paths**: Update organization-specific paths
5. **Network addresses**: Update if using different ports/hosts

### 📝 **Environment Variables**

Your partner should set these environment variables or modify the script defaults:

```bash
# Fabric paths
export PATH=/path/to/fabric/bin:$PATH
export FABRIC_CFG_PATH=/path/to/fabric/config

# Peer configuration
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=YourOrgMSP
export CORE_PEER_ADDRESS=localhost:7051

# Network specific
export CHANNEL_NAME=mychannel
export CHAINCODE_NAME=documentApproval
```

### ✅ **Testing After Setup**

1. **Test Script Launch**:
   ```bash
   cd /path/to/network/test-scripts
   ./test-launcher.sh
   ```

2. **Test Peer Selection**: Choose a peer and verify environment setup

3. **Test Basic Query**: Try a simple chaincode query to verify connectivity

4. **Test Module Loading**: Navigate through different menu options

### 📞 **Support & Troubleshooting**

Common issues and solutions:

1. **"peer: command not found"** → Check PATH to Fabric binaries
2. **"Certificate not found"** → Update TLS certificate paths
3. **"Connection refused"** → Check network addresses and ports
4. **"MSP not found"** → Update MSP configuration paths
5. **"jq: command not found"** → Install jq package

### 📦 **Minimal Sharing Package**

For easier sharing, create a package:

```bash
# Create sharing package
tar -czf document-approval-test-scripts.tar.gz \
  test-scripts/ \
  organizations/ \
  bin/ \
  config/ \
  SHARING-GUIDE.md

# Your partner can extract and follow this guide
```

---

**Note**: This test suite is designed for the custom 5-peer Hyperledger Fabric network with document approval chaincode. Ensure your partner has a compatible network setup or help them adapt the scripts to their network configuration.