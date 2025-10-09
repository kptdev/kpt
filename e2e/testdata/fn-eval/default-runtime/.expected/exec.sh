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

# clear environment variable KRM_FN_RUNTIMETIME if it matches the default
if [ "${KRM_FN_RUNTIMETIME}" = "docker" ]; then
   unset KRM_FN_RUNTIMETIME
fi

echo "KRM_FN_RUNTIMETIME is ${KRM_FN_RUNTIMETIME}"
# run eval with KRM_FN_RUNTIMETIME unset.
kpt fn eval -i ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest -- namespace=staging
