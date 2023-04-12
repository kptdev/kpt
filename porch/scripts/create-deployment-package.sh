#!/bin/bash

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

set -ex

kubectl apply -f - <<EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  namespace: default
  name: "deployment:helloserver:v1"
spec:
  packageName: helloserver
  revision: v1
  repository: deployment
  tasks:
  - type: clone
    clone:
      upstreamRef:
        type: git
        git:
          repo: https://github.com/justinsb/kpt
          ref: main_integration
          directory: porch/config/samples/apps/hello-server/k8s
EOF

kubectl get packagerevision -n default deployment:helloserver:v1 -oyaml

kubectl get packagerevisionresources -n default deployment:helloserver:v1 -oyaml

# Update the package in-place
GCP_PROJECT_ID=$(gcloud config get-value project)
kubectl get packagerevisionresources -n default deployment:helloserver:v1 -oyaml | \
  sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | \
  kubectl replace -f -

kubectl get packagerevisionresources -n default deployment:helloserver:v1 -oyaml
