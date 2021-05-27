#! /bin/bash

set -eo pipefail

kpt fn source |\
kpt fn eval - --image gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=staging
