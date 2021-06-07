#! /bin/bash

set -eo pipefail

# create a temporary directory for results
results=$(mktemp -d)

kpt fn render -o stdout --results-dir $results \
| kpt fn eval - --image gcr.io/kpt-fn/set-annotations:v0.1.3 --results-dir $results -- foo=bar \
| kpt fn eval - --image gcr.io/kpt-fn/set-labels:v0.1.3 --results-dir $results -- tier=backend

# remove temporary directory
rm -r $results