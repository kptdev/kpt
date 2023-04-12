#! /bin/bash
# Copyright 2021 The kpt Authors
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


set -eo pipefail

# create a temporary directory for results
results=$(mktemp -d)

kpt fn render -o stdout --results-dir $results \
| kpt fn eval - --image gcr.io/kpt-fn/set-annotations:v0.1.3 --results-dir $results -- foo=bar \
| kpt fn eval - --image gcr.io/kpt-fn/set-labels:v0.1.3 --results-dir $results -- tier=backend

# remove temporary directory
rm -r $results