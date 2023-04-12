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

VERSION="3.19.4"
OS=$(uname -s)
MACHINE=$(uname -m)

case "${OS}" in
  Linux)  PLATFORM="linux"
  ;;
  Darwin) PLATFORM="osx"
  ;;
  *) echo "Unrecognized OS: %{OS}"
     exit 1
  ;;
esac

function cleanup() {
    [[ -z "${TMPDIR}" ]] || { rm -rf "${TMPDIR}" || true ; }
}
trap cleanup EXIT

TMPDIR=$(mktemp -d)
ARCHIVE="protoc-${VERSION}-${PLATFORM}-${MACHINE}.zip"
URL="https://github.com/protocolbuffers/protobuf/releases/download/v${VERSION}/${ARCHIVE}"

curl -L "${URL}" -o "${TMPDIR}/${ARCHIVE}" 

echo "You may be asked for sudo password..."
sudo -v
sudo unzip -o -q -d /usr/local "${TMPDIR}/${ARCHIVE}" -x "readme.txt"
cd /usr/local ; sudo unzip -Z1 "${TMPDIR}/${ARCHIVE}" | grep -v "readme.txt" | sudo xargs chmod +r
sudo chmod +rx \
   /usr/local/bin/protoc \
   /usr/local/include/google \
   /usr/local/include/google/protobuf \
   /usr/local/include/google/protobuf/compiler
