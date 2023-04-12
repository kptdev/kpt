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

KPT_FN_RUNTIME="${KPT_FN_RUNTIME:=docker}"

kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1

${KPT_FN_RUNTIME} image inspect gcr.io/kpt-fn/search-replace:v0.1
if [[ $? != 0 ]]; then
    echo "ERR: Image could not be found locally and may not have been pulled"
    exit 1
fi
