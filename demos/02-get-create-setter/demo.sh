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

# start demo
echo ""
p "# fetch the package and add to git..."
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0 helloworld"
pe "git add . && git commit -m 'helloworld'"

echo ""
p "#  print the package contents..."
pe "kpt cfg tree helloworld --all"

echo ""
p "#  create a setter for the Service port protocol..."
pe "kpt cfg create-setter helloworld http-protocol TCP --field protocol --type string --kind Service"

echo ""
p "#  list the setters -- the one we just created should be listed..."
pe "kpt cfg list-setters helloworld"

echo ""
p "#  change the Service port protocol using the setter..."
pe "kpt cfg set helloworld http-protocol UDP --description 'justification for UDP' --set-by 'phil'"

echo ""
p "#  observe the updated configuration..."
pe "kpt cfg list-setters helloworld"
pe "kpt cfg tree helloworld --ports"
pe "git diff"
