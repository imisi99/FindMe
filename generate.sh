#!/bin/bash

set -e

protoc \
  --proto_path=proto \
  --go_out=. \
  --go-grpc_out=. \
  proto/emb.proto

protoc \
  --proto_path=proto \
  --go_out=. \
  --go-grpc_out=. \
  proto/rec.proto

echo "Proto files generated successfully"
