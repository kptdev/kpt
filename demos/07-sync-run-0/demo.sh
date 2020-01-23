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

d=$(pwd)

cd $(mktemp -d)
git init

# hide the evidence
clear

pwd

bold=$(tput bold)
normal=$(tput sgr0)

# start demo
p "#  ${bold}setup the local package...${normal}"
pe "kpt pkg init ."
pe "kpt pkg sync set git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld-prod"
pe "kpt pkg sync set git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.2.0 helloworld-canary"
pe "cat Kptfile"

echo ""
p "#  ${bold}sync the packages...${normal}"
pe "kpt pkg sync ."
pe "git status"
pe "git add . && git commit -m 'sync packages'"
pe "kpt cfg tree . --field=metadata.labels"
pe "diff helloworld-prod helloworld-canary"

echo ""
p "#  ${bold}promote from canary to prod...${normal}"
pe "kpt pkg sync set git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.2.0 helloworld-prod"
pe "kpt pkg sync ."
pe "kpt cfg tree . --field=metadata.labels"
pe "diff helloworld-prod helloworld-canary"
