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


docker image inspect gcr.io/kpt-fn/search-replace:v0.1
# if inspect exits with a 0 exit code the image was found locally, remove it
if [[ $? == 0 ]]; then
    docker image rm gcr.io/kpt-fn/search-replace:v0.1
fi
