# Accessing the Configuration as Data UI

The easiest way to access the Configuration as Data UI is running by a docker
container on your local machine where you'll be able to access the UI with your
browser. Running the container locally simplifies the overall setup by allowing
the UI to use your local kubeconfig and Google credentials to access the kubernetes
cluster with Porch installed. This guide will show you how to do this.

## Prerequisites

To access the Configuration as Data UI with a docker container, you will need:

*   [Porch](guides/porch-installation.md) installed on a kubernetes cluster
*   [kubectl](https://kubernetes.io/docs/tasks/tools/) targeting the kubernetes cluster
    with Porch installed
*   [git](https://git-scm.com/)
*   [docker](https://docs.docker.com/get-docker/)

## Running on a GKE cluster

This setup assumes that you have a GKE cluster up and running with porch installed, and that
your current kube context is set to that GKE cluster. We would welcome contributions or feedback
from people that have set this up in other clouds outside of GKE.

First, find a published image in the [kpt-dev/kpt-backstage-plugins container registry](https://console.cloud.google.com/gcr/images/kpt-dev/global/kpt-backstage-plugins/backstage-plugin-cad?project=kpt-dev).
For this example, we will use `gcr.io/kpt-dev/kpt-backstage-plugins/backstage-plugin-cad:v0.1.0`.

Next, create a namespace called `backstage`:

```sh
kubectl create namespace backstage
```

Then, run the following command to set up the backstage deployment and service account.
Change the image name and tag in the `newName` and `newTag` flags in the below `kpt fn eval` command to
the ones you would like to use:

```sh
echo "
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backstage
  namespace: backstage
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backstage
  template:
    metadata:
      labels:
        app: backstage
    spec:
      containers:
        - name: backstage
          image: backstage
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 7007
          env:
            - name: AUTH_GOOGLE_CLIENT_ID
              value: 147984173186-oht42q0t5offa8c7sgd6quu65plvuc86.apps.googleusercontent.com
            - name: AUTH_GOOGLE_CLIENT_SECRET
              value: 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
            - name: NODE_ENV
              value: development
            - name: USE_IN_CLUSTER_CREDENTIALS
              value: "make-it-so"
---
apiVersion: v1
kind: Service
metadata:
  name: backstage
  namespace: backstage
spec:
  selector:
    app: backstage
  ports:
    - name: http
      port: 7007
      targetPort: http
" | kpt fn eval "" -o unwrap --image set-image:v0.1.0 -- \
name=backstage newName=gcr.io/kpt-dev/kpt-backstage-plugins/backstage-plugin-cad newTag=v0.1.0 | \
kubectl apply -f -
```

In your cluster, confirm the backstage deployment is ready and available:

```sh
kubectl get deployment -n backstage
```

Forward port 7007 locally to access backstage. It is important that you do not map this port to another port
as this will cause Backstage and the OAuth client to fail.

```sh
kubectl port-forward --namespace=backstage svc/backstage 7007
```

Open the plugin by browsing to `localhost:7007/config-as-data`. On the plugin, you will need to sign in to your
Google account so that the plugin can access your GKE cluster.

## Running locally in a container

This setup is intended for those developing the plugin. These instructions assume GKE and workload identity,
to simplify authentication configuration, but we would welcome contributions or feedback from people that have set
this up in other clouds.

First, clone the
[kpt-backstage-plugins](https://github.com/GoogleContainerTools/kpt-backstage-plugins)
repository.

```sh
git clone https://github.com/GoogleContainerTools/kpt-backstage-plugins.git
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

This setup is intended for those installing the plugin into existing backstage deployments.

The Configuration as Data UI can be added to an existing
[Backstage](https://backstage.io) application by following the instructions on
the
[Configuration as Data Plugin README](https://github.com/GoogleContainerTools/kpt-backstage-plugins/tree/main/plugins/cad/README.md).
