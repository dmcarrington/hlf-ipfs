#! /bin/bash

if [ "$#" -ne 2 ];
    then echo "Usage: addUser.sh <firstname> <lastname>"
    exit
fi

cat >/tmp/newUser.ldif <<EOF
dn: cn=$1$2,dc=example,dc=com
changetype: add
cn: $1$2
sn: $2
objectClass: organizationalPerson
objectClass: person
objectClass: top
EOF

docker exec ldap-server ldapadd -x -w "adminpw" -D "cn=admin,dc=example,dc=com" -f /tmp/newUser.ldif
docker exec -it ldap-server ldappasswd -x -w "adminpw" -D "cn=admin,dc=example,dc=com" -S "cn=$1$2,dc=example,dc=com"