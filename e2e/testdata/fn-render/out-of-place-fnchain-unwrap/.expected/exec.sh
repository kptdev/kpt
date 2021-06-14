#! /bin/bash

set -eo pipefail

kpt fn render -o stdout \
| kpt fn eval --image gcr.io/kpt-fn/set-annotations:v0.1.3 -- foo=bar \
| kpt fn eval --image gcr.io/kpt-fn/set-labels:v0.1.3 -o unwrap -- tier=backend
