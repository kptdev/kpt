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

KUBECONFIG=$(CURDIR)/hack/local/kubeconfig
BUILDDIR=$(CURDIR)/.build
CACHEDIR=$(CURDIR)/.cache
DEPLOYCONFIGDIR=$(BUILDDIR)/deploy
MODULES = $(shell find . -path ./.build -prune -o -path ./.cache -prune -o -name 'go.mod' -print)

# GCP project to use for development
export GCP_PROJECT_ID ?= $(shell gcloud config get-value project)
export IMAGE_REPO ?= gcr.io/$(GCP_PROJECT_ID)
export IMAGE_TAG ?= latest

.PHONY: all
all: stop network start-etcd start-kube-apiserver start-function-runner run-local

.PHONY: network
network:
	docker network create --subnet 192.168.8.0/24 porch

.PHONY: stop
stop:
	docker stop kube-apiserver || true
	docker rm kube-apiserver || true
	docker stop etcd || true
	docker rm etcd || true
	docker stop function-runner || true
	docker rm function-runner || true
	docker network rm porch || true

.PHONY: start-etcd
start-etcd:
	docker buildx build -t etcd --output=type=docker hack/local/etcd
	mkdir -p $(BUILDDIR)/data/etcd
	docker stop etcd || true
	docker rm etcd || true
	docker run --detach --user `id -u`:`id -g` \
	  --network=porch \
	  --ip 192.168.8.200 \
	  --name etcd -v $(BUILDDIR)/data/etcd:/data \
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
	  --name kube-apiserver -v $(BUILDDIR)/pki:/pki \
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

.PHONY: start-function-runner
start-function-runner:
	$(MAKE) -C ./func build-image
	docker stop function-runner || true
	docker rm -f function-runner || true
	docker run --detach \
	  --network=porch \
	  --ip 192.168.8.202 \
	  --name function-runner \
	  gcr.io/$(GCP_PROJECT_ID)/function-runner:$(IMAGE_TAG)

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

PORCH = $(BUILDDIR)/porch

.PHONY: run-local
run-local: porch
	KUBECONFIG=$(KUBECONFIG) kubectl apply -f hack/local/localconfig.yaml
	KUBECONFIG=$(KUBECONFIG) kubectl apply -f controllers/pkg/apis/porch/v1alpha1/
	$(PORCH) \
	--secure-port 9443 \
	--standalone-debug-mode \
	--kubeconfig="$(KUBECONFIG)" \
	--cache-directory="$(CACHEDIR)" \
	--function-runner 192.168.8.202:9445

.PHONY: run-jaeger
run-jaeger:
	docker run --rm --name jaeger -d -p4317:55680 -p6831:6831/udp -p16686:16686 jaegertracing/opentelemetry-all-in-one:latest

.PHONY: porch
porch:
	cd apiserver; go build -o $(PORCH) ./cmd/porch

.PHONY: fix-headers
fix-headers:
	# TODO: switch to google/addlicense once we have https://github.com/google/addlicense/pull/104
	go run github.com/justinsb/addlicense@v1.0.1 -c "Google LLC" -l apache --ignore ".build/**" --ignore "**/testdata/**" . 2>/dev/null

.PHONY: fix-all
fix-all: fix-headers fmt tidy

KPTDIR = $(abspath $(CURDIR)/..)
.PHONY: push-images
push-images:
	docker buildx build --push --tag "$(IMAGE_REPO)/porch:$(IMAGE_TAG)" -f "${CURDIR}/hack/Dockerfile" "${KPTDIR}"
	make -C controllers/ push-image
	make -C func/ push-image
	make -C test/ push-image

.PHONY: build-images
build-images:
	docker buildx build --load --tag "$(IMAGE_REPO)/porch:$(IMAGE_TAG)" -f "${CURDIR}/hack/Dockerfile" "${KPTDIR}"
	make -C controllers/ build-image
	make -C func/ build-image
	make -C test/ build-image

.PHONY: apply-dev-config
apply-dev-config:
	# TODO: Replace with KCC (or self-host a registry?)
	gcloud services enable artifactregistry.googleapis.com
	gcloud artifacts repositories describe  --location=us-west1 packages --format="value(name)" || gcloud artifacts repositories create  --location=us-west1 --repository-format=docker packages

	# TODO: Replace with kpt function
	cat config/samples/oci-repository.yaml | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -

	# TODO: Replace with KCC (or self-host a registry?)
	gcloud services enable artifactregistry.googleapis.com
	gcloud artifacts repositories describe  --location=us-west1 deployment --format="value(name)" || gcloud artifacts repositories create  --location=us-west1 --repository-format=docker deployment

	# TODO: Replace with kpt function
	cat config/samples/deployment-repository.yaml | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -

.PHONY: deployment-config
deployment-config:
	rm -rf $(DEPLOYCONFIGDIR) || true
	mkdir -p $(DEPLOYCONFIGDIR)/crd
	cp ./controllers/remoterootsync/config/crd/bases/config.cloud.google.com_remoterootsyncsets.yaml $(DEPLOYCONFIGDIR)/0-remoterootsyncsets.yaml
	cp ./controllers/remoterootsync/config/rbac/role.yaml $(DEPLOYCONFIGDIR)/0-remoterootsync-role.yaml
	cp ./controllers/pkg/apis/porch/v1alpha1/config.porch.kpt.dev_repositories.yaml $(DEPLOYCONFIGDIR)/0-repositories.yaml
	cp ./config/deploy/*.yaml ./config/deploy/Kptfile $(DEPLOYCONFIGDIR)
	kpt fn eval $(DEPLOYCONFIGDIR) --image set-image:v0.1.0 -- \
	  "name=gcr.io/example-google-project-id/function-runner:latest" \
	  "newName=$(IMAGE_REPO)/function-runner" \
	  "newTag=$(IMAGE_TAG)"
	kpt fn eval $(DEPLOYCONFIGDIR) --image set-image:v0.1.0 -- \
	  "name=gcr.io/example-google-project-id/porch-server:latest" \
	  "newName=$(IMAGE_REPO)/porch" \
	  "newTag=$(IMAGE_TAG)"
	kpt fn eval $(DEPLOYCONFIGDIR) --image set-image:v0.1.0 -- \
	  "name=gcr.io/example-google-project-id/porch-controllers:latest" \
	  "newName=$(IMAGE_REPO)/porch-controllers" \
	  "newTag=$(IMAGE_TAG)"
	kpt fn eval $(DEPLOYCONFIGDIR) --image set-annotations:v0.1.4 \
	  --match-api-version=v1 --match-kind=ServiceAccount --match-name=porch-server --match-namespace=porch-system -- \
		"iam.gke.io/gcp-service-account=porch-server@$(GCP_PROJECT_ID).iam.gserviceaccount.com"
	kpt fn eval $(DEPLOYCONFIGDIR) --image set-annotations:v0.1.4 \
	  --match-api-version=v1 --match-kind=ServiceAccount --match-name=porch-controllers --match-namespace=porch-system -- \
		"iam.gke.io/gcp-service-account=porch-sync@$(GCP_PROJECT_ID).iam.gserviceaccount.com"

.PHONY: deploy
deploy: deployment-config
	kubectl apply -R -f $(DEPLOYCONFIGDIR)

.PHONY: push-and-deploy
push-and-deploy: push-images deploy
