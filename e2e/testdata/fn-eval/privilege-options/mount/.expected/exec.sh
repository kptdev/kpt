#! /bin/bash
# Copyright 2021 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eo pipefail
# Download schema file
SCHEMA_DIR="schema/master-standalone"
mkdir -p "$SCHEMA_DIR"
curl -sSL 'https://kubernetesjsonschema.dev/master-standalone/configmap-v1.json' -o $SCHEMA_DIR/configmap-v1.json

kpt fn eval \
--image gcr.io/kpt-fn/kubeval:v0.1.1 \
--as-current-user \
--mount type=bind,src=$(pwd)/schema,dst=/schema-dir \
-- \
schema_location=file:///schema-dir

# Remove 'schema' to avoid unwanted diff
rm -r schema
