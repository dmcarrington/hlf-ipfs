/**
 * Server-side IPFS interface
 */
const ipfsClient = require('ipfs-http-client');

// Given the content of a file, write it to our IPFS cluster
function writeToIPFS(fileContent) {
    const ipfs = ipfsClient('ipfs.infura.io', '5001', { protocol: 'https' }); // leaving out the arguments will default to these values
    if(ipfs) {
      return ipfs.add(Buffer.from(fileContent)).then((res) => {
        let ipfsHash = res[0].hash;
        console.log('Saved IPFS file: ' + ipfsHash);
        return ipfsHash;
      });
    } else {
      return 0;
    }
  }

module.exports = {writeToIPFS:writeToIPFS};