/* Server for interacting directly with the HLF SDK. Opens a WebSocket on port 8081 to receive commands from the web app 
*/

var WebSocketServer = require('websocket').server;
var http = require('http');
 
var server = http.createServer(function(request, response) {
    console.log((new Date()) + ' Received request for ' + request.url);
    response.writeHead(404);
    response.end();
});
server.listen(8081, function() {
    console.log((new Date()) + ' Server is listening on port 8081');
});
 
wsServer = new WebSocketServer({
    httpServer: server,
    // You should not use autoAcceptConnections for production
    // applications, as it defeats all standard cross-origin protection
    // facilities built into the protocol and the browser.  You should
    // *always* verify the connection's origin and decide whether or not
    // to accept it.
    autoAcceptConnections: false
});
 
function originIsAllowed(origin) {
  // put logic here to detect whether the specified origin is allowed.
  return true;
}
 
wsServer.on('request', function(request) {
    if (!originIsAllowed(request.origin)) {
      // Make sure we only accept requests from an allowed origin
      request.reject();
      console.log((new Date()) + ' Connection from origin ' + request.origin + ' rejected.');
      return;
    }
    
    var connection = request.accept('ws-protocol', request.origin);
    console.log((new Date()) + ' Connection accepted.');
    connection.on('message', async function(message) {
        if (message.type === 'utf8') {
            console.log('Received Message: ' + message.utf8Data);
            const messageJson = JSON.parse(message.utf8Data);
            if(messageJson.messageType === 'register') {
              const username = messageJson.data.username;
              const password = messageJson.data.password;
              console.log('registering: ', username, ', ', password);
              const enrol = require('./enrolUser').enrolUser;
              let enrolStatus = "fail";
              try{
                enrolStatus = await enrol(username, password);
                console.log("enrol finished: ", enrolStatus);
              } catch(err){
                console.log('Exception from enrol: ', err);
              }
              const reply = {
                  message: 'enrolStatus',
                  status: enrolStatus
              }
              connection.sendUTF(JSON.stringify(reply));              
            } else if(messageJson.messageType === 'commitFile') {
              console.log(messageJson);
              const fileContent = messageJson.data.fileContent;
              const writeToIPFS = require('./webApp/ipfs').writeToIPFS;
              const commitHash = await writeToIPFS(fileContent);
              const reply = {
                message: 'commitStatus',
                hash: commitHash
              }
              // TODO: if successful, add an entry to the blockchain using originator,
              // file hash and recipient
              connection.sendUTF(JSON.stringify(reply));
            }
            
            //connection.sendUTF(message.utf8Data);
        }
        else if (message.type === 'binary') {
            console.log('Received Binary Message of ' + message.binaryData.length + ' bytes');
            connection.sendBytes(message.binaryData);
        }
    });
    connection.on('close', function(reasonCode, description) {
        console.log((new Date()) + ' Peer ' + connection.remoteAddress + ' disconnected.');
    });
});