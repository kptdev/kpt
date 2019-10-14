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


export TYPE_SPEED=500

mkdir package-explore-demo
cd package-explore-demo
git init .

kpt get https://github.com/kubernetes/examples ./examples
git add . && git commit -m 'fetch examples'

########################
# include the magic
########################
. $HOME/demo-magic/demo-magic.sh

# hide the evidence
clear

# Checkout the package
pe "echo 'cloned package https://github.com/kubernetes/examples to examples/'"
pe "tree examples"

# View Full Package
pe "echo 'view the full package contents'"
pe "kpt cat examples/ | less"
pe "kpt rc --kind=false examples/"
pe "kpt rc examples/ | less"
pe "kpt tree examples | less"
pe "clear"

pe "echo 'explore the package contents'"
pe "echo 'show all resources with more than 3 replicas'"
pe "kpt grep 'spec.replicas>3' examples/ | kpt tree --name --replicas"
pe "clear"

pe "echo 'show all resources with .3 or more cpu limits'"
pe "kpt grep 'spec.template.spec.containers[name=\.*].resources.limits.cpu>=300m' examples/ | kpt tree --resources --name --resources --image | less"

pe "echo 'show all resources without cpu limits specified'"
pe "kpt grep 'spec.template.spec.containers' examples/ | kpt grep 'spec.template.spec.containers[name=\.*].resources.limits.cpu>=0' -v | kpt grep 'spec.template.spec.containers[name=\.*].resources.requests.cpu>=0' -v | kpt tree --resources --name  | less"
pe "clear"

pe "echo 'show all resources with containers without an image tag'"
pe "kpt grep 'spec.template.spec.containers[name=\.*].name=\.*' examples/ |  kpt grep 'spec.template.spec.containers[name=\.*].image=\.*:\.*' -v | kpt tree --image --name | less"
pe "clear"

pe "echo 'show all Services with their ports'"
pe "kpt grep 'kind=Service' examples/ | kpt grep 'spec.ports' | kpt tree --ports"

# End of demo
pe ""
