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

stty rows 80 cols 15

export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git

# start demo
clear
echo " "
p "# init a package"
pe "kpt pkg init . --description 'my package'"

echo " "
echo "$ export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git"
p "# add a dependency"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.1.0 hello-world-1"

echo " "
p "# sync the dependency"
pe "kpt pkg sync ."
pe "ls"
pe "kpt cfg count ."

echo " "
p "# add a second dependency at a different version"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.2.0 hello-world-2"
pe "kpt pkg sync ."
pe "ls"
pe "kpt cfg count ."

echo " "
p "# compare the packages"
pe "diff hello-world-1 hello-world-2"
pe "git add . && git commit -m 'synced packaged'"

echo " "
p "# update the first a dependency"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.2.0 hello-world-1"
pe "kpt pkg sync ."
pe "diff hello-world-1 hello-world-2"

pe "clear"