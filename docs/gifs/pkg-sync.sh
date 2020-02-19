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

export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git

# start demo
clear
echo "# start with a package"
echo "kpt pkg init . --description 'my package'"
kpt pkg init . --description 'my package'
pe "tree ."

echo " "
echo "$ export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git"
p "# 'kpt pkg sync set' adds a package dependency to the Kptfile"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.1.0 hello-world-prod"

echo " "
p "# 'kpt sync' syncs all package dependencies "
p "# at the versions specified in the Kptfile"
pe "kpt pkg sync ."
pe "tree ."
pe "kpt cfg count ."

echo " "
p "# the same package may be added multiple times"
p "# each copy may have a different version"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.2.0 hello-world-dev"
pe "kpt pkg sync ."
pe "tree ."

echo " "
p "# view the package differences"
pe "diff hello-world-dev hello-world-prod"
echo "$ git add . && git commit -m 'synced packaged'"
git add . > /dev/null
git commit -m 'synced packaged' > /dev/null

echo " "
p "# 'kpt pkg sync set' may also be used to update the version of a dependency"
pe "kpt pkg sync set \$SRC_REPO/package-examples/helloworld-set@v0.2.0 hello-world-1"
pe "kpt pkg sync ."
pe "diff hello-world-dev hello-world-prod"

p "# for more information see 'kpt help pkg sync'"
p "kpt help pkg sync"
