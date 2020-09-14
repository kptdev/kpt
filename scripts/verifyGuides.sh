#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

mdrip --blockTimeOut 6m0s --mode test \
    --label verifyGuides site/content/en/guides

echo "Success"
