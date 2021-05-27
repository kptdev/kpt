#! /bin/bash
# Copyright 2021 Google LLC
#


set -eo pipefail

# Function gcr.io/kpt-fn-demo/foo:v0.1 prints "foo" to stderr and
# function gcr.io/kpt-fn-demo/bar:v0.1 prints "bar" to stderr.
# We intentionally tag a wrong image as gcr.io/kpt-fn-demo/foo:v0.1, since we
# expect the correct image to be pulled and override the wrong image.
docker pull gcr.io/kpt-fn-demo/bar:v0.1
docker tag gcr.io/kpt-fn-demo/bar:v0.1 gcr.io/kpt-fn-demo/foo:v0.1
