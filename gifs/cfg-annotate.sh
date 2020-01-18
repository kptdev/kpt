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

export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0

# start demo
clear

echo "$ export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0"
pe "kpt pkg get \$PKG helloworld"
pe "git add . && git commit -m 'fetched helloworld'"

# start demo
echo " "
p "# print the Resource counts"
pe "kpt cfg count helloworld"

echo " "
p "# print the structured package"
pe "kpt cfg tree helloworld --name --image --replicas"

echo " "
p "# filter to only print Services"
pe "kpt cfg grep \"kind=Service\" helloworld | kpt cfg tree --name --image --replicas"

echo " "
p "# print the raw Resource configuration"
pe "kpt cfg cat helloworld | less"

echo " "
p "# list settable options"
pe "kpt cfg list-setters helloworld replicas"

echo " "
p "# set the replicas using the cli"
pe "kpt cfg set helloworld replicas 3"

echo " "
p "# view the updated values"
pe "kpt cfg list-setters helloworld replicas"

echo " "
p "# view the updated configuration"
pe "git diff"
pe "git add . && git commit -m 'change replicas to 3'"

pe "clear"
