# Installing Porch

You can install Porch by either using one of the
[released versions](https://github.com/GoogleContainerTools/kpt/releases), or
building Porch from sources.

## Prerequisites

To run one of the [released versions](https://github.com/GoogleContainerTools/kpt/releases)
of Porch on GKE, you will need:

* A [GCP Project](https://console.cloud.google.com/projectcreate)
* [gcloud](https://cloud.google.com/sdk/docs/install)
* [kubectl](https://kubernetes.io/docs/tasks/tools/); you can install it via
  `gcloud components install kubectl`
* [kpt](https://kpt.dev/)
* Command line utilities such as `curl`, `tar`

To build and run Porch on GKE, you will also need:

* A container registry which will work with your GKE cluster.
  [Artifact Registry](https://console.cloud.google.com/artifacts)
  or [Container Registry](https://console.cloud.google.com/gcr) work well
  though you can use others too.
* [go 1.17](https://go.dev/dl/) or newer
* [docker](https://docs.docker.com/get-docker/)
* [Configured docker credential helper](https://cloud.google.com/sdk/gcloud/reference/auth/configure-docker)
* [git](https://git-scm.com/)
* [make](https://www.gnu.org/software/make/)

## Getting Started

Make sure your `gcloud` is configured with your project (alternatively, you can
augment all following `gcloud` commands below with `--project` flag):

```sh
gcloud config set project YOUR_GCP_PROJECT
```

Select a GKE cluster or create a new one:

```sh
gcloud container clusters create-auto --region us-central1 porch-dev
```

**Note:** For development of Porch, in particular for running Porch tests,
Standard GKE cluster is currently preferable. Select a
[GCP region](https://cloud.google.com/compute/docs/regions-zones#available)
 that works best for your needs:

 ```sh
gcloud container clusters create --region us-central1 porch-dev
```

And ensure `kubectl` is targeting your GKE cluster:

Ensure you are targeting the GKE cluster. Make sure to substitute your
specific region and cluster name:

```sh
gcloud container clusters get-credentials --region us-central1 porch-dev
```

## Run Released Version of Porch

To run a released version of Porch, download the release config bundle from
[Porch release page](https://github.com/GoogleContainerTools/kpt/releases).

Untar and applly the config bundle. This will install:

* Porch server
* [Config Sync](https://cloud.google.com/anthos-config-management/docs/config-sync-overview)

```sh
mkdir porch-install
tar xzf path-to-deployment-blueprint.tar.gz -C porch-install
kubectl apply -f porch-install
```

You can verify that Porch is running by querying the `api-resources`:

```sh
kubectl api-resources | grep porch
```
Expected output will include:

```
repositories                                   config.porch.kpt.dev/v1alpha1          true         Repository
functions                                      porch.kpt.dev/v1alpha1                 true         Function
packagerevisionresources                       porch.kpt.dev/v1alpha1                 true         PackageRevisionResources
packagerevisions                               porch.kpt.dev/v1alpha1                 true         PackageRevision
```

You can start [using Porch](./porch-user-guide.md).

## Run Custom Build of Porch

To run custom build of Porch, you will need additional [prerequisites](#prerequisites).
The commands below use [Google Container Registry](https://console.cloud.google.com/gcr).

Clone this repository into `${GOPATH}/src/github.com/GoogleContainerTools/kpt`.

```sh
git clone https://github.com/GoogleContainerTools/kpt.git "${GOPATH}/src/github.com/GoogleContainerTools/kpt"
```

[Configure](https://cloud.google.com/sdk/gcloud/reference/auth/configure-docker)
docker credential helper for your repository

If your use case doesn't require Porch to interact with GCP container registries,
you can build and deploy Porch by running the following command. It will build and
push Porch Docker images into (by default) Google Container Registry named (example
shown is the Porch server image):

`gcr.io/YOUR-PROJECT-ID/porch-server:SHORT-COMMIT-SHA`


```sh
IMAGE_TAG=$(git rev-parse --short HEAD) make push-and-deploy-no-sa
```

If you want to use different repository, you can set `IMAGE_REPO` variable
(see [Makefile](https://github.com/GoogleContainerTools/kpt/blob/main/porch/Makefile#L28)
for details).

The `make push-and-deploy-no-sa` target will install Porch but not Config Sync.
You can install Config Sync in your k8s cluster manually following the
[documentation](https://cloud.google.com/anthos-config-management/docs/how-to/installing-kubectl).

**Note**: The `-no-sa` (no service account) targets create Porch deployment
configuration which does not associate Kubernetes service accounts with GCP
service accounts. This is sufficient for Porch to integate with Git repositories
using Basic Auth, for example GitHub.

As above, you can verify that Porch is running by querying the `api-resources`:

```sh
kubectl api-resources | grep porch
```

And start [using Porch](./porch-user-guide.md) if the Porch resources are
available.

### Workload Identity

To integrate with OCI repositories such as
[Artifact Registry](https://console.cloud.google.com/artifacts) or
[Container Registry](https://console.cloud.google.com/gcr), Porch relies on
[workload identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity).

For that use case, create service accounts and assign roles:

```sh
GCP_PROJECT_ID=$(gcloud config get-value project)

# Create GCP service account for Porch server.
gcloud iam service-accounts create porch-server
# Create GCP service account for Porch sync controller.
gcloud iam service-accounts create porch-sync

# We want to create and delete images. Assign IAM roles to allow repository
# administration.
gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
  --member "serviceAccount:porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
  --role "roles/artifactregistry.repoAdmin"

gcloud iam service-accounts add-iam-policy-binding porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/porch-server]"

gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
  --member "serviceAccount:porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
  --role "roles/artifactregistry.reader"

gcloud iam service-accounts add-iam-policy-binding porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/porch-controllers]"
```

Build Porch, push images, and deploy porch server and controllers using the
`make` target that adds workload identity service account annotations:

```sh
IMAGE_TAG=$(git rev-parse --short HEAD) make push-and-deploy
```

As above, you can verify that Porch is running by querying the `api-resources`:

```sh
kubectl api-resources | grep porch
```

And start [using Porch](guides/porch-user-guide.md) if the Porch resources are
available.

## Deploying Configuration

**Note:** Rather than using Config Sync, this section uses a prototype sync
controller.

Create some example repositories / packages:

```sh
# Create artifact-registry repos etc
make apply-dev-config
# Push a sample hello-world app
make -C config/samples/apps/hello-server push-image
# Create a package for the sample hello-world app
./scripts/create-deployment-package.sh
```

To test out remoterootsync self-applying:

```sh
# Grant more RBAC permissions than are normally needed (equivalent to admin permissions)
kubectl apply -f controllers/remoterootsync/config/samples/hack-self-apply-rbac.yaml

# Apply the RemoteRootSyncSet
cat controllers/remoterootsync/config/samples/hack-self-apply.yaml \
  | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -
```
