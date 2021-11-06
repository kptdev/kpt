# Title

* Author(s): Yuwen Ma <yuwenma@google.com>
* Contributor(s):
    - Sean Sullivan <seans@google.com>
    - Sunil Arora <sunilarora@google.com>
    - Mortent Torkildsen <mortent@google.com>
    - Mengqi Yu <mengqiy@google.com>
    - Natasha Sarkar <natashasarkar@google.com>

* Approver: Sunil Arora <sunilarora@google.com>

## Why

This design is aimed at making `kpt live` be hydration-tool-neutral so that `kpt live`
applicable use cases can be expanded to `kustomize build`, `helm templates` and raw
kubernetes manifests.

### Non-goal
- This doc is not aimed at designing a unified hydration between kpt and kustomize
- This doc is not aimed at changing the kpt in-place hydration models.

## Design

We propose to make `kpt live` support reading configurations from standard input,
detach the Inventory from Kptfile and simplify the Inventory-ResourceGroup mechanism
to use  ResourceGroup only.

### Read standard input resources

We will add a reserved keyword “-” as a special kpt directory. This "-" accepts resources
from the standard input. For example,  `kustomize build | kpt live apply -`

### Initialize a `ResourceGroup` object

`kpt live init` will create a ResourceGroup CR in resourcegroup.yaml. Only three fields
should be given to the CR.

- metadata.name: A client-provided valid RFC 1123 DNS subdomain name. Default value has prefix “inventory-” with
  8 digit numbers. E.g. “inventory-02014045”
- metadata.namespace: a valid namespace. default to value “default”
- metadata.labels.cli-utils.sigs.k8s.io/inventory-id: A client-provided valid label.
  Default value is a UUID e.g. 7c5af957-a3e2-4d68-8c0f-6c1864a66050

### Convert Inventory to ResourceGroup

The client-side Inventory is considered as the identifier of the cluster-side
ResourceGroup CR. Right now, the Inventory is stored in the Kptfile to guarantee
the ResourceGroup is unique. This requires Kptfile to exist to use `kpt live apply`.

To split the Inventory from Kptfile to resourcegroup.yaml and convert the Inventory to
ResourceGroup, `kpt live migrate` should be extended to map the inventory.name,
inventory.namespace, inventory.inventoryID to  ResourceGroup CR metadata.name,
metadata.namespace, metadata.labels.cli-utils.sigs.k8s.io/inventory-id correspondingly.

Inventory in Kptfile
```yaml
apiVersion: kpt.dev/v1betaX
kind: Kptfile
inventory:
  name: <INVENTORY_NAME>
  namespace: <INVENTORY_NAMESPACE>
  inventoryID: <INVENTORY_ID>
```
resoucegroup.yaml
```yaml
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata  
  name: <INVENTORY_NAME>
  namespace: <INVENTORY_NAMESPACE>
  labels:
    cli-utils.sigs.k8s.io/inventory-id: <INVENTORY_ID>
```

### ResourceGroup as a singleton object

`kpt live apply [--resourcegroup-path] -` from STDIN accepts and only accepts a single
ResourceGroup, including the ResourceGroup provided by the flag. It detects
1. If more than one ResourceGroup is found, raise errors and display all the ResourceGroup objects.
2. If no ResourceGroup is found in STDIN and and Kptfile inventory does not exists, raise errors and suggest users to
   run kpt live init
3. If no ResourceGroup is found in STDIN and Kptfile inventory exists, raise errors and
   suggest users to run kpt live migrate

## User Guide

### To hydrate via kustomize and deploy via kpt

#### Day 1

<b>For new users to start from scratch (no Kptfile)</b>
User can run `kpt live init [--resourcegroup-file=CUSTOM_RG.yaml]` to create a
ResourceGroup object and store it in a resourcegroup.yaml file.
Users can customize the file path with the flag “--resourcegroup-file”.

<b>For existing kpt users to migrate from Kptfile</b>
Users run `kpt live migrate [--resourcegroup-file=CUSTOM_RG.yaml]` to convert the
Inventory object from Kptfile to a standalone resourcegroup.yaml file.
Users can customize the file path with the flag “--resourcegroup-file”.

<b>[optional]: Add shareable ResourceGroup to kustomize resources</b>
If the ResourceGroup is expected to be shared in the Gitops workflow, users can add
the resourcegroup.yaml  file path to the .resources field in kustomization.yaml.
This simplifies the Day N deployment by omitting the “–resourcegroup-file“ flag.

#### Day N

- Users can configure the hydration rules in kustomization.yaml
- Users can run `kustomize build <DIR> | kpt live apply -` to hydrate and deploy
  the kustomize-managed configurations. If resourcegroup.yaml is not added to
  kustomize .resources field, users should provide the “–resourcegroup-file“ flag.  
  `kustomize build <DIR> | kpt live apply –resourcegroup-file <CUSTOM_RG.yaml> -`

### To hydrate via helm and deploy via kpt

#### Day 1

