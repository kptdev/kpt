# CRD Status Convention

To enable kpt to calculate the [reconcile status] for CRDs, this document
provides additional conventions for status conditions following the [Kubernetes
API Guideline]. Custom controllers should use the following conditions types to
signal whether a resource has been fully reconciled, and whether it has
encountered any problems:

- `Reconciling`: Indicates that the resource does not yet match its spec. i.e.
  the desired state as expressed in the resource spec object has not been fully
  realized in the cluster. A value of `"True"` means the controller is in the
  process of reconciling the resource while a value of `"False"` means there are
  no work left for the controller.
- `Stalled`: Indicates that the controller is not able to make the expected
  progress towards reconciling the resource. The cause of this status can be
  either that the controller observes an actual problem (like a pod not being
  able to start), or that something is taking longer than expected (similar to
  the `progressDeadlineSeconds` timeout on Deployments). If this condition is
  `"True"`, it should be interpreted that something might be wrong. It does not
  mean that the resource will never be reconciled. Most process in Kubernetes
  retry forever, so this should not be considered a terminal state.

CRDs should also set the `observedGeneration` field in the status object, a
pattern already common in the built-in types. The controller should update this
field every time it sees a new generation of the resource. This allows the kpt
library to distinguish between resources that do not have any conditions set
because they are fully reconciled, from resources that have no conditions set
because they have just been created.

An example of a resource where the latest change has been observed by the
controller which is currently in the process of reconciling would be:

```yaml
apiVersion: example.com
kind: Foo
metadata:
  generation: 12
  name: bar
spec:
  replicas: 1
status:
  observedGeneration: 12
  conditions:
    - lastTransitionTime: "2020-03-25T21:20:38Z"
      lastUpdateTime: "2020-03-25T21:20:38Z"
      message: Resource is reconciling
      reason: Reconciling
      status: "True"
      type: Reconciling
    - lastTransitionTime: "2020-03-25T21:20:27Z"
      lastUpdateTime: "2020-03-25T21:20:39Z"
      status: "False"
      type: Stalled
```

The calculated reconcile status for this resource is `InProgress`.

[kubernetes api guideline]:
  https://www.google.com/url?q=https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md%23typical-status-properties&sa=D&ust=1585160635349000&usg=AFQjCNE3ncANdus3xckLj3fkeupwFUoABw
[reconcile status]: /book/06-deploying-packages/?id=reconcile-status
