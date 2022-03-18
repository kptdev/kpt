#!/usr/bin/env bash
# Copyright 2022 Google LLC
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

function error() {
    echo "$@"
    exit 1 
}

if [[ $# -ne 2 ]]; then
  error "Invalid # of arguments; ${#}. 2 expected: GIT_DIRECTORY OUTPUT_TAR_FILE"
fi

tar -c -f "${2}" -C "${1}" --owner=porch:0 --group=porch:0 \
  --exclude '.git/logs' \
  --exclude '.git/COMMIT_EDITMSG' \
  --exclude '.git/ORIG_HEAD' \
  .git
