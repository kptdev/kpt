#!/usr/bin/env bash
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


ETCD_VER=v3.5.1
DOWNLOAD_URL="https://github.com/etcd-io/etcd/releases/download"

function install-etcd-linux() {
  echo "You will be asked for sudo password ..."
  sudo -v
  curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz | sudo tar -C /usr/local/bin --strip-components 1 -xvz "${DIR}/etcdctl" "${DIR}/etcdutl" "${DIR}/etcd"
}

function install-etcd-darwin() {
  echo "You will be asked for sudo password ..."
  sudo -v
  DIR=$(mktemp -d)

  curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-darwin-amd64.zip -o "${DIR}/etcd-${ETCD_VER}-darwin-amd64.zip"
  unzip -d "${DIR}" "${DIR}/etcd-${ETCD_VER}-darwin-amd64.zip"
  sudo mv ${DIR}/etcd-${ETCD_VER}-darwin-amd64/{etcd,etcdctl,etcdutl} /usr/local/bin
  rm -rf ${DIR}
}

OS=$(uname -s)
case "${OS}" in
  Linux)
    install-etcd-linux
  ;;

  Darwin)
    install-etcd-darwin
  ;;

  *) echo "Unrecognized OS: ${OS}"
     exit 1
  ;;
esac
