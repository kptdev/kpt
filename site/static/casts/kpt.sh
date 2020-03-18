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
. $d/../../demos/demo-magic/demo-magic.sh

cd $(mktemp -d)
git init

# start demo
clear
echo " "
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
p "# 'kpt pkg' -- fetch, update, and sync configuration files using git"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"
pe "tree helloworld"
p "# the package is composed of YAML or JSON files which may be directly applied to a cluster"
pe "less helloworld/deploy.yaml"
git add . && git commit -m 'helloworld package' > /dev/null

pe "clear"
p "# 'kpt cfg' -- examine and modify configuration files"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg tree helloworld --replicas"
pe "kpt cfg set helloworld replicas 3"
pe "kpt cfg tree helloworld --replicas"
p "# the raw YAML or JSON files have been modified by kpt"
pe "git diff"
git commit -a -m 'helloworld package' > /dev/null

pe "clear"
p "# 'kpt fn' --  generate, transform, validate configuration files using containerized functions (run locally)"
pe "kpt cfg tree helloworld --resources"
pe "kpt cfg annotate helloworld --kv tshirt-size=small --kind Deployment"
pe "kpt fn run helloworld --image gcr.io/kustomize-functions/example-tshirt:v0.1.0"
p "# the function set resources on the Deployment using the annotation to determine the size"
pe "kpt cfg tree helloworld --resources"

pe "clear"
p "# kpt is designed to work in collaboration with tools developed by the Kubernetes project itself"
pe "kubectl apply -R -f helloworld"
pe "kubectl get all -o yaml | kpt cfg tree --image --ports"

pe "clear"
p "# kpt works just as well with kustomize as raw config"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-kustomize helloworld-kustomize"
pe "kustomize build helloworld-kustomize/ | kpt cfg fmt"
pe "kubectl apply -k helloworld-kustomize"

p "# for more information see 'kpt help'"
p "kpt help"
