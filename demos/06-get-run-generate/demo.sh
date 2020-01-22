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

# hide the evidence
git init
clear

# start demo
echo ""
echo "  ${bold}fetch the package...${normal}"
pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-generate@v0.2.0 helloworld"

echo ""
echo "  ${bold}print the package resources...${normal}"
pe "config tree helloworld"
pe "cat helloworld/helloworld.yaml"

echo ""
echo "  ${bold}run the generator...${normal}"
pe "kpt cfg run helloworld"

echo ""
echo "  ${bold}print the generated resources...${normal}"
pe "kpt cfg tree helloworld --all"

echo ""
echo "  ${bold}update the config...${normal}"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg set helloworld replicas 5"
pe "kpt cfg list-setters helloworld"

echo ""
echo "  ${bold}run the generator...${normal}"
pe "kpt cfg run helloworld"

echo ""
echo "  ${bold}print the updated resources...${normal}"
pe "kpt cfg tree helloworld --all"
