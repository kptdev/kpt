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

########################
# include the magic
########################
. ../../demos/demo-magic/demo-magic.sh

cd $(mktemp -d)
git init > /dev/null

kpt pkg get https://github.com/kubernetes/examples/staging examples > /dev/null
git add . > /dev/null
git commit -m 'fetched examples' > /dev/null

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null

# start demo
clear

echo "# start with examples repo"
echo "$ kpt pkg desc examples"
kpt pkg desc examples

echo " "
p "# 'kpt cfg grep' searches for Resource within a package"
pe "kpt cfg grep 'spec.replicas>10' examples"

echo " "
p "# grep can be used with tree to filter results summaries"
pe "kpt cfg grep 'spec.replicas>10' examples | kpt cfg tree --replicas"

echo " "
p "# like other kpt commands, grep can accept Resources from stdin"
pe "kubectl get all -o yaml | kpt cfg grep 'kind=Service' | kpt cfg tree"

echo " "
p "# grep's results can be inverted with -v"
pe "kubectl get all -o yaml | kpt cfg grep -v 'kind=Service' | kpt cfg tree"

p "# for more information see 'kpt help cfg grep'"
p "kpt help cfg grep"
