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

stty rows 90 cols 20

# start demo
clear

kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null

echo "$ # start with a package fetched at version v0.1.0"
echo "$ kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"
echo "$ git add . && git commit -m 'fetched helloworld'"

echo " "
p "# 'kpt pkg update' will pull in upstream changes to a package"
pe "kpt pkg desc helloworld"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"
pe "git diff"

echo " "
p "# reset back to version v0.1.0"
pe "git checkout helloworld"

echo " "
p "# update will merge remote changes rather than replacing the package"
p "# to keep local modifications"
p "# create a local change to the package by adding an annotation"
pe "kpt cfg annotate helloworld --kv demo=update"
pe "git diff"
pe "git add . && git commit -m 'updated annotations'"
git add . > /dev/null
git commit -m 'updated annotations' > /dev/null

echo " "
p "# update the package to v0.2.0 to add the labels"
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"

echo " "
p "# the package has both the locally added annotations and the remotely added labels"
pe "git diff"
pe "kpt cfg tree helloworld --field=metadata.annotations.demo --field=metadata.labels.app"

p "# for more information see 'kpt help pkg update'"
p "kpt help pkg update"
