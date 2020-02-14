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

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null

# start demo
clear

# start demo
echo "# 'kpt cfg' -- programmatically modify and view Resource configuration"
echo "# (tutorial uses package-examples/helloworld-set@v0.1.0)"

echo " "
p "# kpt cfg cat -- print the raw package contents"
pe "kpt cfg cat helloworld"

echo " "
p "# kpt cfg set -- list and set fields"
pe "kpt cfg list-setters helloworld"
pe "kpt cfg set helloworld replicas 3"
pe "git diff"

echo " "
p "# kpt cfg tree -- print the package with a tree structure"
pe "kpt cfg tree helloworld --name --image --replicas"

echo " "
p "# kpt cfg grep -- filter Resources"
pe "kpt cfg grep \"kind=Service\" helloworld | kpt cfg tree"

echo " "
p "# kpt cfg fmt -- order fields in yaml files"
pe "kpt cfg fmt helloworld"

echo " "
p "# kpt cfg count -- print the Resource counts"
pe "kpt cfg count helloworld"

p "# for more information see 'kpt help cfg'"
p "kpt help cfg"
