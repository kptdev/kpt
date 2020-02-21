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

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.2.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null

# start demo
clear

echo "# start with helloworld package"
echo "$ kpt pkg desc helloworld"
kpt pkg desc helloworld

# start demo
echo " "
p "# 'kpt cfg create-setter' creates a new field setter which can be used to modify field values on the commandline"
pe "kpt cfg tree helloworld --replicas"
pe "kpt cfg create-setter helloworld replicas 5 --set-by 'package-default' --description 'need more than 3'"
pe "kpt cfg list-setters helloworld"

echo " "
p "# use the setter to change the replicas from 5 to 11"
pe "kpt cfg tree helloworld --replicas"
pe "kpt cfg set helloworld replicas 11 --set-by 'phil' --description 'temporarily scale up for increased traffic'"
pe "kpt cfg tree helloworld --replicas"

echo " "
p "# setters can be used to create substitutions"
pe "kpt cfg tree helloworld --image --field metadata.labels.version"
pe "kpt cfg create-setter helloworld version 0.1.0 --set-by 'package-default' --description 'latest release'"
pe "kpt cfg create-subst helloworld version gcr.io/kpt-dev/helloworld-gke:0.1.0 --pattern gcr.io/kpt-dev/helloworld-gke:VERSION_SETTER  --value VERSION_SETTER=version"
pe "kpt cfg list-setters helloworld"


echo " "
p "# use the setter to change the version label and image field"
pe "kpt cfg tree helloworld --field metadata.labels.version --image"
pe "kpt cfg set helloworld version 0.2.0 --set-by 'phil' --description 'production release'"
pe "kpt cfg tree helloworld --field metadata.labels.version --image"


p "# for more information see 'kpt help cfg create-setters'"
p "kpt help cfg create-setters"
