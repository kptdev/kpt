#! /bin/bash

set -eo pipefail

IMAGE_TAG="gcr.io/kpt-fn-demo/printenv:v0.1"
export EXPORT_ENV="export_env_value"

kpt fn source \
| kpt fn eval - --image $IMAGE_TAG -e EXPORT_ENV -e FOO=BAR
