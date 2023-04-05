#!/bin/bash
# Copyright 2019 The kpt Authors
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

export d=$(pwd)
cd $(mktemp -d)
git init > /dev/null

export PKG=https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@v0.5.0

asciinema rec --overwrite -i 1 -c $d/$1.sh $d/$1.cast
