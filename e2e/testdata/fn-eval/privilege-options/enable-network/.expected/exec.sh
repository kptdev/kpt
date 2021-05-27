#! /bin/bash

set -eo pipefail

kpt fn eval \
--image gcr.io/kpt-fn/kubeval:v0.1 \
--network \
-- \
schema_location='https://kubernetesjsonschema.dev'
