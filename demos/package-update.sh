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


export TYPE_SPEED=100

mkdir packaging-rebase-demo
cd packaging-rebase-demo
git init .
pwd

########################
# include the magic
########################
. $HOME/demo-magic/demo-magic.sh

# hide the evidence
clear

# Checkout the package
pe "echo 'clone the cockroachdb package'"
pe "kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 ./cockroachdb"
pe "git add . && git commit -m 'fetch cockroachdb v1.0.0'"
pe "clear"

# View
pe "echo 'view the package contents'"
pe "tree cockroachdb/"
pe "kpt rc cockroachdb/"
pe "kpt tree cockroachdb/ --image --replicas --name --resources"
pe "kpt cat cockroachdb/ | less"
pe "clear"

# Customize
pe "echo 'customize the package'"
pe "kpt cockroachdb/ set replicas cockroachdb --value 7"
pe "kpt cockroachdb/ get replicas cockroachdb"

pe "kpt cockroachdb/ set cpu-limits cockroachdb --value 500m"
pe "kpt cockroachdb/ get cpu-limits cockroachdb"

pe "git diff -u"
pe "kpt tree cockroachdb/ --image --replicas --name --resources"

pe "git add . && git commit -m 'change cockroachdb cpu and replicas'"
pe "clear"

# Update the package
pe "kpt update cockroachdb/@v1.1.0"
pe "git diff -u"
pe "kpt tree cockroachdb/ --image --replicas --name --resources"
pe "kpt cockroachdb/ get replicas cockroachdb"
pe "kpt cockroachdb/ get cpu-limits cockroachdb"
pe "git add . && git commit -m 'update cockroachdb to v1.1.0'"
pe "clear"

# End of demo
pe ""
