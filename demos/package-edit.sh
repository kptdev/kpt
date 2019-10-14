#!/bin/bash
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


export TYPE_SPEED=500

mkdir package-edit-demo
cd package-edit-demo
git init .

kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 ./cockroachdb
git add . && git commit -m 'fetch cockroachdb'

########################
# include the magic
########################
. $HOME/demo-magic/demo-magic.sh

# hide the evidence
clear

# Checkout the package
pe "echo 'cloned package https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 to cockroachdb/'"
pe "kpt tree cockroachdb/ --all"
pe "clear"

pe "echo 'display duck-typed commands (get + set)'"
pe "kpt cockroachdb/"
pe "kpt cockroachdb/ set"
pe "kpt cockroachdb/ get"
pe "clear"

pe "echo 'set replicas using the cli'"
pe "kpt cockroachdb/ get replicas cockroachdb"
pe "kpt cockroachdb/ set replicas cockroachdb --value 7"
pe "kpt cockroachdb/ get replicas cockroachdb"
pe "git diff -u"
pe "git add . && git commit -m 'update cockroachdb replicas to 7'"
pe "clear"

pe "echo 'set cpu and memory resources using the cli'"
pe "kpt cockroachdb/ set cpu-limits cockroachdb --value 500m"
pe "kpt cockroachdb/ set cpu-requests cockroachdb --value 300m"
pe "kpt cockroachdb/ set memory-limits cockroachdb --value 800M"
pe "kpt cockroachdb/ set memory-requests cockroachdb --value 400M"
pe "kpt tree cockroachdb/ --name --resources --replicas --env"
pe "git diff -u"
pe "git add . && git commit -m 'update cockroachdb resources'"
pe "clear"

pe "echo 'set environment variables using the cli'"
pe "kpt cockroachdb/ set env cockroachdb --name FOO --value BAR"
pe "kpt tree cockroachdb/ --name --resources --replicas --env"
pe "git diff -u"
pe "git add . && git commit -m 'update cockroachdb env with FOO'"
pe "clear"

pe "echo 'set image using the cli -- expect to fail'"
pe "kpt cockroachdb/ get image cockroachdb"
pe "kpt cockroachdb/ set image cockroachdb --value foo:v1"
pe "less cockroachdb/Kptfile"
pe "clear"

# End of demo
pe ""
