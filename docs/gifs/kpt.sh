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
. ../../demos/demo-magic/demo-magic.sh

cd $(mktemp -d)
git init

# start demo
clear
echo " "
export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
p "# 'kpt pkg' commands fetch and update configuration packages from git repos"
pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"
echo "$ git add . && git commit -m 'helloworld package'"
git add . && git commit -m 'helloworld package' > /dev/null

echo " "
p "# 'kpt cfg' commands display and modify local configuration files"
pe "kpt cfg tree helloworld --replicas --image"
pe "kpt cfg set helloworld replicas 3"
pe "git diff -c"
git commit -a -m 'helloworld package' > /dev/null
echo "$ git commit -a -m 'helloworld package'"

echo " "
p "# 'kpt fn' commands generate, transform and validate configuration"
p "# using functions packaged in containers (run locally)"
pe "kpt cfg tree helloworld --resources"
pe "kpt cfg annotate helloworld --kv tshirt-size=small --kind Deployment"
pe "kpt fn run helloworld --image gcr.io/kustomize-functions/example-tshirt:v0.1.0"
pe "kpt cfg tree helloworld --resources"

echo " "
p "# kpt is designed to integrate with tools developed by the Kubernetes project"
p "# such as kubectl and kustomize"
pe "kubectl apply -R -f helloworld"
pe "kubectl get all -o yaml | kpt cfg tree --image --ports"

pe "kpt pkg get git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-kustomize helloworld-kustomize"
pe "kustomize build helloworld-kustomize/ | kpt cfg fmt"
pe "kubectl apply -k helloworld-kustomize"

p "# for more information see 'kpt help'"
p "kpt help"
