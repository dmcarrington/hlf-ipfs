# Hyperledger Fabric IPFS file transfer recorder

This rproject demonstrates how Hyperledger Fabric can be used to record transfers of files transferred across the IPFS network. It is based upon the hlf-ldap repository for authenticating user accounts. At a high level, a user, once enrolled in the Fabric CA, can initiate a file transfer using IPFS to another user on the system. The details of the transfer, including originator, recipient, file name and IPFS entry, are recorded in the Fabric chaincode. The intention is that this information could be examined in future by a hypothetical auditor.
 
## Basic Network Config

Note that this basic configuration uses pre-generated certificates and
key material, and also has predefined transactions to initialize a 
channel named "mychannel".

To (re)build the webApp docker image, run ``docker-compose build``.
To start the network, run ``start.sh``.
To stop it, run ``stop.sh``
To completely remove the network on your system, run ``teardown.sh``.

## Usage (CLI)
To enrol the initial Admin user in the CA, run ``enrolAdmin.sh``.
To add a new user to the LDAP server, run ``addUser.sh <firstname> <lastname>``. This will create a user with the username ``<firstname><lastname>``. Enter a password for the user when prompted. You can then enrol this user in the CA using ``enrolUser.sh <username> <password>``.

## Usage (SDK)
Add new users to the LDAP server using the ``addUser.sh`` script as above.
To register a new user to the CA, Click on "Sign in", then "I need to create an account", and enter the credentials provided to the addUser script.

## Webapp
Intended to provide a user logon interface which will then allow authenticated interaction with the blockchain.

At present, the user can log on using an account that has been created using the ``addUser.sh`` followed by one of the ``enrolUser`` scripts.
The ``I need to create an account`` route enrols a previously created LDAP user into the HLF network.

After logging in, provided the account has been successfully registered with Fabric, the user should see lists of all transfers that have been initiated by them, and all transfers that have them as them as the recipient. A simple web form allows the user to upload a file to IPFS and specify a recipient.

## TODO
Complete work on getting 'open' buttons to work.
Fix updating of lists after committing a new file.
Display file name and received status. 
Encrypt files using PGP or similar before saving to IPFS

### Notes on building Chaincode with 3rd party dependencies
The fileTransfer chaincode requires the use of 3rd party go components. This requires the code to be vendored before being instantiated on the Fabric network. I used the following steps, based on https://www.youtube.com/watch?v=-mlUaJbFHcM.

``cd /home/davidcarrington/go/src/github.com/hyperledger/fabric/
cp -r ~/git/hlf-ldap/chaincode/fileTransfer .
cd fileTransfer/
govendor init
govendor fetch github.com/google/uuid
govendor add +external
cd ..
cp -r fileTransfer ~/git/hlf-ldap/chaincode/
cd ~/git/hlf-ldap/chaincode/
go build
cd fileTransfer/
go build
rm fileTransfer``

<a rel="license" href="http://creativecommons.org/licenses/by/4.0/"><img alt="Creative Commons License" style="border-width:0" src="https://i.creativecommons.org/l/by/4.0/88x31.png" /></a><br />This work is licensed under a <a rel="license" href="http://creativecommons.org/licenses/by/4.0/">Creative Commons Attribution 4.0 International License</a>

## Maintainer
David Carrington (dmcarrington@googlemail.com)