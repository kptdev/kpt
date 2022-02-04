# Config Connector Status Convention

`kpt` includes custom rules for [Config Connector] resources to make them easier to work
with. This document describes how kpt uses fields and conditions on Config Connector
resources to compute [reconcile status].

Config Connector resources expose the `observedGeneration` field in the status
object, and `kpt` will always report a resource as being `InProgress` if the
`observedGeneration` doesn't match the value of `metadata.generation`.

If the `Ready` condition is `True`, a Config Connector resource will be reported
as `Current`, i.e it has been successfully reconciled.

If the `Ready` condition is `False`, `kpt` will look at the `Reason` field on the
condition object to determine whether the resource is making progress towards
reconciliation. The possible values mirrors those used by [Config Connector events].
If the value is one of the following, the resource is considered to have failed
reconciliation:
- `ManagementConflict`
- `UpdateFailed`
- `DeleteFailed`
- `DependencyInvalid`

Note that this doesn't necessarily mean it could never successfully reconcile. 
The Config Connector controller will keep retrying. But it does mean that the
resource is in a state where an external change is most likely needed to resolve
the issue. Typical examples would be missing permissions or an API that has not
been enabled.

Similar to all other resources, a Config Connector resource will be in the `Terminating`
state if the `metadata.deletionTimestamp` is set, and considered fully deleted when
the resource no longer exists in the cluster.

[reconcile status]: /book/06-deploying-packages/?id=reconcile-status
[Config Connector]: https://cloud.google.com/config-connector/docs/overview
[Config Connector events]: https://cloud.google.com/config-connector/docs/how-to/monitoring-your-resources
