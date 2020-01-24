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
git init > /dev/null

export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null


# start demo
clear

echo "# start with helloworld package"
echo "$ kpt pkg desc helloworld"
kpt pkg desc helloworld

echo " "
p "# 'kpt cfg count' prints Resource counts grouped by kind"
pe "kpt cfg count helloworld"

echo " "
p "# count can summarize Resource counts from other sources using stdin"
pe "kubectl get all -o yaml | kpt cfg count"

echo " "
p "# count can be useful for understanding larger repos"
pe "kpt pkg get https://github.com/kubernetes/examples/staging examples"
pe "kpt cfg count examples"

echo " "
p "# count can also print the total Resource count without grouping by kind"
pe "kpt cfg count examples --kind=false"

p "# for more information see 'kpt help cfg count'"
p "kpt help cfg count"
