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

# Build keys for running kube-apiserver
# Based on https://kubernetes.io/docs/tasks/administer-cluster/certificates/

DIR=.build/pki
mkdir -p ${DIR}
cd ${DIR}


if [[ ! -f service-account.pub ]]; then
  openssl genrsa -out service-account.key 2048
  openssl rsa -in service-account.key -pubout -out service-account.pub
fi


cat > csr.conf <<EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
#C = <country>
#ST = <state>
#L = <city>
#O = <organization>
#OU = <organization unit>
CN = 127.0.0.1

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = kubernetes
DNS.2 = kubernetes.default
DNS.3 = kubernetes.default.svc
DNS.4 = kubernetes.default.svc.cluster
DNS.5 = kubernetes.default.svc.cluster.local
IP.1 = 127.0.0.1
#IP.2 = <MASTER_CLUSTER_IP>

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF

if [[ ! -f ca.crt ]]; then
  openssl genrsa -out ca.key 2048
  openssl req -x509 -new -nodes -key ca.key -subj "/CN=127.0.0.1" -days 10000 -out ca.crt
fi

if [[ ! -f apiserver.crt ]]; then
  openssl genrsa -out apiserver.key 2048
  openssl req -new -key apiserver.key -out apiserver.csr -config csr.conf

  openssl x509 -req -in apiserver.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out apiserver.crt -days 10000 \
    -extensions v3_ext -extfile csr.conf
fi

if [[ ! -f admin.crt ]]; then
  openssl genrsa -out admin.key 2048
  openssl req -new -key admin.key -out admin.csr -subj "/CN=admin/O=system:masters" -days 10000

  openssl x509 -req -in admin.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out admin.crt -days 10000 \
    -extensions v3_ext -extfile csr.conf
fi
