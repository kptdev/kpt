#!/bin/bash

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

set -ex

kubectl apply -f - <<EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  namespace: default
  name: "deployment:myfirstnginx:v1"
spec:
  packageName: myfirstnginx
  revision: v1
  repository: deployment
  tasks:
  - type: clone
    clone:
      upstreamRef:
        type: git
        git:
          repo: https://github.com/GoogleContainerTools/kpt
          ref: v0.7
          directory: package-examples/nginx
EOF

kubectl get packagerevision -n default deployment:myfirstnginx:v1 -oyaml

kubectl get packagerevisionresources -n default deployment:myfirstnginx:v1 -oyaml

