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
. demo-magic/demo-magic.sh

cd $(mktemp -d)
git init

# hide the evidence
clear

bold=$(tput bold)
normal=$(tput sgr0)
stty rows 50 cols 180

# start demo
echo ""
echo "  ${bold}fetch the package...${normal}"
pe "kpt get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"

echo ""
echo "  ${bold}print its contents...${normal}"
pe "config tree helloworld --image --ports --name --replicas  --field 'metadata.labels'"

echo ""
echo "  ${bold}add to git...${normal}"
pe "git add helloworld && git commit -m 'fetch helloworld package at v0.1.0'"

echo ""
echo "  ${bold}print setters...${normal}"
pe "config set helloworld"

echo ""
echo "  ${bold}change a value...${normal}"
pe "config set helloworld replicas 3 --set-by phil --description 'minimal HA mode'"

echo ""
echo "  ${bold}print setters again...${normal}"
pe "config set helloworld"

echo ""
echo "  ${bold}print its contents...${normal}"
pe "config tree helloworld --name --replicas"

echo ""
echo "  ${bold}view the diff...${normal}"
pe "git diff"

echo ""
echo "  ${bold}commit changes...${normal}"
pe "git add helloworld && git commit -m 'set replicas to 3'"

echo ""
echo "  ${bold}update the package to a new version...${normal}"
pe "kpt update helloworld@v0.2.0 --strategy=resource-merge"

echo ""
echo "  ${bold}view the diff...${normal}"
pe "git diff"

echo ""
echo "  ${bold}print its contents...${normal}"
pe "config tree helloworld --name --replicas --field 'metadata.labels'"

echo ""
echo "  ${bold}update git...${normal}"
pe "git add helloworld && git commit -m 'update helloworld package to v0.2.0'"
