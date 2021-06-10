#! /bin/bash

set -eo pipefail

kpt fn render -o stdout
