// Copyright 2021 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package live

var (
	kptFile = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
upstreamLock:
  type: git
  git:
    commit: 786b898857bd7e9647c229d5f39b0be4de86c915
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
inventory:
  namespace: test-namespace
  name: inventory-obj-name
  inventoryID: XXXXXXXXXX-FOOOOOO
`
	kptFileWithPipeline = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: kptfileWithPipeline
pipeline:
  mutators:
  - image: k8s.gcr.io/pause:latest
    configPath: cm.yaml
`
	podA = `
apiVersion: v1
kind: Pod
metadata:
  name: pod-a
  namespace: test-namespace
  labels:
    name: test-pod-label
spec:
  containers:
  - name: kubernetes-pause
    image: k8s.gcr.io/pause:2.0
`
	deploymentA = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: test-deployment
spec:
  replicas: 1
`
	configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
data: {}
`
	crd = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: custom.io
spec:
  conversion:
    strategy: None
  group: custom.io
  names:
    kind: Custom
    listKind: CustomList
    plural: customs
    singular: custom
  scope: Cluster
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        description: This is for testing
        type: object
    served: true
    storage: true
    subresources: {}
`
	cr = `
apiVersion: custom.io/v1
kind: Custom
metadata:
  name: cr
`
	localConfig = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
  annotations:
    config.kubernetes.io/local-config: "true"
data: {}
`
	notLocalConfig = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
  annotations:
    config.kubernetes.io/local-config: "false"
data: {}
`
)
