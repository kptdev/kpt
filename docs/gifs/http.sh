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
git init

#stty rows 50 cols 180

# start demo
clear
echo "#"
echo "# get the package"
echo "#"
export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
echo "export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git"
pe "kpt pkg get \$SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld"
pe "git add . && git commit -m 'fetched helloworld'"

echo "#"
echo "# customize the package"
echo "#"
pe "kpt config set helloworld replicas 3"
pe "kpt config list-setters helloworld replicas"
pe "git add . && git commit -m 'change replicas to 3'"

echo "#"
echo "# pull in upstream updates by merging Resources"
echo "#"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"
pe "git diff helloworld"
pe "git add . && git commit -m 'update helloworld to 0.2.0'"

echo "#"
echo "# apply the package to a cluster"
echo "#"
pe "kpt http apply -f helloworld"

pe "clear"