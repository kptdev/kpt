# Simple example

This example creates an IAMPolicyMember so that the reference Kubernetes Service Account can access GCP services using the referenced GCP Service Account.
It also annotates the KSA with the `iam.gke.io/gcp-service-account` annotation.

## Setup
Create a Kubernetes Service Account (KSA) in the default namespace:

```
kubectl -n default create sa my-example-ksa
```

Use Config Connector to create a GCP Service Account (GSA):

```
cat <<EOF | kubectl apply -f -
apiVersion: iam.cnrm.cloud.google.com/v1beta1
kind: IAMServiceAccount
metadata:
  name: my-example-gsa
  namespace: config-control
spec:
  displayName: Example Service Account
EOF
```

Apply the `simple.yaml` manifest:
```
kubectl apply -f simple.yaml
```

We can verify that the iam policy member has been created with
gcloud:
```
gcloud iam service-accounts get-iam-policy my-example-gsa@<PROJECT>.iam.gserviceaccount.com
```

And see that the KSA have been annotated:
```
kubectl get sa my-example-ksa -oyaml
```