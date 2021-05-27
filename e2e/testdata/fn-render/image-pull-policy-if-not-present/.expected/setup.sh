#! /bin/bash

set -eo pipefail

# Function gcr.io/kpt-fn-demo/foo:v0.1 prints "foo" to stderr and
# function gcr.io/kpt-fn-demo/bar:v0.1 prints "bar" to stderr.
# We intentionally tag a wrong image as pull gcr.io/kpt-fn-demo/bar:v0.1
docker pull gcr.io/kpt-fn-demo/foo:v0.1
docker tag gcr.io/kpt-fn-demo/foo:v0.1 gcr.io/kpt-fn-demo/bar:v0.1
