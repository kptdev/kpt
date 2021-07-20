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

# don't add licenses to the site directory, it will break the docs
# and will add them to the theme which is a submodule (bad)
which addlicense || go get github.com/google/addlicense
find . -type f \
! -path "./site/*" \
! -path "./docs/*" \
! -path "**/.expected/results.yaml" \
! -path "**/generated/*" \
| xargs $GOBIN/addlicense -y 2021 -l apache
