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
p "# 'kpt cfg create-subst' creates a new field substitution derived from setters"
p "# first, print the field we want to substitute"
pe "kpt cfg tree helloworld --image"

p "# second, create a setter for the version, restrict it only to the image field"
pe "kpt cfg create-setter helloworld version 0.1.0 --set-by 'package-default' --description 'latest release' --field 'image'"

p "# finally, create the substitution which replaces a marker with the setter value"
pe "kpt cfg create-subst helloworld version gcr.io/kpt-dev/helloworld-gke:0.1.0 --pattern gcr.io/kpt-dev/helloworld-gke:VERSION_SETTER  --value VERSION_SETTER=version"

echo " "
p "# the setter and substitution metadata was written to the Kptfile"
pe "cat helloworld/Kptfile"

echo " "
p "# the substitution is referenced by the image field"
pe "less helloworld/deploy.yaml"

echo " "
p "# use the setter to change the tag on the image field"
pe "kpt cfg tree helloworld --image"
pe "kpt cfg set helloworld version 0.2.0 --set-by 'phil' --description 'production release'"
pe "kpt cfg tree helloworld --image"

p "# for more information see 'kpt help cfg create-subst'"
p "kpt help cfg create-subst"
