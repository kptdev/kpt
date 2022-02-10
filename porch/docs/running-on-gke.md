Create a GKE cluster with

# We need the release-channel=rapid as we depend on k8s 1.22 (because of priority and fairness APIs moving to beta2)
```
gcloud container clusters create-auto --region us-central1 --release-channel=rapid porch-dev
```


Ensure you are targeting the GKE cluster:
```
gcloud container clusters get-credentials --region us-central1 porch-dev
```


Push the image:

```
hack/build-image.sh  --project $(gcloud config get-value project) --push

make -C controllers/ push-image
```

Deploy porch apiserver:

```
kubectl apply -f controllers/pkg/apis/porch/v1alpha1/
	
GCP_PROJECT_ID=$(gcloud config get-value project)
kpt fn source ./config/deploy/ -o unwrap | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -


kubectl annotate serviceaccount apiserver \
    --namespace porch-system \
    iam.gke.io/gcp-service-account=porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com
```

Set up workload identity (todo: replace with KCC)

```
gcloud iam service-accounts create porch-server
# We want to create and delete images
gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
    --member "serviceAccount:porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
    --role "roles/artifactregistry.repoAdmin"
gcloud iam service-accounts add-iam-policy-binding porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/apiserver]"
```

```
gcloud iam service-accounts create porch-sync
gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
    --member "serviceAccount:porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
    --role "roles/artifactregistry.reader"
gcloud iam service-accounts add-iam-policy-binding porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/porch-controllers]"
```

Deploy porch controllers:

```
make -C controllers/ push-image

kubectl apply -f controllers/remoterootsync/config/crd/bases/

GCP_PROJECT_ID=$(gcloud config get-value project)
kpt fn source ./controllers/config/deploy/ -o unwrap | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -

kubectl annotate serviceaccount porch-controllers \
    --namespace porch-system \
    iam.gke.io/gcp-service-account=porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com

# Bounce the pods (because we're running :latest currently)
kubectl delete pod -n porch-system -l k8s-app=porch-controllers
```

Create some example repositories / packages:

```
make apply-dev-config
./config/samples/create-deployment-package.sh
```

To test out remoterootsync self-applying:

```
# Grant more RBAC permissions than are normally needed (equivalent to admin permissions)
kubectl apply -f controllers/remoterootsync/config/samples/hack-self-apply-rbac.yaml

# Apply the RemoteRootSyncSet
cat controllers/remoterootsync/config/samples/hack-self-apply.yaml | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -
```
