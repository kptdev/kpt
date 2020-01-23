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
echo "  ${bold}fetch the package...${normal}"
pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld1"

echo ""
echo "  ${bold}fetch the package, automatically setting field values...${normal}"
pe "KPT_SET_REPLICAS=3 kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld2"

pe "kpt cfg tree helloworld1 --replicas"
pe "kpt cfg tree helloworld2 --replicas"

pe "kpt cfg list-setters helloworld1"
pe "kpt cfg list-setters helloworld2"

pe "diff helloworld1 helloworld2"
