# Running Rollouts Locally

## Prerequisites

To run Rollouts locally, you will need:

* Linux machine (technically it is possible to run Rollouts locally on a Mac but
  due to differences in Docker between Linux and Mac, the Rollouts scripts are
  confirmed to work on Linux)
* [go 1.21](https://go.dev/dl/) or newer
* `make`
* Either access to GKE clusters, or Kind.

This doc will go through:

- [Running in kind](#running-in-kind): How to run the controller and target clusters as Kind clusters.
- [Running locally with a KCC Cluster](#running-the-controller-locally-with-a-kcc-management-cluster): How to run the controller locally while connected to a KCC management cluster and target clusters.

There are also sample Rollout objects for each.

---

## Running in Kind

### Creating a management cluster 
To spin up a admin Kind cluster with the rollouts controller, run:

```sh
make run-in-kind`
```

This will create a new kind cluster for you called `rollouts-management-cluster` with the rollouts
controller manager running and relevant CRDs installed. It will store the kubeconfig for the
new kind cluster in `~/.kube/admin-cluster`.

Verify that the admin cluster has been created correctly with your kubeconfig setup:

```sh
KUBECONFIG=~/.kube/admin-cluster kubectl get nodes
NAME                                        STATUS   ROLES           AGE   VERSION
rollouts-management-cluster-control-plane   Ready    control-plane   57s   v1.25.3
```

Verify the CRDs are installed:

```sh
$ KUBECONFIG=~/.kube/admin-cluster kubectl api-resources | grep gitops
progressiverolloutstrategies                                                 gitops.kpt.dev/v1alpha1                   true         ProgressiveRolloutStrategy
remotesyncs                                                                  gitops.kpt.dev/v1alpha1                   true         RemoteSync
rollouts                                                                     gitops.kpt.dev/v1alpha1                   true         Rollout
```

Verify the controller is running:

```sh
$ KUBECONFIG=~/.kube/admin-cluster kubectl get pods -nrollouts-system
NAME                                           READY   STATUS    RESTARTS   AGE
rollouts-controller-manager-7f556b8667-6kzbl   2/2     Running   0          20s
```

### Restarting the management controller

If you make code changes, all you have to do is rerun `make run-in-kind`.

### Creating the target clusters

To create a Kind target cluster, run:

```sh
make run-target-in-kind
```

This will spin up a new Kind cluster and install Config Sync to it. It will also create
a ConfigMap representation of the Kind cluster in the `kind-clusters` namespace, and the ConfigMap will have 
a sample label `location: example` as well as the kubeconfig for the new Kind cluster. It will also store the new Kind
cluster's kubeconfig in `~/.kube/$NAME`.

The default name for the cluster `rollouts-target`, and default Config Sync version installed is `v1.14.2`.

Verify that the target cluster has been created correctly with your kubeconfig setup:

```sh
KUBECONFIG=~/.kube/rollouts-target kubectl get nodes
NAME                           STATUS   ROLES           AGE   VERSION
rollouts-target-control-plane   Ready    control-plane   24m   v1.25.3
```

Verify that Config Sync was installed:

```sh
KUBECONFIG=~/.kube/rollouts-target kubectl api-resources | grep configsync
reposyncs                                      configsync.gke.io/v1beta1              true         RepoSync
rootsyncs                                      configsync.gke.io/v1beta1              true         RootSync
```

You can optionally specify a name for the target cluster, and a version of Config Sync that you want installed:
```sh
NAME=target CS_VERSION=vX.Y.Z make run-target-in-kind
```

### Creating a Rollout object

Here is an example of a simple `Rollout` object which uses Kind clusters:

```yaml
apiVersion: gitops.kpt.dev/v1alpha1
kind: Rollout
metadata:
  name: sample
spec:
  description: sample
  clusters:
    sourceType: Kind
    kind:
      namespace: kind-clusters
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
        location: example
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
KUBECONFIG=~/.kube/rollouts-target kubectl get rootsyncs -nconfig-management-system
```

Deleting the Rollout object should likewise delete the associated RemoteSync and Rootsync objects. You can 
look at the controller logs to verify that the various Remotesync/Rootsync objects are being created, updated,
or deleted.

---

## Running the controller locally with a KCC Management cluster.

### Creating a management cluster with KCC running
First, you must create a config-controller cluster. You can follow the instructions on the 
[config-controller-setup guide](https://cloud.google.com/anthos-config-management/docs/how-to/config-controller-setup) to create a config-controller cluster. 

Make sure your kubeconfig is connected to this cluster:

```sh
KUBECONFIG=~/.kube/admin-cluster gcloud container clusters get-credentials <your cluster> --region <your region> --project <your project>
```

### Provisioning target clusters
The next step will be to provision new target clusters. This example names the target cluster `gke-n`;
you can replace `gke-n` with whatever name you want for your target cluster. 

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

You can repeat the above steps to create as many target clusters as you want.

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
Rollouts requires a git syncer installed on target clusters. Currently, only Config Sync is supported. 
To install Config Sync on your target clusters, follow these steps.

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


### Running Rollouts controller locally

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

### Creating a Rollout object

Here is an example of a simple `Rollout` object which uses KCC:

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

### Restarting the local controller

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
