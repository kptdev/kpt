#!/bin/bash
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

rm -f ./$1.cast ./$1.gif
asciinema rec -i 1 -c ./$1.sh $1.cast
asciicast2gif -s 2 -S 2 ./$1.cast $1.gif

# copy to gs
gsutil -h "Cache-Control:no-cache,max-age=0" cp -a public-read $1.gif gs://kpt-dev/docs/$1.gif
gsutil -h "Cache-Control:no-cache,max-age=0" cp -a public-read $1.cast gs://kpt-dev/docs/$1.cast
rm ./$1.cast ./$1.gif
