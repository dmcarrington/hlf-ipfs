var FabricClient = require('fabric-client');
var fs = require('fs');
var path = require('path');
var configFilePath = path.join(__dirname, './ConnectionProfile.yml');
const CONFIG = fs.readFileSync(configFilePath, 'utf8')
class FBClient extends FabricClient {
    constructor(props) {
        super(props);
    }
    submitTransaction(requestData) {
        var returnData;
        var _this = this;
        var channel = this.getChannel();
        var peers = this.getPeersForOrg();
        var event_hub = this.getEventHub(peers[0].getName());
        return channel.sendTransactionProposal(requestData).then(function (results) {
            var proposalResponses = results[0];
            var proposal = results[1];
            let isProposalGood = false;
            if (proposalResponses && proposalResponses[0].response &&
                proposalResponses[0].response.status === 200) {
                isProposalGood = true;
                console.log('Transaction proposal was good');
            } else {
                throw new Error(results[0][0].details);
                console.error('Transaction proposal was bad');
            }
            returnData = proposalResponses[0].response.payload.toString();
            returnData = JSON.parse(returnData);
            if (isProposalGood) {
                console.log(
                    'Successfully sent Proposal and received ProposalResponse: Status - %s, message - "%s"',
                    proposalResponses[0].response.status, proposalResponses[0].response.message);
                var request = {
                    proposalResponses: proposalResponses,
                    proposal: proposal
                };
                var transaction_id_string = requestData.txId.getTransactionID();
                var promises = [];
                var sendPromise = channel.sendTransaction(request);
                promises.push(sendPromise); 
                let txPromise = new Promise((resolve, reject) => {
                    let handle = setTimeout(() => {
                        event_hub.disconnect();
                        resolve({ event_status: 'TIMEOUT' });
                    }, 3000);
                    event_hub.connect();
                    event_hub.registerTxEvent(transaction_id_string, (tx, code) => {
                        clearTimeout(handle);
                        event_hub.unregisterTxEvent(transaction_id_string);
                        event_hub.disconnect();
                        var return_status = { event_status: code, tx_id: transaction_id_string };
                        if (code !== 'VALID') {
                            console.error('The transaction was invalid, code = ' + code);
                            resolve(return_status);
                        } else {
                            console.log('The transaction has been committed on peer ' + event_hub._ep._endpoint.addr);
                            resolve(return_status);
                        }
                    }, (err) => {
                        console.log(err)
                        reject(new Error('There was a problem with the eventhub ::' + err));
                    });
                });
                promises.push(txPromise);
                return Promise.all(promises);
            } else {
                console.error('Failed to send Proposal or receive valid response. Response null or status is not 200. exiting...');
                throw new Error('Failed to send Proposal or receive valid response. Response null or status is not 200. exiting...');
            }
        }).then((results) => {
            console.log('Send transaction promise and event listener promise have completed');
            if (results && results[0] && results[0].status === 'SUCCESS') {
                console.log('Successfully sent transaction to the orderer.');
            } else {
                console.error('Failed to order the transaction. Error code: ' + response.status);
            }
            if (results && results[1] && results[1].event_status === 'VALID') {
                console.log('Successfully committed the change to the ledger by the peer');
            } else {
                console.log('Transaction failed to be committed to the ledger due to ::' + results[1].event_status);
            }
        }).then(function () {
            return returnData;
        })
    }
    query(requestData) {
        var channel = this.getChannel();
        return channel.queryByChaincode(requestData).then((response_payloads) => {
            var resultData = JSON.parse(response_payloads.toString('utf8'));
            return resultData;
        }).then(function(resultData) {
            if (resultData.constructor === Array) {
                resultData = resultData.map(function (item, index) {
                    if (item.data) {
                        return item.data
                    } else {
                        return item;
                    }
                })
            }
            return resultData;
        });
    }
}
var fabricClient = new FBClient();
fabricClient.loadFromConfig(configFilePath);
module.exports = fabricClient;
