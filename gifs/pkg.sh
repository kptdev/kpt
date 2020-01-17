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
p "# 'kpt pkg get' fetches a package from a git repository"
pe "kpt pkg get \$SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld"
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
echo "$ git add . && git commit -a -m 'helloworld package'"

echo " "
p "# 'kpt pkg desc' lists information about the source of the package"
pe "kpt pkg desc helloworld"

echo " "
p "# packages may publish metadata for how to set specific fields"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg set helloworld replicas 3"
pe "kpt cfg list-setters helloworld replicas"
pe "git diff helloworld"
git commit -a -m 'helloworld package' > /dev/null
echo "$ git commit -a -m 'helloworld package'"

echo " "
p "# 'kpt pkg diff' displays a diff of the local package"
p "# against an upstream version"
pe "kpt pkg diff helloworld@v0.1.0"

echo " "
p "# 'kpt pkg update' pulls upstream updates and merges them, "
p "# keeping local changes to the package"
pe "git diff helloworld"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"
pe "git diff helloworld"

p "# for more information see 'kpt help pkg'"
p "kpt help pkg"