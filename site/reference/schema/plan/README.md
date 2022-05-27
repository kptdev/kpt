# Plan

The `kpt alpha live plan` command gives the option of outputting the plan in KRM format. This will be wrapped in a `ResourceList` 
resource so it can easily be piped into kpt functions for validation. 

The plan contains a list of actions (under `spec.actions`) with one item for each resource associated with the package being 
applied. Each entry in the list contains the `apiVersion`, `kind`, `name`, and `namespace` to identify the resource. It
also contains an `action` field, which defines which action will be taken on the resource during apply. It can have one of
the values:
* **Create**: The resource does not currently exist in the cluster and will be created.
* **Unchanged**: The resource exists in the cluster and there are no chnages to it.
* **Delete**: The resource exists in the cluster, but is not among the applied resources. Therefore it will be pruned.
* **Update**: The resource exists in the cluster and will be updated.
* **Skip**: No changes will be made to this resource due to the presence of one or more lifecycle directives.
* **Error**: An error was encountered for the resource.

There is also an `original` field which contains the resource from the cluster before apply (is it does exist), and
an `updated` field that contains the resource after apply (but the state in the cluster remains unchanged). Finally, there
is an `error` field that will have a more detailed error message in the cases where the value of the `action` field is `Error`.

The OpenAPI
[schema is available here](https://raw.githubusercontent.com/GoogleContainerTools/kpt/main/site/reference/schema/kptfile/kptfile.yaml).