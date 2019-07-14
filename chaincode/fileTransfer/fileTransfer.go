package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("fileTransfer")

// SimpleChaincode example simple Chaincode implementation
type FileTransferChaincode struct {
}

type transfer struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	UUID       string `json:"uuid"`
	Name       string `json:"name"` //the fieldtags are needed to keep case from bouncing around
	FileHash   string `json:"fileHash"`
	Recipient  string `json:"recipient"`
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	logger.Info(" ############ Chaincode main  ############")
	err := shim.Start(new(FileTransferChaincode))
	if err != nil {
		fmt.Printf("Error starting FileTransfer chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *FileTransferChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Info(" ############ Chaincode init ############")
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *FileTransferChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	logger.Info(" ############ Invoke " + function + " ############")
	fmt.Println("invoke is running " + function)

	var result string
	var err error

	// Handle different functions
	if function == "initTransfer" { //create a new transfer
		return t.initTransfer(stub, args)
	} else if function == "delete" { //delete a transfer
		return t.delete(stub, args)
	} else if function == "readTransfer" { //read a transfer
		return t.readTransfer(stub, args)
	} else if function == "queryTransfersByOriginator" { //find transfers for owner X using rich query
		return t.queryTransfersByOriginator(stub, args)
	} else if function == "queryTransfersByRecipient" {
		return t.queryTransfersByRecipient(stub, args)
	} else if function == "queryTransfers" { //find transfers based on an ad hoc rich query
		return t.queryTransfers(stub, args)
	} else if function == "getHistoryForTransfer" { //get history of values for a transfer
		return t.getHistoryForTransfer(stub, args)
	} else if function == "query" {
		result, err = get(stub, args)
		if err != nil {
			return shim.Error(err.Error())
		}
	
		return shim.Success([]byte(result))
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

func get(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("Incorrect arguments, expecting a key.")
	}

	value, err := stub.GetState(args[0])
	if err != nil {
		return "", fmt.Errorf("Failed to get asset %s with error %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", args[0])
	}
	return string(value), nil
}

// ============================================================
// initTransfer - create a new transfer, store into chaincode state
// ============================================================
func (t *FileTransferChaincode) initTransfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	logger.Info("############ initTransfer ############")
	//   0       1           3
	// "alice", "sfdfsdf", "bob"
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init transfer")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}

	id, err := uuid.NewUUID()
	if err != nil {
		// handle error
		return shim.Error("Failed to generate UUID for product")
	}
	uuid := id.String()
	originatorName := strings.ToLower(args[0])
	fileHash := strings.ToLower(args[1])
	recipientName := strings.ToLower(args[2])

	// ==== Check if transfer already exists ====
	transferAsBytes, err := stub.GetState(uuid)
	if err != nil {
		return shim.Error("Failed to get transfer: " + err.Error())
	} else if transferAsBytes != nil {
		fmt.Println("This transfer already exists: " + uuid)
		return shim.Error("This transfer already exists: " + uuid)
	}

	// ==== Create transfer object and marshal to JSON ====
	objectType := "fileTransfer"
	transfer := &transfer{objectType, uuid, originatorName, fileHash, recipientName}
	transferJSONasBytes, err := json.Marshal(transfer)
	if err != nil {
		return shim.Error(err.Error())
	}
	//Alternatively, build the marble json string manually if you don't want to use struct marshalling
	//marbleJSONasString := `{"docType":"Marble",  "name": "` + marbleName + `", "color": "` + color + `", "size": ` + strconv.Itoa(size) + `, "owner": "` + owner + `"}`
	//marbleJSONasBytes := []byte(str)

	// === Save transfer to state ===
	err = stub.PutState(uuid, transferJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  ==== Index the marble to enable originator-based range queries, e.g. return all transfers originated by one person ====
	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~originatorName~fileHash.
	//  This will enable very efficient state range queries based on composite keys matching indexName~color~*
	indexName := "originator~hash"
	originatorHashIndexKey, err := stub.CreateCompositeKey(indexName, []string{transfer.Name, transfer.FileHash})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the marble.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutState(originatorHashIndexKey, value)

	// ==== Marble saved and indexed. Return success ====
	fmt.Println("- end init fileTransfer")
	return shim.Success(nil)
}

// ===============================================
// readTransfer - read a transfer from chaincode state
// ===============================================
func (t *FileTransferChaincode) readTransfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var uuid, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting fileHash of the transfer to query")
	}

	uuid = args[0]
	valAsbytes, err := stub.GetState(uuid) //get the transfer from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + uuid + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Transfer does not exist: " + uuid + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ==================================================
// delete - remove a transfer key/value pair from state
// ==================================================
func (t *FileTransferChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var transferJSON transfer
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	transferUUID := args[0]

	// to maintain the originator~hash index, we need to read the transfer first and get its originator
	valAsbytes, err := stub.GetState(transferUUID) //get the marble from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + transferUUID + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Transfer does not exist: " + transferUUID + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &transferJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + transferUUID + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(transferUUID) //remove the transfer from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// maintain the index
	indexName := "originator~hash"
	originatorHashIndexKey, err := stub.CreateCompositeKey(indexName, []string{transferJSON.Name, transferJSON.FileHash})
	if err != nil {
		return shim.Error(err.Error())
	}

	//  Delete index entry to state.
	err = stub.DelState(originatorHashIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

// =======Rich queries =========================================================================
// Two examples of rich queries are provided below (parameterized query and ad hoc query).
// Rich queries pass a query string to the state database.
// Rich queries are only supported by state database implementations
//  that support rich query (e.g. CouchDB).
// The query string is in the syntax of the underlying state database.
// With rich queries there is no guarantee that the result set hasn't changed between
//  endorsement time and commit time, aka 'phantom reads'.
// Therefore, rich queries should not be used in update transactions, unless the
// application handles the possibility of result set changes between endorsement and commit time.
// Rich queries can be used for point-in-time queries against a peer.
// ============================================================================================

// ===========================================================================================
// constructQueryResponseFromIterator constructs a JSON array containing query results from
// a given result iterator
// ===========================================================================================
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}

// =========================================================================================
// getQueryResultForQueryString executes the passed in query string.
// Result set is built and returned as a byte array containing the JSON results.
// =========================================================================================
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

// ===== Example: Parameterized rich query =================================================
// queryTransfersByOriginator queries for transfers based on a passed in originator.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *FileTransferChaincode) queryTransfersByOriginator(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	originatorName := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"fileTransfer\",\"originatorName\":\"%s\"}}", originatorName)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// ===== Example: Parameterized rich query =================================================
// queryTransfersByRecipient queries for transfers based on a passed in recipient.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *FileTransferChaincode) queryTransfersByRecipient(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	recipientName := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"fileTransfer\",\"recipientName\":\"%s\"}}", recipientName)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// ===== Example: Ad hoc rich query ========================================================
// queryTransfers uses a query string to perform a query for marbles.
// Query string matching state database syntax is passed in and executed as is.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the queryTransfersForOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *FileTransferChaincode) queryTransfers(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "queryString"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func (t *FileTransferChaincode) getHistoryForTransfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	transferUUID := args[0]

	fmt.Printf("- start getHistoryForTransfer: %s\n", transferUUID)

	resultsIterator, err := stub.GetHistoryForKey(transferUUID)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the marble
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON marble)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryForTransfer returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}
