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

pwd

# start demo
echo ""
p "#  ${bold}fetch the package...${normal}"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-generate@v0.2.0 helloworld"

echo ""
p "#  ${bold}print the package resources (only-generators)...${normal}"
pe "config tree helloworld"
pe "cat helloworld/helloworld.yaml"

echo ""
p "#  ${bold}run the generator...${normal}"
pe "kpt fn run helloworld"

echo ""
p "#  ${bold}print the generated resources...${normal}"
pe "kpt cfg tree helloworld --name --image --replicas --ports"

echo ""
p "#  ${bold}update the generator using a setter...${normal}"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg set helloworld replicas 5"
pe "kpt cfg tree helloworld --name --replicas --field=metadata.annotations.foo"

echo ""
p "#  ${bold}update the generated (output) config with an annotation...${normal}"
pe "kpt cfg annotate helloworld --kv foo=bar --kind Deployment"
pe "kpt cfg tree helloworld --name --replicas --field=metadata.annotations.foo"

echo ""
p "#  ${bold}run the generator again${normal}"
pe "kpt fn run helloworld"

echo ""
p "#  ${bold}see that the changes have been merged...${normal}"
pe "kpt cfg tree helloworld --name --replicas --field=metadata.annotations.foo"
