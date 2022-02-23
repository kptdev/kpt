# Running ok GKE

Create a GKE cluster:

**Note**: We need the release-channel=rapid as we depend on k8s 1.22 (because of priority and fairness APIs moving to beta2)

```
gcloud container clusters create-auto --region us-central1 --release-channel=rapid porch-dev
```

Ensure you are targeting the GKE cluster:
```
gcloud container clusters get-credentials --region us-central1 porch-dev
```

Create service accounts and assign roles:
```
GCP_PROJECT_ID=$(gcloud config get-value project)
gcloud iam service-accounts create porch-server
gcloud iam service-accounts create porch-sync

# We want to create and delete images
gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
    --member "serviceAccount:porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
    --role "roles/artifactregistry.repoAdmin"
gcloud iam service-accounts add-iam-policy-binding porch-server@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/apiserver]"

gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \
    --member "serviceAccount:porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
    --role "roles/artifactregistry.reader"
gcloud iam service-accounts add-iam-policy-binding porch-sync@${GCP_PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${GCP_PROJECT_ID}.svc.id.goog[porch-system/porch-controllers]"
```

Build Porch, push images, and deploy porch server and controllers:

```
IMAGE_TAG=$(git rev-parse --short HEAD) make push-and-deploy
```

Create some example repositories / packages:

```
# Create artifact-registry repos etc
make apply-dev-config
# Push a sample hello-world app
make -C config/samples/apps/hello-server push-image
# Create a package for the sample hello-world app
./config/samples/create-deployment-package.sh
```

To test out remoterootsync self-applying:

```
# Grant more RBAC permissions than are normally needed (equivalent to admin permissions)
kubectl apply -f controllers/remoterootsync/config/samples/hack-self-apply-rbac.yaml

# Apply the RemoteRootSyncSet
cat controllers/remoterootsync/config/samples/hack-self-apply.yaml | sed -e s/example-google-project-id/${GCP_PROJECT_ID}/g | kubectl apply -f -
```
