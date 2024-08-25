#!/bin/bash

# go install github.com/protocolbuffers/protobuf@latest
# go install pkg/mod/github.com/googleapis/googleapis@v0.0.0-20240823220356-a67e27687c1b/
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --proto_path=$GOPATH/pkg/mod/github.com/protocolbuffers/protobuf@v5.27.2+incompatible/src/ \
       --proto_path=$GOPATH/pkg/mod/github.com/googleapis/googleapis@v0.0.0-20240823220356-a67e27687c1b/ \
       --proto_path=. \
       --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative spec.proto
