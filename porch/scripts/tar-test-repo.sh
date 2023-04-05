#!/usr/bin/env bash
# Copyright 2022 The kpt Authors
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

GIT_DIRECTORY="${1}"
OUTPUT_TAR_FILE="${2}"

if [ -d "${GIT_DIRECTORY}" ]; then

  tar -c -v -f "${OUTPUT_TAR_FILE}" -C "${GIT_DIRECTORY}" --owner=porch:0 --group=porch:0 \
    --sort=name --mtime='PST 2022-02-02' \
    --exclude '.git/logs' \
    --exclude '.git/COMMIT_EDITMSG' \
    --exclude '.git/ORIG_HEAD' \
    --exclude '.git/info/exclude' \
    .git

else
  error "${GIT_DIRECTORY} doesn't exist"
fi
