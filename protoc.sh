#!/bin/bash

protoc --proto_path=../googleapis/ --proto_path=. --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative spec.proto
