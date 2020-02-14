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

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0
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
p "# 'kpt cfg create-setter' creates a new field setter by annotating Resource fields"
pe "kpt cfg tree helloworld --replicas"
pe "kpt cfg create-setter helloworld replicas 5 --field replicas --type integer"
pe "kpt cfg list-setters helloworld"

echo " "
p "# setters can be created for a partial field value as a form of substitution"
p "# partial setter values must be unique strings within the field"
pe "kpt cfg tree helloworld --image"
pe "kpt cfg create-setter helloworld version 0.1.0 --field image --partial --type string"
pe "kpt cfg list-setters helloworld"

echo " "
p "# multiple full and partial fields on multiple objects may be set using a single setter"
pe "kpt cfg tree helloworld --field metadata.labels.version"
pe "kpt cfg create-setter helloworld version 0.1.0 --field version --type string"
pe "kpt cfg list-setters helloworld"

echo " "
p "# demo using the setters"
pe "kpt cfg tree helloworld --field metadata.labels.version --replicas --image"
pe "kpt cfg set helloworld version 0.2.0"
pe "kpt cfg set helloworld replicas 11"
pe "kpt cfg tree helloworld --field metadata.labels.version --replicas --image"

echo " "
p "# setter values include a description and set-by for consumers"
pe "kpt cfg create-setter helloworld service-type LoadBalancer --field type  --type string --description 'external traffic' --set-by 'package-default'"
pe "kpt cfg list-setters helloworld"

p "# for more information see 'kpt help cfg create-setters'"
p "kpt help cfg create-setters"
