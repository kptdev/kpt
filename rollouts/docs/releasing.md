# Releasing Rollouts

The current release process for Rollouts is manual and subject to change. Here are the steps.

## Build the controller images

First, go to the [kpt-dev](https://console.cloud.google.com/gcr/images/kpt-dev/global/rollouts-controller?project=kpt-dev) to view existing rollouts controller images. Note the previous released version so that
you can decide what the next release version should be.

For example, if the previous released version is `v0.0.2`, you will most likely want to do `v0.0.3` next.

Set the next release version:
```sh
export VERSION=<version>
```

Build the docker image:
```sh
IMG=gcr.io/kpt-dev/rollouts-controller:$VERSION make docker-build
```

Make sure you are connected to the kpt-dev project, and then push the image:
```sh
gcloud config set project kpt-dev
IMG=gcr.io/kpt-dev/rollouts-controller:$VERSION make docker-push
```

Run the `create-manifests` script to update the manifests:

```sh
./scripts/create-manifests.sh --controller-image gcr.io/kpt-dev/rollouts-controller:$VERSION
```

Create a pull request with the generated changes. Users can now use `kpt pkg get` to pull the
new Rollout manifests with the updated image.
