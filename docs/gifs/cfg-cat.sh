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

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.1.0
kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null

# start demo
clear

echo " "
p "# 'kpt cfg cat' prints raw Resources from a package"
pe "kpt cfg cat helloworld"

echo " "
p "# by default, cat does not print Resources annotated with config.kubernetes.io/local-config."
p "# To show local-config use the flags --exclude-non-local and --include-local."
pe "kpt cfg cat helloworld --exclude-non-local --include-local"

p "# for more information see 'kpt help cfg cat'"
p "kpt help cfg cat"
