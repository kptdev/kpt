## RemoteRootSync controller

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
