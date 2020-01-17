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
. ../demos/demo-magic/demo-magic.sh

cd $(mktemp -d)
git init

stty rows 80 cols 15

# start demo
clear
echo " "
export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
echo "$ export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git"
p "# pull down a package"
pe "kpt pkg get \$SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld"
pe "git add . && git commit -m 'fetched helloworld'"

echo " "
p "# list setters published by the packages"
pe "kpt cfg list-setters helloworld"

echo " "
p "# set the replicas on the cli"
pe "kpt cfg set helloworld replicas 3"
pe "kpt cfg list-setters helloworld replicas"
pe "git add . && git commit -m 'change replicas to 3'"

echo " "
p "# pull in upstream updates"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"

echo " "
p "# show the changes to the raw configuration"
pe "git diff helloworld"
pe "git add . && git commit -m 'update helloworld to 0.2.0'"

echo " "
p "# apply the package to a cluster with kubectl apply or kpt svr apply"
pe "kubectl apply -R -f helloworld"

pe "clear"
