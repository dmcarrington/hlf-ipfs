package main

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("minimalcc")

// Transaction
func initialise(stub shim.ChaincodeStubInterface) pb.Response {

	args := stub.GetArgs()

	name1 := string(args[1])
	amount1 := args[2]

	logger.Infof("Name1: %s Amount1: %s", name1, string(amount1))

	err := stub.PutState(name1, amount1)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to store state %v", err))
	}

	name2 := string(args[3])
	amount2 := args[4]

	logger.Infof("Name2: %s Amount2: %s", name2, string(amount2))

	err = stub.PutState(name2, amount2)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to store state %v", err))
	}

	return shim.Success([]byte("Initialisation completed"))
}

func pay(stub shim.ChaincodeStubInterface) pb.Response {

	args := stub.GetStringArgs()

	payee := args[1]
	payer := args[3]
	amtToPay, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to convert string to integer %v", err))
	}

	logger.Infof("Pay: %s amount: %d from: %s", payee, amtToPay, payer)

	state, err := stub.GetState(payee)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get payee state %v", err))
	}

	var payeeCurrentState int
	payeeCurrentState, err = strconv.Atoi(string(state))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get payee current state %v", err))
	}

	logger.Infof("Before payment - payeeCurrentState: %d", payeeCurrentState)

	payeeCurrentState = payeeCurrentState + amtToPay

	var payerCurrentState int
	state, err = stub.GetState(payer)
	payerCurrentState, err = strconv.Atoi(string(state))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get payer current state %v", err))
	}

	logger.Infof("Before payment - payerCurrentState: %d", payerCurrentState)

	payerCurrentState = payerCurrentState - amtToPay

	logger.Infof("After payment - payee state: %d payer state: %d", payeeCurrentState, payerCurrentState)

	err = stub.PutState(payee, []byte(strconv.Itoa(payeeCurrentState)))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to update payee state %v", err))
	}

	err = stub.PutState(payer, []byte(strconv.Itoa(payerCurrentState)))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to update payer state %v", err))
	}

	return shim.Success([]byte("Payment done"))
}

// Queries
func query(stub shim.ChaincodeStubInterface) pb.Response {

	logger.Info("query")

	args := stub.GetStringArgs()

	queryStatement := args[1]

	queryResult, err := stub.GetState(queryStatement)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to execute query find %v and error %v", queryStatement, err))
	}

	return shim.Success([]byte(fmt.Sprintf("%v has %v", queryStatement, string(queryResult))))

}

// SimpleChaincode representing a class of chaincode
type SimpleChaincode struct{}

// Init to initiate the SimpleChaincode class
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Info("Hello Init")
	fcn, _ := stub.GetFunctionAndParameters()
	if fcn == "init" {
		return initialise(stub)
	}

	return shim.Error("Fail to initialise state")
}

// Invoke a method specified in the SimpleChaincode class
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Info("Hello Invoke")
	fcn, _ := stub.GetFunctionAndParameters()

	if fcn == "pay" {
		return pay(stub)
	}

	if fcn == "query" {
		return query(stub)
	}

	return shim.Success([]byte("Invoke"))
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		logger.Debugf("Error: %s", err)
	}
}
