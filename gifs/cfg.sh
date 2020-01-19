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

export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null

# start demo
clear

echo "# start with helloworld package"
echo "$ kpt pkg desc helloworld"
kpt pkg desc helloworld

# start demo
echo " "
p "# 'kpt cfg' contains commands for printing and modifying local packages of configuration"

echo " "
p "# print the Resource counts"
pe "kpt cfg count helloworld"

echo " "
p "# print the package using a tree structure"
pe "kpt cfg tree helloworld --name --image --replicas"

echo " "
p "# filter to only print Services"
pe "kpt cfg grep \"kind=Service\" helloworld | kpt cfg tree --name --image --replicas"

echo " "
p "# list setters and set fields"
pe "kpt cfg list-setters helloworld replicas"
pe "kpt cfg set helloworld replicas 3"

echo " "
p "# format configuration"
pe "kpt cfg fmt helloworld"

p "# for more information see 'kpt help cfg'"
p "kpt help cfg"