<b>For new users to start from scratch (no Kptfile)</b>
User can run `kpt live init --resourcegroup-file=<DIR>/resourcegroup.yaml` to create
a ResourceGroup object and store it in the helm template <DIR>.

<b>For existing kpt users to migrate from Kptfile</b>
Users run `kpt live migrate --resourcegroup-file=<DIR>/resourcegroup.yaml` to convert
the Inventory object from Kptfile to  a standalone resourcegroup.yaml file.

#### Day N

Users can run `helm templates <DIR> | kpt live apply -` to hydrate and deploy the
helm-managed resources.

See kpt issue #2399 for expected Inventory usage in package publisher/consumer.

## FAQ

### Will Inventory be deprecated from Kptfile?

Kpt still supports inventory in Kptfile and it is not required to migrate to the
standalone resourcegroup-path. In fact, users who do not use STDIN in `kpt live apply`
will still have inventory read from Kptfile by default unless the –inventory-path flag
is given.

### Why not kpt live apply -k?

We originally propose to use a single command kpt live apply -k (similar idea as
kubectl apply -k ) to kustomize-hydrate and kpt-deploy the resources, storing the
inventory in the annotation fields of the kustomization.yaml (See below).
The inventory can be auto added to kustomization.yaml by kpt live init
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  annotations:
     kpt-inventory-id:
     kpt-inventory-name:
     kpt-inventory-namespace: 
```
#### Pros

- Extreme simple user steps. Kustomize users only need to install kpt and run kpt live
  init. Then they can start using kpt live apply to deploy their kustomize resources.
  No manual changes to any configurations.
- Easy to understand. The command is similar to  kubectl apply -k which makes it easier
  to be accepted and remembered.

There are two main reasons to abandon this approach:
- Making Kpt a hydration-neutral deployment tool helps it better hook up with different
  hydration tools.
- kpt is not planning to treat kustomize differently and thus it does not make sense to
  provide kustomize a shortcut.

## Alternatives Considered

### Use Inventory metadata to store ResourceGroup info

Store the Kptfile inventory to a standalone Inventory object. Since the inventory is
the metadata of the cluster side ResourceGroup,  hiding the ResourceGroup behind
inventory provides users simpler syntax and mitigates the probability of overriding
the ResourceGroup unintentionally (e.g. via kubectl apply).

inventory.yaml
```yaml
apiVersion: kpt.dev/v1betaX
kind: Inventory
metadata:
  name: <INVENTORY_NAME>
  namespace: <INVENTORY_NAMESPACE>
inventoryID: <INVENTORY_ID>
```

#### Pros

- Simpler resource syntax
- Mitigate unintentional resourceGroup override.

#### Cons
- The client-side inventory requires users to manage.
  This new resource increases the overall complexity of the kpt actuation config.
  We frequently receive user confusions about the difference between ResourceGroup
  and Inventory and why they cannot configure inventory like other kubernetes resource
  (adoption mismatch errors).

### Use Inventory spec to store ResourceGroup info


inventory.yaml
```yaml
apiVersion: kpt.dev/v1betaX
kind: Inventory
spec:
  resourceGroup:
    name: <INVENTORY_NAME>
    Namespace: <INVENTORY_NAMESPACE>
    labels:
      cli-utils.sigs.k8s.io/inventory-id: <INVENTORY_ID>
```
#### Pros

- Obey the kubernetes convention and make it clearer that users are expected to provide the resourceGroup kind with name, namespace and labels.
- Mitigate unintentional resourceGroup override.

#### Cons
More complex syntax than ResourceGroup or Inventory metadata solution

### Add a controller for ResourceGroup pruning; No client-side Inventory

Inventory is a [cli-utils object](https://github.com/kubernetes-sigs/cli-utils/blob/52b000849deb2b233f0a58e9be16ca2725a4c9cf/pkg/inventory/inventory.go#L30) used to store the [ResourceGroup metadata](https://docs.google.com/document/d/1x_02xGKZH6YGuc3K-KNeY4KmADPAwzrlQKkvTNXYekc/edit#heading=h.265kfx5ku27).
It is stored in Kptfile .Inventory and required by `kpt live apply`. In lieu of a user configuration, it is more of a data for kpt to identify the corresponding ResourceGroup for pruning. This adds non-trivial overheads for users to understand, maintain and retrieve the inventory.

The alternative proposal is to remove the inventory but use a new controller to prune the resource.
The controller will be installed together with ResourceGroup CRD (in each `kpt live apply`).
It will find the previous ResourceGroup CR from current resources, three-way merging and deploying
the new resources and update the ResourceGroup.

#### Pros
- Improve the user experience on using kpt live and minimize the confusions around inventory and ResourceGroup
- Eliminate the pain points on retrieving the missing Inventory (list all ResourceGroup, check the .spec.resources and try one by one until not encounter the “missing adoption” error).

#### Cons
The controller needs multiple requests to find out the previous ResourceGroup CR. This causes heavy traffic in a production cluster.  
The controller cannot 100% guarantee the resourceGroup reverse lookup are accurate.
Since the pruning happens on the cluster side, kpt can no longer provide real-time status updates.
