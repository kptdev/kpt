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

########################
# include the magic
########################
. $d/../../demos/demo-magic/demo-magic.sh

kpt pkg get $PKG helloworld > /dev/null
git add . > /dev/null
git commit -m 'fetched helloworld' > /dev/null
kpt svr apply -R -f helloworld > /dev/null


# start demo
clear

echo " "
p "# 'kpt cfg annotate' -- set an annotation on one or more Resources"
pe "kpt cfg tree helloworld --field 'metadata.annotations.foo'"
pe "kpt cfg annotate helloworld --kv foo=bar"
pe "kpt cfg tree helloworld --field 'metadata.annotations.foo'"

echo " "
p "# which Resources are annotated may be filtered by apiVersion, kind, name and namespace"
pe "kpt cfg tree helloworld --field 'metadata.annotations.baz'"
pe "kpt cfg annotate helloworld --kv baz=qux --kind Service"
pe "kpt cfg tree helloworld --field 'metadata.annotations.baz'"

echo " "
p "# multiple annotations may be specified at once"
pe "kpt cfg tree helloworld --field metadata.annotations.a --field metadata.annotations.b"
pe "kpt cfg annotate helloworld --kv a=c --kv b=d"
pe "kpt cfg tree helloworld --field metadata.annotations.a --field metadata.annotations.b"

p "# for more information see 'kpt help cfg annotate'"
p "kpt help cfg annotate"
