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
. $d/../../../demos/demo-magic/demo-magic.sh

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.5.0
kpt pkg get $PKG helloworld > /dev/null
kpt live init helloworld > /dev/null
kpt live apply --wait-for-reconcile --wait-timeout=2m helloworld > /dev/null

# start demo
clear

echo " "
p "# 'kpt live destroy' -- remove all resources in the package from the cluster"
pe "kpt live destroy helloworld"
