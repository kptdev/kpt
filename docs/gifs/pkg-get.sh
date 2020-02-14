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
p "# 'kpt pkg get' fetches a directory of configuration from a remote git repository subdirectory"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0 helloworld"
pe "tree ."
pe "cat helloworld/deploy.yaml"
pe "cat helloworld/service.yaml"

echo " "
p "# the same remote package may be fetched to multiple different local copies"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.2.0 helloworld2"
pe "tree ."

echo " "
pe "kpt pkg desc *"

echo " "
echo "rm -rf helloworld helloworld2"
rm -rf helloworld helloworld2
p "# 'kpt pkg get' can fetch arbitrary nested subdirectories from a repo as packages"
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples examples"
pe "tree examples"

echo " "
p "# subdirectories may be versioned independently from one another by including the subdirectory as part of the tag"
pe "git ls-remote https://github.com/GoogleContainerTools/kpt | grep /package-examples/"
pe "# when resolving a version, first a tag matching the 'subdirectory/version'"
pe "# is matched, and if it is not found a tag matching 'version' is matched"

echo " "
p "# for more information see 'kpt help pkg get'"
p "kpt help pkg get"
