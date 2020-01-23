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
echo ""
echo "  ${bold}fetch the package and add to git...${normal}"
pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"
pe "git add . && git commit -m 'helloworld'"

echo ""
echo "  ${bold}print the package contents...${normal}"
pe "kpt cfg tree helloworld --all"

echo ""
echo "  ${bold}list the setters...${normal}"
pe "kpt cfg list-setters helloworld"

echo ""
echo "  ${bold}create a setter for the Service port protocol...${normal}"
pe "kpt cfg create-setter helloworld http-protocol TCP --field protocol --type string --kind Service"

echo ""
echo "  ${bold}list the setters again -- the new one should show up...${normal}"
pe "kpt cfg list-setters helloworld"

echo ""
echo "  ${bold}change the Service port protocol using the setter...${normal}"
pe "kpt cfg set helloworld http-protocol UDP --description 'justification for UDP' --set-by 'phil'"

echo ""
echo "  ${bold}observe the updated configuration...${normal}"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg tree helloworld --ports"
pe "git diff"
