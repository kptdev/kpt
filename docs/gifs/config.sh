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

stty rows 80 cols 30

export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0

# start demo
clear
echo "$ export PKG=git@github.com:GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0"
pe "kpt pkg get \$PKG helloworld"
pe "git add . && git commit -m 'fetched helloworld'"

# start demo
echo "#"
echo "# view the package"
echo "#"
pe "kpt config count helloworld"
pe "kpt config cat helloworld | less"
pe "kpt config tree helloworld --name --image --replicas"
pe "kpt config grep \"kind=Service\" helloworld | kpt config tree --name --image --replicas"

echo ""
echo ""
echo "#"
echo "# modify the package"
echo "#"
pe "kpt config list-setters helloworld replicas"
pe "kpt config set helloworld replicas 3"
pe "kpt config list-setters helloworld replicas"
pe "git diff"
pe "git add . && git commit -m 'change replicas to 3'"

pe "clear"
