#!/bin/bash
set -e
export PATH="$PWD:$PATH"
go test -v -tags docker ./e2e -run TestFnRender/condition
