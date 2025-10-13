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

KRM_FN_RUNTIMETIME="${KRM_FN_RUNTIMETIME:=docker}"

# Function ghcr.io/kptdev/krm-functions-catalog/foo:v0.1 prints "foo" to stderr and
# function ghcr.io/kptdev/krm-functions-catalog/bar:v0.1 prints "bar" to stderr.
# We intentionally tag a wrong image as ghcr.io/kptdev/krm-functions-catalog/foo:v0.1, since we
# expect the correct image to be pulled and override the wrong image.
${KRM_FN_RUNTIMETIME} pull ghcr.io/kptdev/krm-functions-catalog/bar:v0.1
${KRM_FN_RUNTIMETIME} tag ghcr.io/kptdev/krm-functions-catalog/bar:v0.1 ghcr.io/kptdev/krm-functions-catalog/foo:v0.1
