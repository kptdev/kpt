#! /bin/bash

set -eo pipefail
# Download schema file
SCHEMA_DIR="schema/master-standalone"
mkdir -p "$SCHEMA_DIR"
curl -sSL 'https://kubernetesjsonschema.dev/master-standalone/configmap-v1.json' -o $SCHEMA_DIR/configmap-v1.json

kpt fn eval \
--image gcr.io/kpt-fn/kubeval:v0.1 \
--as-current-user \
--mount type=bind,src=$(pwd)/schema,dst=/schema-dir \
-- \
schema_location=file:///schema-dir

# Remove 'schema' to avoid unwanted diff
rm -r schema
