#! /bin/bash
if [ "$#" -ne 2 ];
    then echo "Usage: enrolUser.sh <username> <password>"
    exit
fi

docker exec -it ca.example.com fabric-ca-client enroll -u http://$1:$2@localhost:7054
