#! /bin/bash

set -eo pipefail

# create a temporary directory
TEMP_DIR=$(mktemp -d)

kpt fn source \
| kpt fn eval - --image gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=staging \
| kpt fn sink $TEMP_DIR

# copy back the resources
rm -r ./*
cp $TEMP_DIR/* .

# remove temporary directory
rm -r $TEMP_DIR
