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

We will change the existing `kpt live apply` from STDIN to not require the existence of Kptfile (
to be more specific, the `inventory` file from Kptfile). As long as the STDIN contains
one and only one valid ResourceGroup, `kpt live apply` should be able to create/match the
ResourceGroup in cluster.

### Initialize a `ResourceGroup` object

`kpt live init` will create a ResourceGroup CR in resourcegroup.yaml. 

By default, the ResoureGroup will have name and namespace assigned as below. And users can
override via existing flags "--name" and "--namespace".

- metadata.name: A client-provided valid RFC 1123 DNS subdomain name. Default value has prefix “inventory-” with
  8 digit numbers. E.g. "inventory-02014045"
- metadata.namespace: a valid namespace. default to value "default"

If users want to reuse an existing inventory (from Kptfile) or ResourceGroup (which has been deployed to the cluster),
they shall provide the value of the inventory's inventory-id or the ResourceGroup's "metadata.labels[0].cli-utils.sigs.k8s.io/inventory-id"
via "--inventory-id" flag.

### Convert Inventory to ResourceGroup

The client-side Inventory is considered as the identifier of the cluster-side
ResourceGroup CR. Right now, the Inventory is stored in the Kptfile to guarantee
the ResourceGroup is unique. This requires Kptfile to exist to use `kpt live apply`.

To split the Inventory from Kptfile to resourcegroup.yaml and convert the Inventory to
ResourceGroup, `kpt live migrate` should be extended to map the inventory.name,
inventory.namespace, inventory.inventoryID to ResourceGroup CR metadata.name,
metadata.namespace, metadata.labels.cli-utils.sigs.k8s.io/inventory-id correspondingly.

Inventory in Kptfile
```yaml
apiVersion: kpt.dev/v1
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

### Simplify the Inventory

Current inventory contains inventory-id which is required to match the label `cli-utils.sigs.k8s.io/inventory-id`.

For new users, they should no longer need to be exposed to the inventory-id, but kpt will
build one composed by `name-namespace` on the fly. 

For existing users, the inventory-id is still
required in the standalone ResourceGroup file to guarantee the adoption matches, unless they use "--inventory-policy=adopt"
to override the label. This flag is only required as a one-off via `kpt live apply -` to override the label to `name-namespace`.

### ResourceGroup as a singleton object

`kpt live apply [--rg-file] -` from STDIN accepts and only accepts a single
ResourceGroup, including the ResourceGroup provided by the flag. It detects
1. If more than one ResourceGroup is found, raise errors and display all the ResourceGroup objects.
2. If no ResourceGroup is found in STDIN and and Kptfile inventory does not exists, raise errors and suggest users to
   run kpt live init
3. If no ResourceGroup is found in STDIN and Kptfile inventory exists, raise errors and
   suggest users to run kpt live migrate

### New flags

#### `--rg-file` 

- description: The file path to the ResourceGroup CR, default to `resourcegroup.yaml`
- short form `--rg`
- This flag will be added to `kpt live init`, `kpt live migrate` and `kpt live apply`

#### `--name` for inventory

- description: The name for the ResourceGroup
- This flag will continue to be used by `kpt live init`. Rather than overriding the
  inventory.name in Kptfile, it will override the default metadata.name in the standalone ResourceGroup file.

#### `--namespace` for inventory

- description: The namespace for the ResourceGroup
- This flag will continue to be used by `kpt live init`, Rather than overriding the
  inventory.namespace in Kptfile, it will override the default metadata.namespace in the standalone ResourceGroup file.

#### `--inventory-id` for inventory

- description: Inventory identifier. This is used to detect overlap between
  two sets of ResourceGroup managed resources that might use the same name and namespace.
- This flag will continue to be accepted by `kpt live init` for backward compatibility reasons. 
  If given, ResourceGroup will store the inventory-id value in "metadata.labels[0].cli-utils.sigs.k8s.io/inventory-id"
  of the ResourceGroup. 
  If not given, the ResourceGroup labels will be empty and the value of "<name>-<namespace>" will be
  used as the "cli-utils.sigs.k8s.io/inventory-id" label in `kpt live apply` from STD.

### `resoucegroup.yaml` interaction with `kpt fn` commands

All `kpt fn` commands should treat `resoucegroup.yaml` as a special resource 
and not modify it in any scenario. It should not be considered as a meta resource and 
`include-meta-resources` flag should not include the `ResourceGroup` resource in 
the `ResourceList`. In the future, we can add a new flag `--include-rg` if there are
valid use-cases to modify or include `ResourceGroup` resource in `ResourceList`.

## User Guide

### To hydrate via kustomize and deploy via kpt

#### Day 1

##### For new users to start from scratch (no Kptfile)
User can run `kpt live init [--rg-file=resourcegroup.yaml]` to create a
ResourceGroup object and store it in a resourcegroup.yaml file.
Users can customize the file path with the flag “--rg-file”.

##### For existing kpt users to migrate from Kptfile
Users run `kpt live migrate [--rg-file=resourcegroup.yaml]` to convert the
Inventory object from Kptfile to a standalone resourcegroup.yaml file.
Users can customize the file path with the flag “--rg-file”.

##### [optional]: Add shareable ResourceGroup to kustomize resources
If the ResourceGroup is expected to be shared in the Gitops workflow, users can add
the resourcegroup.yaml  file path to the .resources field in kustomization.yaml.
This simplifies the Day N deployment by omitting the “–rg-file“ flag.

#### Day N

- Users can configure the hydration rules in kustomization.yaml
- Users can run `kustomize build <DIR> | kpt live apply -` to hydrate and deploy
  the kustomize-managed configurations. If resourcegroup.yaml is not added to
  kustomize .resources field, users should provide the “–rg-file“ flag.  
  `kustomize build <DIR> | kpt live apply –rg-file <resourcegroup.yaml> -`

### To hydrate via helm and deploy via kpt

#### Day 1

##### For new users to start from scratch (no Kptfile)
User can run `kpt live init --rg-file=<DIR>/resourcegroup.yaml` to create
a ResourceGroup object and store it in the helm template <DIR>.

##### For existing kpt users to migrate from Kptfile
Users run `kpt live migrate --rg-file=<DIR>/resourcegroup.yaml` to convert
the Inventory object from Kptfile to  a standalone resourcegroup.yaml file.

#### Day N

Users can run `helm templates <DIR> | kpt live apply -` to hydrate and deploy the
helm-managed resources.

See kpt issue #2399 for expected Inventory usage in package publisher/consumer.

## FAQ

### Will Inventory be deprecated from Kptfile?

Kpt still supports inventory in Kptfile and it is not required to migrate to the
standalone rg-path. In fact, users who do not use STDIN in `kpt live apply`
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
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
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
