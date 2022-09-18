# Simple example

This example adds a RootSync with name `simple` to two GKE clusters
created with Config Connector.

## Setup
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

Apply the `simple.yaml` manifest:
```
k apply -f simple.yaml
```