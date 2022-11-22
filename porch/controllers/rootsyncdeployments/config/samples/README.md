# Simple example

This example adds a RootSyncDeployment with name `simple` that will deploy the referenced
PackageRevision to all GKE or Config Controller clusters with the `foo: bar` label. When the
version of the PackageRevision is changed, the clusters will be updated progressively.

## Setup

### Clusters
Create clusters with Config Connector using the following two manifests:
```
apiVersion: container.cnrm.cloud.google.com/v1beta1
kind: ContainerCluster
metadata:
  name: gke-one
  namespace: config-control
spec:
  location: us-central1
  initialNodeCount: 1
  workloadIdentityConfig:
    workloadPool: ${PROJECT-ID}.svc.id.goog
```

```
apiVersion: container.cnrm.cloud.google.com/v1beta1
kind: ContainerCluster
metadata:
  name: gke-two
  namespace: config-control
spec:
  location: us-central1
  initialNodeCount: 1
  workloadIdentityConfig:
    workloadPool: ${PROJECT-ID}.svc.id.goog
```

Install Config Management through the Google Cloud console to make sure
Config Sync is available in the cluster.

### Porch package
Make sure you have a repository registered with Porch (this example assumes the repository is named `blueprint`). Create a new package `foo`, add a ConfigMap to the package, and publish it:
```
kpt alpha rpkg init foo --repository=blueprint -n default --workspace=foo
kpt alpha rpkg pull blueprint-16f93511a8fd4774c928e09a7e135f9852161f74 -n default ./pull
cat <<EOF >>./pull/cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: foo
  namespace: default
data:
  cm: foo
EOF
kpt alpha rpkg push blueprint-16f93511a8fd4774c928e09a7e135f9852161f74 -n default ./pull
kpt alpha rpkg propose blueprint-16f93511a8fd4774c928e09a7e135f9852161f74 -n default
kpt alpha rpkg approve blueprint-16f93511a8fd4774c928e09a7e135f9852161f74 -n default
rm -fr ./pull
```

Then create a new revision of the package and publish it.
```
kpt alpha rpkg edit blueprint-16f93511a8fd4774c928e09a7e135f9852161f74 -n default --workspace foo2
kpt alpha rpkg pull blueprint-93dabbfa9de63b507d13d366185a70ee2ed087f7 -n default ./pull
cat <<EOF >./pull/cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: foo2
  namespace: default
data:
  cm: foo2
EOF
kpt alpha rpkg push blueprint-93dabbfa9de63b507d13d366185a70ee2ed087f7 -n default ./pull
kpt alpha rpkg propose blueprint-93dabbfa9de63b507d13d366185a70ee2ed087f7 -n default
kpt alpha rpkg approve blueprint-93dabbfa9de63b507d13d366185a70ee2ed087f7 -n default
rm -fr ./pull
```

## Running the example
Update the name of the PackageRevision in the `simple.yaml` file to point to the PackageRevision
you want to deploy (if you ended up with different names than the example)

Label the first GKE cluster with the required labels:
```
kubectl -n config-control label containerclusters.container.cnrm.cloud.google.com gke-one foo=bar
```

Apply the RootSyncDeployment to install the PackageRevision in the gke-one cluster:
```
kubectl apply -f simple.yaml
```

Watch the progress:
```
k get rootsyncdeployments.config.porch.kpt.dev simple -oyaml -w
```

Label the second cluster as well:
```
kubectl -n config-control label containerclusters.container.cnrm.cloud.google.com gke-two foo=bar
```

Update the RootSyncDeployment spec to point to the next revision of the Package:
```
spec:
  packageRevision:
    name: blueprint-93dabbfa9de63b507d13d366185a70ee2ed087f7
```

Apply it again:
```
kubectl apply -f simple.yaml
```

Watch the progress:
```
k get rootsyncdeployments.config.porch.kpt.dev simple -oyaml -w
```
