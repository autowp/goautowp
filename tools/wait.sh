#!/bin/sh

set -e

waitforit -address tcp://localhost:5672 -timeout 30
waitforit -address tcp://localhost:3306 -timeout 30
waitforit -address tcp://localhost:5432 -timeout 30
waitforit -address tcp://localhost:8081 -timeout 30

echo "waiting for mysql"
while ! docker-compose logs mysql | grep -q "ready for connections";
do
    sleep 1
    echo "."
done

while ! echo "select version()" | mysql -u root -ppassword autowp --host=127.0.0.1 | grep version;
do
    sleep 1
    echo "."
done
