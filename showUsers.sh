#! /bin/bash

# List all users in the LDAP server
docker exec ldap-server ldapsearch -x -L -H ldap://ldap-server -b dc=example,dc=com -D "cn=admin,dc=example,dc=com" -w adminpw
