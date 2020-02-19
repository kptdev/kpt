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

export PROMPT_TIMEOUT=3600

########################
# include the magic
########################
. demo-magic/demo-magic.sh

cd $(mktemp -d)
git init

# hide the evidence
clear

pwd

bold=$(tput bold)
normal=$(tput sgr0)

# start demo
clear
p "# fetch the package..."
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"

echo " "
p "# print its contents..."
pe "kpt cfg tree helloworld --image --ports --name --replicas  --field 'metadata.labels'"

echo " "
p "# add to git..."
pe "git add helloworld && git commit -m 'fetch helloworld package at v0.1.0'"

echo " "
p "# print setters..."
pe "kpt cfg list-setters helloworld"

echo " "
p "# change a value..."
pe "kpt cfg set helloworld replicas 3 --set-by phil --description 'minimal HA mode'"

echo " "
p "# print setters again..."
pe "kpt cfg list-setters helloworld"

echo " "
p "# print its contents..."
pe "kpt cfg tree helloworld --name --replicas"

echo " "
p "# view the diff..."
pe "git diff"

echo " "
p "# commit changes..."
pe "git add helloworld && git commit -m 'set replicas to 3'"

echo " "
p "# update the package to a new version..."
pe "kpt pkg update helloworld@v0.2.0 --strategy=resource-merge"

echo " "
p "# view the diff..."
pe "git diff"

echo " "
p "# print its contents..."
pe "kpt cfg tree helloworld --name --replicas --field 'metadata.labels'"

