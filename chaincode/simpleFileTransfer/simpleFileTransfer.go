/*
 * Simple smart contract for recording a transfer over IPFS
 */

package main

/* Imports
 * 4 utility libraries for formatting, handling bytes, reading and writing JSON, and string manipulation
 * 2 specific Hyperledger Fabric specific libraries for Smart Contracts
 */
import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

// Define the Smart Contract structure
type SmartContract struct {
}

// Define the car structure, with 4 properties.  Structure tags are used by encoding/json library
type fileTransfer struct {
	UUID       string `json:"uuid"`
	Originator string `json:"originator"`
	FileHash   string `json:"fileHash"`
	Recipient  string `json:"recipient"`
}

/*
 * The Init method is called when the Smart Contract "simpleFileTransfer" is instantiated by the blockchain network
 * Best practice is to have any Ledger initialization in separate function -- see initLedger()
 */
func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil)
}

/*
 * The Invoke method is called as a result of an application request to run the Smart Contract "simpleFileTransfer"
 * The calling application program has also specified the particular smart contract function to be called, with arguments
 */
func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {

	// Retrieve the requested Smart Contract function and arguments
	function, args := APIstub.GetFunctionAndParameters()
	// Route to the appropriate handler function to interact with the ledger appropriately
	if function == "queryTransfer" {
		return s.queryTransfer(APIstub, args)
	} else if function == "initLedger" {
		return s.initLedger(APIstub)
	} else if function == "createTransfer" {
		return s.createTransfer(APIstub, args)
	} else if function == "queryAllTransfers" {
		return s.queryAllTransfers(APIstub, args)
	} else if function == "queryTransfersByRecipient" {
		return s.queryTransfersByRecipient(APIstub, args)
	}

	return shim.Error("Invalid Smart Contract function name.")
}

// ======================== queryTransfer =================================================
// queryTransfersByRecipient queries for transfers based on a specific key.
// args[0]: key of the record to search for
// =========================================================================================
func (s *SmartContract) queryTransfer(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	transferAsBytes, _ := APIstub.GetState(args[0])
	return shim.Success(transferAsBytes)
}

// Set the initial state of the ledger - currently unused
func (s *SmartContract) initLedger(APIstub shim.ChaincodeStubInterface) sc.Response {

	return shim.Success(nil)
}

// ======================== createTransfer =================================================
// createTransfer creates a new transfer of a single file from an originator and recipient.
// args[0]: originator
// args[1]: hash of the file in ipfs
// args[2]: recipient
// =========================================================================================
func (s *SmartContract) createTransfer(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	id, err := uuid.NewUUID()
	if err != nil {
		// handle error
		return shim.Error("Failed to generate UUID for product")
	}
	uuid := id.String()

	originator := args[0]
	fileHash := args[1]
	recipient := args[2]

	// Ensure that this transfer has not already been created before continuing
	key, _ := APIstub.CreateCompositeKey("fileTransfer", []string{originator, fileHash, recipient})
	existingTransfer, err := APIstub.GetState(key)

	if err != nil {
		return shim.Error("Failed to get transfer: " + err.Error())
	} else if existingTransfer != nil {
		return shim.Error("This transfer already exists: " + fileHash)
	}

	var transfer = fileTransfer{UUID: uuid, Originator: originator, FileHash: fileHash, Recipient: recipient}

	transferAsBytes, _ := json.Marshal(transfer)

	APIstub.PutState(key, transferAsBytes)

	return shim.Success(nil)
}

// ============= queryTransfersByRecipient =================================================
// queryTransfersByRecipient queries for transfers based on a passed in recipient.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (recipient).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SmartContract) queryTransfersByRecipient(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	recipientName := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"recipient\":\"%s\"}}", recipientName)

	resultsIterator, err := APIstub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
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

	fmt.Printf("- queryAllTransfers:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

// ==================== queryAllTransfers =================================================
// queryTransfersByRecipient queries for transfers based on a passed in originator.
// This is an example of a query using the partial composite key to locate a record
// =========================================================================================
func (s *SmartContract) queryAllTransfers(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	resultsIterator, err := APIstub.GetStateByPartialCompositeKey("fileTransfer", []string{args[0]})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
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

	fmt.Printf("- queryAllTransfers:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {

	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}
