```
kind create cluster
kind export kubeconfig
```


```
IMAGE_TAG=$(date +%Y%m%dT%H%M%S)
IMAGE_REPO=localkind IMAGE_TAG=${IMAGE_TAG} make build-images

# TODO: Do we need to load on Linux?
kind load docker-image localkind/porch-function-runner:${IMAGE_TAG}
kind load docker-image localkind/porch-controllers:${IMAGE_TAG}
kind load docker-image localkind/porch-wrapper-server:${IMAGE_TAG}
kind load docker-image localkind/porch-server:${IMAGE_TAG}
kind load docker-image localkind/test-git-server:${IMAGE_TAG}

IMAGE_REPO=localkind IMAGE_TAG=${IMAGE_TAG} make deploy-no-sa
```

```
cat ../e2e/testdata/porch/git-server.yaml | \
  sed -e s/test-git-namespace/git-system/g |
  sed -e s~GIT_SERVER_IMAGE~localkind/test-git-server:${IMAGE_TAG}~g |
  kubectl apply -f -
```



```

kubectl apply -f - <<EOF
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: blueprints
  namespace: default
spec:
  description: Blueprints Git Repository
  content: Package
  type: git
  git:
    repo: https://github.com/justinsb/kpt-samples
    branch: packages
    directory: ""
EOF

```


```

kubectl apply -f - <<EOF
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: deployment
  namespace: default
spec:
  content: Package
  deployment: true
  description: 'Deployment Repository'
  type: git
  git:
    branch: main-branch
    createBranch: true
    #directory: /
    repo: http://git-server.git-system.svc.cluster.local:8080/deployment
EOF

```

```
kubectl config set-context $(kubectl config current-context) --namespace default
kubectl get packagerevision
kubectl get packagerevision --field-selector spec.packageName=echo
kubectl get packagerevision --field-selector spec.packageName=echo -oyaml

kubectl get packagerevisionresources
kubectl get packagerevisionresources --field-selector spec.packageName=echo
kubectl get packagerevisionresources --field-selector spec.packageName=echo -oyaml

```

```
kubectl apply -f - <<EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  namespace: default
  name: "deployment:external-secrets:v1"
spec:
  packageName: external-secrets
  revision: v1
  repository: deployment
  tasks:
  - type: clone
    clone:
      upstreamRef:
        type: git
        git:
          repo: https://github.com/justinsb/kpt-samples
          ref: packages
          directory: external-secrets
EOF

```

```

kubectl get packagerevision -n default  --field-selector spec.packageName=external-secrets
kubectl get packagerevision -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment 
kubectl get packagerevision -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment  -oyaml

kubectl get packagerevisionresources -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment
kubectl get packagerevisionresources -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment -oyaml | less


```

kubectl apply -f - <<EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  namespace: default
  name: "deployment:external-secrets:v1"
spec:
  packageName: external-secrets
  revision: v1
  repository: deployment
  tasks:
  - type: clone
    clone:
      upstreamRef:
        type: git
        git:
          repo: https://github.com/justinsb/kpt-samples
          ref: packages
          directory: external-secrets
  - type: eval
    eval:
      image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        first-label: label-value
        another-label: another-label-value
EOF

```

```
kubectl get packagerevision -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment 
kubectl get packagerevision -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment  -oyaml

kubectl get packagerevisionresources -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment
kubectl get packagerevisionresources -n default  --field-selector spec.packageName=external-secrets --field-selector spec.repository=deployment -oyaml | less
```

# Reset procedure

```

k delete repository --all
k delete pod -n porch-system --all
k delete pod -n git-system --all

```
