#! /bin/bash

set -eo pipefail

# create a temporary directory
TEMP_DIR=$(mktemp -d)

kpt fn render -o $TEMP_DIR

# copy the resources file back but retain Kptfile
cp $TEMP_DIR/resources.yaml ./

# remove temporary directory
rm -r $TEMP_DIR
