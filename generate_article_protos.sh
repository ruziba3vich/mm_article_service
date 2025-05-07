#!/bin/bash
# chmod +x generate_article_protos.sh

set -e

PROTO_DIR="./protos"

OUT_DIR="./genprotos"

mkdir -p $OUT_DIR

protoc -I=$PROTO_DIR \
  --go_out=$OUT_DIR \
  --go-grpc_out=$OUT_DIR \
  $PROTO_DIR/article_protos/article.proto

echo "protos generated successfully"
