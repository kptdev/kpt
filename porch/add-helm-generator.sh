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

kubectl delete packagerevision -n default --field-selector=spec.revision=v1,spec.packageName=postgres,spec.repository=deployment || true

kubectl apply -f - <<EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  namespace: default
  name: "deployment:postgres:v1"
spec:
  packageName: postgres
  revision: v1
  repository: deployment
  tasks:
  - type: clone
    clone:
      generator:
        config:
          apiVersion: fn.kpt.dev/v1alpha1
          kind: RenderHelmChart
          helmCharts:
          - chartArgs:
              name: postgresql
              repo: https://charts.bitnami.com/bitnami
            templateOptions:
              values:
                valuesInline:
                  postgresqlDataDir: /kpt/postgresql/data
  - type: eval
    eval:
      image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        bucket-label: bucket-label-value
        another-label: another-label-value
  - type: eval
    eval:
      image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        porch: rocks
EOF

kubectl get packagerevision -n default --field-selector=spec.revision=v1,spec.packageName=postgres,spec.repository=deployment -oyaml

kubectl get packagerevisionresources -n default --field-selector=spec.revision=v1,spec.packageName=postgres,spec.repository=deployment -oyaml
