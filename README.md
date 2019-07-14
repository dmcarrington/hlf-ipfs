# Hyperledger Fabric LDAP interface

This repository holds example code for using LDAP authentication for enrolling users in the Hyperledger Fabric CA. At present, all interactions are carried out using shell scripts.

## Aims for future development
1) Add scripts to add and remove users on the LDAP docker.
2) Start using SDK functions where possible to replace shell scripting.
3) Add a simple web UI to allow a user to log on, authenticated against the LDAP server, and then execute chaincode as that user.

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

## WebSocket server **
This creates a WebSocket server that listens for messages on port 8081 and interacts with the HLF network via the SDK. At present, only the enrol functionality has been implemented.

## Webapp
Intended to provide a user logon interface which will then allow authenticated interaction with the blockchain.

At present, the user can log on using an account that has been created using the ``addUser.sh`` followed by one of the ``enrolUser`` scripts.
The ``I need to create an account`` route enrols a previously created LDAP user into the HLF network.

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