#!/bin/bash

protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative spec.proto

protoc spec.proto --js_out=import_style=commonjs:../autowp-frontend/generated --grpc-web_out=import_style=commonjs+dts,mode=grpcwebtext:../autowp-frontend/generated
