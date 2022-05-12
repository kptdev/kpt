# Accessing the Configuration as Data UI

The easiest way to access the Configuration as Data UI is running by a docker
container on your local machine where you'll be able to access the UI with your
browser. Running the container locally simplifies the overall setup by allowing
the UI to use your local kubeconfig and Google credentials to access the GKE
cluster with Porch installed. This guide will show you how to do this.

## Prerequisites

To access the Configuration as Data UI with a docker container, you will need:

*   [Porch](guides/porch-installation.md) installed on a GKE cluster
*   [kubectl](https://kubernetes.io/docs/tasks/tools/) targeting the GKE cluster
    with Porch installed
*   [git](https://git-scm.com/)
*   [docker](https://docs.docker.com/get-docker/)

## Running locally in a container

First, clone the
[kpt-backstage-plugins](https://github.com/GoogleContainerTools/kpt-backstage-plugins)
repository.

```sh
git clone https://github.com/GoogleContainerTools/kpt-backstage-plugins.git kpt-backstage-plugins
cd kpt-backstage-plugins
```

Next, build the kpt-backstage-plugins image.

```sh
docker build --target backstage-app-local --tag kpt-backstage-plugins .
```

And create a new container using the kpt-backstage-plugins image. The two
attached volumnes allows the UI to connect to your GKE using your local Google
credentials, and the UI will be exposed over port 7007.

```sh
docker run -v ~/.kube/config:/root/.kube/config -v ~/.config/gcloud:/root/.config/gcloud -p 7007:7007 kpt-backstage-plugins
```

And now access the Configuration as Data UI by opening your browser to
http://localhost:7007/config-as-data.

## Running in Backstage

The Configuration as Data UI can be added to an existing
[Backstage](https://backstage.io) application by following the instructions on
the
[Configuration as Data Plugin README](https://github.com/GoogleContainerTools/kpt-backstage-plugins/tree/main/plugins/cad/README.md).
