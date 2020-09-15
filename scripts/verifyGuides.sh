#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

mdrip -alsologtostderr -v 10 --blockTimeOut 20m0s --mode test \
    --label verifyGuides site/content/en/guides

echo "Success"
