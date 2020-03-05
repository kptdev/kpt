## live

Reconcile configuration files with the live state

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="coming..." speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    # run the tutorial from the cli
    kpt tutorial live

[tutorial-script]

### Synopsis

Tool to safely apply and delete kubernetes package resources from clusters.

| Command   | Description                                       |
|-----------|---------------------------------------------------|
| [init]    | initialize a package creating a local file        |
| [apply]   | apply a package to the cluster                    |
| [preview] | preview the operations that apply will perform    |
| [destroy] | remove the package from the cluster               |

**Data Flow**: local configuration or stdin -> kpt live -> apiserver (Kubernetes cluster)

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files             | apiserver                |
| apiserver               | stdout                   |

#### Pruning
kpt live apply will automatically delete resources which have been
previously applied, but which are no longer included. This clean-up
functionality is called pruning. For example, consider a package
which has been applied with the following three resources:

```
service-1 (Service)
deployment-1 (Deployment)
config-map-1 (ConfigMap)
```

Then imagine the package is updated to contain the following resources,
including a new ConfigMap named `config-map-2` (Notice that `config-map-1`
is not part of the updated package):

```
service-1 (Service)
deployment-1 (Deployment)
config-map-2 (ConfigMap)
```

When the updated package is applied, `config-map-1` is automatically
deleted (pruned) since it is omitted.


In order to take advantage of this automatic clean-up, a package must contain
a **grouping object template**, which is a ConfigMap with a special label. An example is:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-grouping-object
  labels:
    cli-utils.sigs.k8s.io/inventory-id: test-group
```

And the special label is:

```
cli-utils.sigs.k8s.io/inventory-id: *group-name*
```

`kpt live apply` recognizes this template from the special label, and based
on this kpt will create new grouping object with the metadata of all applied
objects in the ConfigMap's data field. Subsequent `kpt live apply` commands can
then query the grouping object, and calculate the omitted objects, cleaning up
accordingly. When a grouping object is created in the cluster, a hash suffix
is added to the name. Example:

```
test-grouping-object-17b4dba8
```

#### Status
kpt live apply also has support for computing status for resources. This is 
useful during apply for making sure that not only are the set of resources applied
into the cluster, but also that the desired state expressed in the resource are
fully reconciled in the cluster. An example of this could be applying a deployment. Without
looking at the status, the operation would be reported as successful as soon as the
deployment resource has been created in the apiserver. With status, kpt live apply will
wait until the desired number of pods have been created and become available.

Status is computed through a set of rules for specific types, and
functionality for polling a set of resources and computing the aggregate status
for the set. For CRDs, there is a set of recommendations that if followed, will allow
kpt live apply to correctly compute status.

###

[tutorial-script]: ../gifs/live.sh
[init]: init.md
[apply]: apply.md
[preview]: preview.md
[destroy]: destroy.md
