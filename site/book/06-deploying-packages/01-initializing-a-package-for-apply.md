Before you can apply the package to the cluster, it needs to be initialized
using `live init`. This is a one-time client-side operation that adds metadata
to the `Kptfile` specifying a `ResourceGroup` resource on the cluster associated
with this package. The `ResourceGroup`resource will be created by the
`live apply` command.

Let's initialize the `wordpress` package:

```shell
$ kpt live init wordpress
initializing Kptfile inventory info (namespace: default)...success
```

This adds the `inventory` section to the `Kptfile`:

```yaml
# wordpress/Kptfile (Excerpt)
inventory:
  namespace: default
  name: inventory-10285025
  inventoryID: 06ca0268f3ccda82bfd44bec273530a4f72fa9a7-1623278424452000000
```

`ResourceGroup` is a namespace-scoped resource. By default, `live init` command
uses heuristics to automatically choose the namespace. In this example, all the
resources in `wordpress` package were in the `default` namespace, so it chose
`default` for the namespace. Alternatively, you can manually configure the name
and namespace of the `ResourceGroup` resource.

?> Refer to the [init command reference][init-doc] for usage.

!> Once a package is applied to the cluster, you do not want to change the
metadata in the `inventory` section as it severs the association between the
package and the `ResourceGroup` and will be destructive.

[init-doc]: /reference/cli/live/init/
