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

# Put your stuff here
bold=$(tput bold)
normal=$(tput sgr0)

# start demo
echo ""
p "#  ${bold}fetch the package...${normal}"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0 helloworld"

echo ""
p "#  ${bold}print the package resources...${normal}"
pe "kpt cfg tree helloworld --resources --field 'metadata.annotations.tshirt-size'"
pe "cat helloworld/deploy.yaml"

echo ""
p "#  ${bold}annotate the Deployment with a tshirt-size...${normal}"
pe "kpt cfg annotate helloworld --kv tshirt-size=small --kind Deployment"
pe "kpt cfg tree helloworld --resources --field 'metadata.annotations.tshirt-size'"

echo ""
p "#  ${bold}locally run the tshirt-size function against the package...${normal}"
pe "kpt fn run helloworld --image gcr.io/kustomize-functions/example-tshirt:v0.1.0"
pe "kpt cfg tree helloworld --resources --field 'metadata.annotations.tshirt-size'"

echo ""
p "#  ${bold}change the size from small to large...${normal}"
pe "kpt cfg annotate helloworld --kv tshirt-size=large --kind Deployment"
pe "kpt fn run helloworld --image gcr.io/kustomize-functions/example-tshirt:v0.1.0"
pe "kpt cfg tree helloworld --resources --field 'metadata.annotations.tshirt-size'"
