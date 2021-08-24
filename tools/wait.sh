#!/bin/bash

set -e

waitforit -address tcp://localhost:5672 -timeout 30
waitforit -address tcp://localhost:3306 -timeout 30
waitforit -address tcp://localhost:5432 -timeout 30
waitforit -address tcp://localhost:8081 -timeout 30

echo "waiting for mysql"
while ! docker logs container | grep -q "ready for connections";
do
    sleep 1
    echo "."
done