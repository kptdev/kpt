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
echo "$ export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git"
p "# fetch the package"
export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
pe "kpt pkg get \$SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld"
pe "git add . && git commit -m 'fetched helloworld'"

p "# make local changes to the package"
pe "kpt cfg annotate helloworld --kv example.com/demo=update"
pe "git diff"
pe "git add . && git commit -m 'fetched helloworld'"

echo " "
p "# pull in upstream updates from v0.2.0 which adds a label"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"
pe "git status"

echo " "
p "# package contains both locally added annotation and upstream label update"
pe "git diff"
pe "kpt cfg tree helloworld --field=metadata.annotations --field=metadata.labels"
