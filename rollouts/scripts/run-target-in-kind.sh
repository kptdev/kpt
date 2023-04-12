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

KUBECONFIG=/tmp/kubeconfig.yaml
KIND_CONFIG=/tmp/kind-cluster.yaml
MGMTKUBECONFIG=~/.kube/admin-cluster

DEFAULT_NAME=rollouts-target
DEFAULT_NAMESPACE=kind-clusters

# TODO: fetch the latest version of config sync instead of hardcoding it here
DEFAULT_CS_VERSION="v1.14.2"

# set default values
if [ -z "$NAME" ]
then
  NAME=$DEFAULT_NAME
fi

if [ -z "$NAMESPACE" ]
then
  NAMESPACE=$DEFAULT_NAMESPACE
fi

if [ -z "$CS_VERSION" ]
then
  CS_VERSION=$DEFAULT_CS_VERSION
fi

# create the kind cluster and store the kubeconfig
cat <<EOF > $KIND_CONFIG
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "$(hostname -I | awk '{print $1}')"
EOF

kind create cluster --config $KIND_CONFIG --kubeconfig $KUBECONFIG --name $NAME
kind get kubeconfig --name $NAME > $KUBECONFIG

# create the kind-clusters namespace and put the ConfigMap in it
KUBECONFIG=$MGMTKUBECONFIG kubectl create namespace $NAMESPACE
KUBECONFIG=$MGMTKUBECONFIG kubectl create configmap $NAME -n $NAMESPACE --from-file=kubeconfig.yaml=$KUBECONFIG
KUBECONFIG=$MGMTKUBECONFIG kubectl label configmap $NAME -n $NAMESPACE location=example

mkdir -p ~/.kube
cp $KUBECONFIG ~/.kube/$NAME

# install config sync 
KUBECONFIG=~/.kube/$NAME kubectl apply -f "https://github.com/GoogleContainerTools/kpt-config-sync/releases/download/${CS_VERSION}/config-sync-manifest.yaml"
