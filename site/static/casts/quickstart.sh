#!/bin/bash
# Copyright 2021 Google LLC
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

# start demo
clear

echo " "
p "# First, let’s fetch the kpt package from Git to your local filesystem."
pe "kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/nginx@v0.4"
pe "cd nginx"

echo " "
p "# Next, let’s quickly view the content of the package."
pe "kpt pkg tree"

echo " "
p "# Initialize a local Git repo and commit the forked copy of the package."
pe "git init; git add .; git commit -m 'Pristine nginx package'"

echo " "
p "# Often, you want to automatically mutate and/or validate resources in a package. kpt fn commands enable you to execute programs called kpt functions."
pe "kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-path=spec.**.app' 'put-value=my-nginx'"
pe "git diff"

echo " "
p "# kpt live commands provide the functionality for deploying packages to a Kubernetes cluster."
pe "kpt live init"
pe "kpt live install-resource-group"
pe "kpt live apply --dry-run"
pe "kpt live apply --reconcile-timeout=15m"

echo " "
p "# At some point, there will be a new version of the upstream nginx package you may want to use."
pe "git add .; git commit -m 'My customizations'"
pe "kpt pkg update @v0.5"
pe "kpt live apply --reconcile-timeout=15m"

echo " "
p "# If you need to delete your package from the cluster,"
pe "kpt live destroy"
