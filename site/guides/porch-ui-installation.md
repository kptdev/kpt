# Accessing the Configuration as Data UI

You can access the Configuration as Data UI UI either by running the UI on a
cluster or integrating the UI into an existing Backstage installation.

## Prerequisites

To access the Configuration as Data UI, you will need:

- [Porch](guides/porch-installation.md) installed on a Kubernetes cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/) targeting the Kubernetes
  cluster with Porch installed
- [kpt CLI](https://kpt.dev/installation/kpt-cli) installed

## Running on a GKE cluster

This setup assumes that you have a GKE cluster up and running with Porch installed, and that
your current kube context is set to that GKE cluster. We would welcome contributions or feedback
from people that have set this up in other clouds outside of GKE.

First, create a namespace called `backstage`:

```sh
kubectl create namespace backstage
```

Second, Google OAuth Credentials will need to be created to allow the UI to use
Google authentication to authenticate users with the UI and to the Kubernetes
API Server.

To create the Google OAuth Credentials:

1. Sign in to the [Google Console](https://console.cloud.google.com)
2. Select or create a new project from the dropdown menu on the top bar
3. Navigate to
   [APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials)
4. Click **Create Credentials** and choose `OAuth client ID`
5. Configure an OAuth consent screen, if required
   - For scopes, select `openid`, `auth/userinfo.email`,
     `auth/userinfo.profile`, and `auth/cloud-platform`.
   - Add any users that will want access to the UI if using External user type
6. Set **Application Type** to `Web Application` with these settings:
   - `Name`: Config as Data UI (or any name you prefer)
   - `Authorized JavaScript origins`: http://localhost:7007
   - `Authorized redirect URIs`:
     http://localhost:7007/api/auth/google/handler/frame
7. Click Create
8. Copy the Client ID and Client secret displayed

We will now need to add the credentials as a secret to the cluster. Be sure to
replace the PLACEHOLDER values prior to executing:

```sh
kubectl create secret generic -n backstage cad-google-oauth-client --from-literal=client-id=CLIENT_ID_PLACEHOLDER --from-literal=client-secret=CLIENT_SECRET_PLACEHOLDER
```

Next, find a published image in the
[kpt-dev/kpt-backstage-plugins container registry](https://console.cloud.google.com/gcr/images/kpt-dev/global/kpt-backstage-plugins/backstage-plugin-cad?project=kpt-dev).
For this example, we will use
`gcr.io/kpt-dev/kpt-backstage-plugins/backstage-plugin-cad:v0.1.3`.

Now, run the following command to set up the backstage deployment and service.
Change the image name and tag in the `newName` and `newTag` flags in the below
`kpt fn eval` command to the ones you would like to use:

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
          ports:
            - name: http
              containerPort: 7007
          env:
            - name: AUTH_GOOGLE_CLIENT_ID
              valueFrom:
                secretKeyRef:
                  name: cad-google-oauth-client
                  key: client-id
                  optional: false
            - name: AUTH_GOOGLE_CLIENT_SECRET
              valueFrom:
                secretKeyRef:
                  name: cad-google-oauth-client
                  key: client-secret
                  optional: false
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
name=backstage newName=gcr.io/kpt-dev/kpt-backstage-plugins/backstage-plugin-cad newTag=v0.1.3 | \
kubectl apply -f -
```

In your cluster, confirm the backstage deployment is ready and available:

```sh
kubectl get deployment -n backstage backstage
```

Forward port 7007 locally to access backstage. It is important that you do not map this port to another port
as this will cause Backstage and the OAuth client to fail.

```sh
kubectl port-forward --namespace=backstage svc/backstage 7007
```

Open the plugin by browsing to `localhost:7007/config-as-data`. On the plugin, you will need to sign in to your
Google account so that the plugin can access your GKE cluster.

## Running in Backstage

This setup is intended for those installing the plugin into existing backstage deployments.

The Configuration as Data UI can be added to an existing
[Backstage](https://backstage.io) application by following the instructions on
the
[Configuration as Data Plugin README](https://github.com/GoogleContainerTools/kpt-backstage-plugins/tree/main/plugins/cad/README.md).
