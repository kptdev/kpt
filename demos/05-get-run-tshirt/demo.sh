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

# hide the evidence
git init
clear

# Put your stuff here
bold=$(tput bold)
normal=$(tput sgr0)
stty rows 50 cols 180

# start demo
echo ""
echo "  ${bold}fetch the package...${normal}"
pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-tshirt@v0.1.0 helloworld"

echo ""
echo "  ${bold}print the package resources...${normal}"
pe "config tree helloworld --resources --field 'metadata.annotations.tshirt-size'"

echo ""
echo "  ${bold}print the sizer...${normal}"
pe "cat helloworld/helloworld.yaml"

echo ""
echo "  ${bold}change the tshirt-size...${normal}"
pe "config set helloworld"
pe "config set helloworld tshirt-size large --description 'need lots of resources' --set-by phil"
pe "config set helloworld"
pe "config tree helloworld --resources --field 'metadata.annotations.tshirt-size'"
pe "config run helloworld"

echo ""
echo "  ${bold}print the updated package resources...${normal}"
pe "config tree helloworld --resources --field 'metadata.annotations.tshirt-size'"
