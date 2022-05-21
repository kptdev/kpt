Before you can apply the package to the cluster, it needs to be initialized
using `live init`. This is a one-time client-side operation that adds metadata
to the `ResourceGroup` CR (by default located in `resourcegroup.yaml` file)
specifying the name, namespace and inventoryID of the `ResourceGroup` resource
`live apply` command will use to store the inventory (list of the resources applied).

Let's initialize the `wordpress` package:

```shell
$ kpt live init wordpress
initializing "resourcegroup.yaml" inventory info (namespace: default)...success
```

This creates the `ResourceGroup` CR in the `resourcegroup.yaml` file:

```yaml
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: inventory-74096247
  namespace: default
  labels:
    cli-utils.sigs.k8s.io/inventory-id: 0a32e2c0200b4bd4c19cd3e097086b4648b8902d-1653113657067255815
```

`ResourceGroup` is a namespace-scoped resource. By default, `live init` command
uses heuristics to automatically choose the namespace. In this example, all the
resources in `wordpress` package were in the `default` namespace, so it chose
`default` for the namespace. Alternatively, you can manually configure the name
and namespace of the `ResourceGroup` resource.

?> Refer to the [init command reference][init-doc] for usage.

!> Once a package is applied to the cluster, you do not want to change the
`ResourceGroup` CR; doing so severs the association between the
package and the inventory in the cluster, leading to destructive operations.

[init-doc]: /reference/cli/live/init/
