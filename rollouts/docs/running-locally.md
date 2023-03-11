# Running Rollouts Locally

## Prerequisites

To run Rollouts locally, you will need:

* Linux machine (technically it is possible to run Rollouts locally on a Mac but
  due to differences in Docker between Linux and Mac, the Rollouts scripts are
  confirmed to work on Linux)
* [go 1.19](https://go.dev/dl/) or newer
* `make`
* Access to GKE clusters.

## Cluster Setup

### Creating a management cluster with KCC running
First, you must create a config-controller cluster. You can follow the instructions on the 
[config-controller-setup guide](https://cloud.google.com/anthos-config-management/docs/how-to/config-controller-setup) to create a config-controller cluster. 

Make sure your kubeconfig is connected to this cluster:

```sh
KUBECONFIG=~/.kube/admin-cluster gcloud container clusters get-credentials <your cluster> --region <your region> --project <your project>
```

### Provisioning child clusters
The next step will be to provision new child clusters. This example names the child cluster `gke-n`;
you can replace `gke-n` with whatever name you want for your child cluster. 

```sh
# first step is to create a deployable instance of the package
kpt pkg get git@github.com:droot/kpt-packages.git/cluster@main gke-n --for-deployment

# now render the package
kpt fn render

# assuming your kubectl is configured to talk to a management cluster with KCC running (the previous step).
# provision the cluster using following commands
KUBECONFIG=~/.kube/admin-cluster kpt live init gke-n
KUBECONFIG=~/.kube/admin-cluster kpt live apply gke-n
```

You can repeat the above steps to create as many child clusters as you want.

### Setting up kubeconfig
Next, we will configure kubeconfig to be able to talk to each cluster individually.

```sh
# once kpt live status is showing current for a cluster package
# run the following
KUBECONFIG=~/.kube/gke-n gcloud container clusters get-credentials gke-n --region us-west1 --project <your project>

# now ~/.kube/gke-n file has been updated with the credentials for gke-n cluster
# verify if this is working
KUBECONFIG=~/.kube/gke-n kubectl get pods -n kube-system
```

### Set up Config Sync
Rollouts requires a git syncer installed on child clusters. Currently, only Config Sync is supported. 
To install Config Sync on your child clusters, follow these steps.

First, set the release version of Config Sync that you would like to install:

```sh
export CS_VERSION=vX.Y.Z
```

Then, apply the core Config Sync manifests to your cluster:

```sh
KUBECONFIG=~/.kube/gke-n kubectl apply -f "https://github.com/GoogleContainerTools/kpt-config-sync/releases/download/${CS_VERSION}/config-sync-manifest.yaml"
```

Then, for kubernetes version v1.25 and newer, optionally apply the asm.yaml manifest:

```sh
KUBECONFIG=~/.kube/gke-n kubectl apply -f "https://github.com/GoogleContainerTools/kpt-config-sync/releases/download/${CS_VERSION}/acm-psp.yaml"
```

If you wish to install Config Sync from source instead of using a released version, you can follow
the [Config Sync installation guide](https://github.com/GoogleContainerTools/kpt-config-sync/blob/main/docs/installation.md).


## Running Rollouts locally

Clone this repository into `${GOPATH}/src/github.com/GoogleContainerTools/kpt`.

```sh
git clone https://github.com/GoogleContainerTools/kpt.git "${GOPATH}/src/github.com/GoogleContainerTools/kpt"
```

Enter the rollouts directory:

```
cd rollouts
```

Download dependencies:

```sh
make tidy
```

Assuming your kubeconfig is configured to talk to the management cluster with KCC installed (that you created
in the above steps), apply the manifests:

```sh
KUBECONFIG=~/.kube/admin-cluster make install
```

Confirm the CRDs are installed:

```sh
KUBECONFIG=~/.kube/admin-cluster kubectl api-resources | grep gitops.kpt.dev

progressiverolloutstrategies  gitops.kpt.dev/v1alpha1  true  ProgressiveRolloutStrategy
remotesyncs                   gitops.kpt.dev/v1alpha1  true  RemoteSync
rollouts                      gitops.kpt.dev/v1alpha1  true  Rollout
```

Now you are ready to run the controller locally:

```sh
KUBECONFIG=~/.kube/admin-cluster go run main.go
```

## Creating a Rollout object

Here is an example of a simple `Rollout` object:

```yaml
apiVersion: gitops.kpt.dev/v1alpha1
kind: Rollout
metadata:
  name: sample
spec:
  description: sample
  clusters:
    sourceType: KCC
  packages:
    sourceType: GitHub
    github:
      selector:
        org: droot
        repo: store
        directory: namespaces
        revision: main
  targets:
    selector:
      matchLabels:
        location/city: example
  syncTemplate:
    type: RootSync
  packageToTargetMatcher:
    type: AllClusters
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxConcurrent: 2
```

Apply this to your management cluster with `kubectl apply -f`. View the created Rollout, RemoteSyncs, and RootSync objects to verify that the controller is running properly:

```sh
# see the rollouts object
KUBECONFIG=~/.kube/admin-cluster kubectl get rollouts sample

# see the remotesync objects that the rollouts controller created
KUBECONFIG=~/.kube/admin-cluster kubectl get remotesyncs

# see the rootsync object that the remotesync controller created
KUBECONFIG=~/.kube/gke-n kubectl get rootsyncs -nconfig-management-system
```

Deleting the Rollout object should likewise delete the associated RemoteSync and Rootsync objects. You can 
look at the controller logs to verify that the various Remotesync/Rootsync objects are being created, updated,
or deleted.

## Restarting Rollouts

If you make code changes, all you have to do is stop the controller (ctrl+C in the terminal where it is running),
and rerun `go run main.go`.

If you make changes to the Rollouts API, refer to the `Changing Rollouts API` section in the [development guide](./development.md).

## Troubleshooting

### Deleting the Rollout object

The controllers must be running when you delete the Rollout object; otherwise the finalizer will prevent deletion. If you find yourself stuck on deleting a Rollout or RemoteSync object due to an API change or change
in the controller code, you can manually remove the finalizer from the object using `kubectl edit`. 

### API Changes

Make sure you reinstall the CRDs if there are changes to the API. Failure to do so can result in unexpected
behavior in the controllers.
