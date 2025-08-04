package main

import (
    "bytes"
    // "context"
    "crypto/x509"
    "encoding/json"
    // "errors"
    "flag"
    "fmt"
    "os"
    "path"
    "time"

    "github.com/hyperledger/fabric-gateway/pkg/client"
    "github.com/hyperledger/fabric-gateway/pkg/hash"
    "github.com/hyperledger/fabric-gateway/pkg/identity"
    // "github.com/hyperledger/fabric-protos-go-apiv2/gateway"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    // "google.golang.org/grpc/status"
)

const (
    mspID        = "ISMSP"
    cryptoPath   = "../../network/organizations/peerOrganizations/is.example.com"
    certPath     = cryptoPath + "/users/User1@is.example.com/msp/signcerts"
    keyPath      = cryptoPath + "/users/User1@is.example.com/msp/keystore"
    tlsCertPath  = cryptoPath + "/tlsca/tlsca.is.example.com-cert.pem"
    peerEndpoint = "dns:///localhost:7051"
    gatewayPeer  = "peer0.is.example.com"
)

func main() {
    fmt.Println("Application started!")
    action := flag.String("action", "", "submit, decide, status, all")
    id := flag.String("id", "", "document ID")
    hashVal := flag.String("hash", "", "document hash (for submit)")
    approver := flag.String("approver", "", "approver email (for decide)")
    approvers := flag.String("approvers", "", "comma-separated approvers (for submit)")
    decision := flag.String("decision", "", "APPROVE or REJECT (for decide)")
    flag.Parse()

    if *action == "" || (*action != "all" && *id == "") {
        fmt.Println("Usage: go run main.go -action=<submit|decide|status|all> -id=<docId> [other flags]")
        os.Exit(1)
    }

    conn := newGrpcConnection()
    defer conn.Close()

    idObj := newIdentity()
    sign := newSign()
    gw, err := client.Connect(
        idObj,
        client.WithSign(sign),
        client.WithHash(hash.SHA256),
        client.WithClientConnection(conn),
        client.WithEvaluateTimeout(5*time.Second),
        client.WithEndorseTimeout(15*time.Second),
        client.WithSubmitTimeout(5*time.Second),
        client.WithCommitStatusTimeout(1*time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer gw.Close()

    cc := os.Getenv("CHAINCODE_NAME")
    if cc == "" {
        cc = "documentApproval"
    }
    ch := os.Getenv("CHANNEL_NAME")
    if ch == "" {
        ch = "mychannel"
    }

    network := gw.GetNetwork(ch)
    contract := network.GetContract(cc)

    switch *action {
    case "submit":
        if *hashVal == "" || *approvers == "" {
            panic("submit requires -hash and -approvers")
        }
        approversJson, err := json.Marshal(splitAndTrim(*approvers))
        if err != nil {
            panic(err)
        }
        _, err = contract.SubmitTransaction("SubmitDocument", *id, *hashVal, mspID+"::"+getClientName(idObj), string(approversJson))
        checkErr("SubmitDocument", err)

    case "decide":
        if *approver == "" || *decision == "" {
            panic("decide requires -approver and -decision")
        }
        _, err = contract.SubmitTransaction("ApproveDocument", *id, *approver, *decision)
        checkErr("ApproveDocument", err)

    case "status":
        result, err := contract.EvaluateTransaction("QueryDocumentStatus", *id)
        checkErr("QueryDocumentStatus", err)
        fmt.Println(formatJSON(result))

    case "all":
        result, err := contract.EvaluateTransaction("GetAllDocuments")
        checkErr("GetAllDocuments", err)
        fmt.Println(formatJSON(result))

    default:
        panic("Unknown action")
    }
}

func newGrpcConnection() *grpc.ClientConn {
    certPEM, err := os.ReadFile(tlsCertPath)
    if err != nil {
        panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
    }

    cert, err := identity.CertificateFromPEM(certPEM)
    if err != nil {
        panic(err)
    }

    pool := x509.NewCertPool()
    pool.AddCert(cert)
    creds := credentials.NewClientTLSFromCert(pool, gatewayPeer)

    conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(creds))
    if err != nil {
        panic(fmt.Errorf("failed to create gRPC connection: %w", err))
    }

    return conn
}

func newIdentity() *identity.X509Identity {
    pem, err := os.ReadFile(firstFile(certPath))
    if err != nil {
        panic(err)
    }
    cert, err := identity.CertificateFromPEM(pem)
    if err != nil {
        panic(err)
    }
    id, err := identity.NewX509Identity(mspID, cert)
    if err != nil {
        panic(err)
    }
    return id
}

func newSign() identity.Sign {
    pem, err := os.ReadFile(firstFile(keyPath))
    if err != nil {
        panic(err)
    }
    pk, err := identity.PrivateKeyFromPEM(pem)
    if err != nil {
        panic(err)
    }
    sign, err := identity.NewPrivateKeySign(pk)
    if err != nil {
        panic(err)
    }
    return sign
}

func firstFile(dir string) string {
    f, err := os.Open(dir)
    if err != nil {
        panic(err)
    }
    names, err := f.Readdirnames(1)
    if err != nil {
        panic(err)
    }
    return path.Join(dir, names[0])
}

func splitAndTrim(s string) []string {
    var out []string
    for _, p := range bytes.Split([]byte(s), []byte(",")) {
        out = append(out, string(bytes.TrimSpace(p)))
    }
    return out
}

func checkErr(name string, err error) {
    if err != nil {
        fmt.Printf("%s failed: %v\n", name, err)
        os.Exit(1)
    }
    fmt.Printf("%s completed\n", name)
}

func formatJSON(data []byte) string {
    var pretty bytes.Buffer
    if err := json.Indent(&pretty, data, "", "  "); err != nil {
        panic(fmt.Errorf("failed to parse JSON: %w", err))
    }
    return pretty.String()
}

func getClientName(id *identity.X509Identity) string {
    // Simplistic client ID for demo purposes
    return mspID
}