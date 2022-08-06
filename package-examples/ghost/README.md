### Ghost Application

"Ghost is a powerful app for new-media creators to publish, share, and grow a business around their content. It comes with modern tools to build a website, publish content, send newsletters & offer paid subscriptions to members."
https://ghost.org/

### Quick start

#### Get KPT Pacakge
```bash
export NAMESPACE=<YOUR NAMESPACE>
# make sure the namespace is correct and exists. Otherwise, create the namespace
kubectl create namespace ${NAMESPACE}

# You get this Ghost package by running
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/ghost@main ${NAMESPACE} --for-deployment
```

#### Update the KRM resources to your own data

Updating the KRM resources are easy with variant constructor
```bash
kpt fn render ${NAMESPACE}
```

#### Deploy the KRM resources to your cluster

```bash
# Initialize inventory info. You only need to run this if do not have resourcesgroup.yaml
kpt live init ${NAMESPACE}

kpt live apply ${NAMESPACE}
```

You need to manually update the Ghost Host IP after deployment.
```bash
# Get external IP from Service
kubectl get -n ${NAMESPACE} service/ghost-app -ojsonpath='{.status.loadBalancer.ingress[].ip}'
```
Use this IP to replace the "EXTERNAL_IP_FROM_SERVICE" value in `ghost/deployment-ghost.yaml`

Re-apply the new deployment.
```bash
kpt live apply
``` 
