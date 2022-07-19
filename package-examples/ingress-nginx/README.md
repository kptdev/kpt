# ingress-nginx

## Description

Nginx Ingress Controller package

## Usage

### Fetch the package

```sh
kpt pkg get git@github.com:googlecontainertools/kpt.git/package-examples/ingress-nginx ingress-nginx
```

### View package content

```sh

kpt pkg tree ingress-nginx

```

### Apply the package

```sh

kpt live init ingress-nginx
kpt live apply ingress-nginx --reconcile-timeout=2m --output=table

```

### How was this package created

```sh

# download the static manifests from the github releases
wget -O ingress-nginx.yaml https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.3.0/deploy/static/provider/cloud/deploy.yaml

```

Starlark function to add `app.kubernetes.io/component` label if it doesn't exists.

```yaml

## default-label.yaml

apiVersion: fn.kpt.dev/v1alpha1
kind: StarlarkRun
metadata:
  name: set-cluster-label
  annotations:
source: |
  # set the component label to cluster if not specified
  def setlabel(resources):
    for resource in resources:
      curr_labels = resource.get("metadata").get("labels")
      if "app.kubernetes.io/component" not in curr_labels:
        resource["metadata"]["labels"]["app.kubernetes.io/component"] = "controller"
  setlabel(ctx.resource_list["items"])
```

Create the package

```sh

mkdir ingress-nginx
kpt pkg init ingress-nginx

cat ingress-nginx.yaml |kpt fn eval - -o unwrap -i starlark:v0.4.3 --fn-config default-label.yaml| kubectl-slice  --template '{{ index "app.kubernetes.io/component" .metadata.labels }}/{{.kind | lower}}-{{.metadata.name|dottodash}}.yaml' -o ingress-nginx --dry-run

```