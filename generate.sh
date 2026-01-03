#!/bin/bash

set -e

protoc \
  --proto_path=proto \
  --go_out=. \
  --go-grpc_out=. \
  proto/findme.proto

echo "Proto files generated successfully"
