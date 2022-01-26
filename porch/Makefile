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

.PHONY: all
all: stop network start-etcd start-kube-apiserver run-local

.PHONY: network
network:
	docker network create --subnet 192.168.8.0/24 porch

.PHONY: stop
stop:
	docker stop kube-apiserver || true
	docker rm kube-apiserver || true
	docker stop etcd || true
	docker rm etcd || true
	docker network rm porch || true

.PHONY: start-etcd
start-etcd:
	docker buildx build -t etcd --output=type=docker hack/local/etcd
	mkdir -p .build/data/etcd
	docker stop etcd || true
	docker rm etcd || true
	docker run --detach --user `id -u`:`id -g` \
	  --network=porch \
	  --ip 192.168.8.200 \
	  --name etcd -v `pwd`/.build/data/etcd:/data \
	  etcd --listen-client-urls http://0.0.0.0:2379 --advertise-client-urls http://127.0.0.1:2379

.PHONY: start-kube-apiserver
start-kube-apiserver:
	docker buildx build -t kube-apiserver --output=type=docker hack/local/kube-apiserver
	docker stop kube-apiserver || true
	docker rm kube-apiserver || true
	hack/local/makekeys.sh
	docker run --detach --user `id -u`:`id -g` \
	  --network=porch \
	  --ip 192.168.8.201 \
	  --name kube-apiserver -v `pwd`/.build/pki:/pki \
	  --add-host host.docker.internal:host-gateway \
	  kube-apiserver \
	  --etcd-servers http://etcd:2379 \
	  --secure-port 9444 \
	  --service-account-issuer=https://kubernetes.default.svc.cluster.local \
	  --service-account-key-file=/pki/service-account.pub \
	  --service-account-signing-key-file=/pki/service-account.key \
	  --cert-dir=/pki \
	  --authorization-mode=RBAC \
	  --anonymous-auth=false \
	  --client-ca-file=/pki/ca.crt

MODULES = $(shell find . -path ./forks -prune -o -name 'go.mod' -print)
.PHONY: generate
generate: $(MODULES)
	@for f in $(^D); do (cd $$f; echo "Generating $$f"; go generate -v ./...) || exit 1; done

.PHONY: tidy
tidy: $(MODULES)
	@for f in $(^D); do (cd $$f; echo "Tidying $$f"; go mod tidy) || exit 1; done

.PHONY: test
test: $(MODULES)
	@for f in $(^D); do (cd $$f; echo "Testing $$f"; go test ./...) || exit 1; done

.PHONY: vet
vet: $(MODULES)
	@#for f in $(^D); do (cd $$f; echo "Checking $$f"; go run honnef.co/go/tools/cmd/staticcheck@latest ./...); done
	@for f in $(^D); do (cd $$f; echo "Vetting $$f"; go vet ./...); done

.PHONY: fmt
fmt: $(MODULES)
	@for f in $(^D); do (cd $$f; echo "Formatting $$f"; gofmt -s -w .); done

KUBECONFIG_DOCKER=$(CURDIR)/hack/local/kubeconfig
KUBECONFIG_LOCAL=$(CURDIR)/hack/local/kubeconfig.local

.PHONY: run-local
run-local:
	KUBECONFIG=$(KUBECONFIG_DOCKER) kubectl apply -f hack/local/localconfig.yaml
	KUBECONFIG=$(KUBECONFIG_DOCKER) kubectl apply -f controllers/pkg/apis/porch/v1alpha1/
	cd apiserver; go run ./cmd/porch \
	--secure-port 9443 \
	--standalone-debug-mode \
	--kubeconfig=$(KUBECONFIG_DOCKER) \
	--cache-directory="$(CURDIR)/.cache"

.PHONY: run-jaeger
run-jaeger:
	docker run --rm --name jaeger -d -p4317:55680 -p6831:6831/udp -p16686:16686 jaegertracing/opentelemetry-all-in-one:latest

.PHONY: porch
porch:
	cd apiserver; go build ./cmd/porch

BUILDDIR=$(CURDIR)/.build

.PHONY: start-local-etcd
start-local-etcd:
	etcd \
	  --listen-client-urls http://127.0.0.1:2379 \
	  --advertise-client-urls http://127.0.0.1:2379 \
	  --name porch \
	  --data-dir $(BUILDDIR)/data/porch.etcd

$(BUILDDIR)/kubernetes:
	git clone https://github.com/kubernetes/kubernetes --branch v1.23.1 --depth=1 $(BUILDDIR)/kubernetes

$(BUILDDIR)/kube-apiserver: $(BUILDDIR)/kubernetes
	cd $(BUILDDIR)/kubernetes; git checkout refs/tags/v1.23.1
	cd $(BUILDDIR)/kubernetes; make generated_files
	cd $(BUILDDIR)/kubernetes; CGO_ENABLED=0 go build -o $(BUILDDIR)/kube-apiserver ./cmd/kube-apiserver

.PHONY: start-local-apiserver
start-local-apiserver: $(BUILDDIR)/kube-apiserver
	hack/local/makekeys.sh
	$(BUILDDIR)/kube-apiserver \
	  --etcd-servers http://localhost:2379 \
	  --secure-port 9444 \
	  --service-account-issuer=https://kubernetes.default.svc.cluster.local \
	  --service-account-key-file=$(BUILDDIR)/pki/service-account.pub \
	  --service-account-signing-key-file=$(BUILDDIR)/pki/service-account.key \
	  --cert-dir=$(BUILDDIR)/pki \
	  --authorization-mode=RBAC \
	  --anonymous-auth=false \
	  --client-ca-file=$(BUILDDIR)/pki/ca.crt

.PHONY: start-local-porch
start-local-porch:
	KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl apply -f hack/local/apiservice.local.yaml
	KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl apply -f controllers/pkg/apis/porch/v1alpha1/
	cd apiserver; go run ./cmd/porch \
	--secure-port 9443 \
	--standalone-debug-mode \
	--kubeconfig=$(KUBECONFIG_LOCAL) \
	--cache-directory="$(CURDIR)/.cache"
