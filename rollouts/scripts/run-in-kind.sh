#! /usr/bin/env bash
# Copyright 2023 The kpt Authors
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

mkdir -p ~/.kube
KUBECONFIG=~/.kube/admin-cluster

# create kind cluster
kind delete cluster --name=rollouts-management-cluster --kubeconfig $KUBECONFIG
kind create cluster --name=rollouts-management-cluster --kubeconfig $KUBECONFIG

# install crds
KUBECONFIG=$KUBECONFIG make install
KUBECONFIG=$KUBECONFIG kubectl apply -f ./manifests/crds/containercluster.yaml

# build and set image
IMG=$IMG make docker-build
kind load docker-image $IMG --name=rollouts-management-cluster
(cd config/manager && $KUSTOMIZE edit set image controller=$IMG)

# deploy
$KUSTOMIZE build ./config/default | sed --expression='s/imagePullPolicy: Always/imagePullPolicy: IfNotPresent/g' | KUBECONFIG=$KUBECONFIG kubectl apply -f -

# wait for controller to be ready
KUBECONFIG=$KUBECONFIG kubectl rollout status deployment rollouts-controller-manager --namespace rollouts-system