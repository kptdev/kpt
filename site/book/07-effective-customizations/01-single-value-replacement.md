## Scenario

I have a single value replacement in my package. I don’t want package consumers 
to look through all the yaml files to find the value I want them to set. It 
seems easier to just create a parameter for this value and have the user look 
at Kptfile for inputs.

Example storage bucket:

```yaml
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: my-bucket # kpt-set: ${project-id}-${name}
  namespace: ns-test # kpt-set: ${namespace}
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
    cnrm.cloud.google.com/project-id: my-project # kpt-set: ${project-id}
spec:
  storageClass: standard # kpt-set: ${storage-class}
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
```

The corresponding Kptfile:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: bucket
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2
      configMap:
        name: todo-bucket-name
        namespace: todo-namespace
        project-id: todo-project-id
        storage-class: standard
```


## Problems

1. With package popularity the single values inevitably expand to provide a 
facade to a large portion of the data.  That defeats the purpose of minimizing 
the cognitive load.  With this small example almost half of the StorageBucket configuration is now covered with parameters.
1. Some values like resource names are used as references so setting them in 
one place needs to trigger updates in all the places where they are referenced.
1. If additional resources that have similar values are added to the package 
new string replacements need to be added.  In this case everything will need
to also be marked up with project ID and namespace.
1. If a package is used as a sub-package the string replacement parameters need 
to be surfaced to the parent package and if the parent package already expects 
some values to be set and the parameters do not exist, the sub-package needs to 
be updated.

## Solutions:

1. kpt allows the user to edit a particular value directly in the configuration 
data and will handle upstream merge.  When [editing the yaml] directly the 
consumers are not confined to the parameters that the package author has 
provided.  [kpt pkg update] merges the local edits made by consumer with the 
changes in the upstream package made by publisher. In this case `storageClass` 
can be set directly by the user.
1. Attributes like resource names which are often updated by consumers to add 
prefix or suffix (e.g. *-dev, *-stage, *-prod, na1-*, eu1-*) are best handled 
by the [ensure-name-substring] function that will handle dependency updates as 
well as capture all the resources in the package.
1. Instead of setting a particular value on a resource a bulk operation can be 
applied to all the resources that fit a particular interface.  This can be done 
by a custom function or by [set-namespace], [search-and-replace] , [set-labels] 
and [set-annotations] functions.

New bucket configuration:

```yaml
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: bucket
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
spec:
  storageClass: standard
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
```

The suggested customizations are now in the Kptfile:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: bucket
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.2.0
      configMap:
        namespace: example-ns
    - image: gcr.io/kpt-fn/ensure-name-substring:v0.1.1
      configMap:
        prepend: project111-
    - image: gcr.io/kpt-fn/set-annotations:v0.1.4
      configMap:
        cnrm.cloud.google.com/project-id: project111
```

The resource configuration YAML doesn't need to be marked up with where the 
namespace value needs to go.  The [set-namespace] function is smart enough to 
find all the appropriate resources that need the namespace.

We have put in the starter name `bucket` and have an [ensure-name-substring] 
that shows the package consumer that the project ID prefix is what we suggest.
However if they have a different naming convention they can alter the name 
prefix or suffix on all the resources in the pacakge.

Since we are trying to set the annotation to the project ID we can use the 
[set-annotations] function one time and the annotation are going to be set on 
all the resources in the package.  If we add additional resources or whole 
sub packages we will get the consistent annotations across all resources 
without having to find all the places where annotations can go.

[editing the yaml]: /book/03-packages/03-editing-a-package
[kpt pkg update]: /book/03-packages/05-updating-a-package
[ensure-name-substring]: https://catalog.kpt.dev/ensure-name-substring/v0.1/
[search-and-replace]: https://catalog.kpt.dev/search-replace/v0.2/
[set-labels]: https://catalog.kpt.dev/set-labels/v0.1/
[set-annotations]: https://catalog.kpt.dev/set-annotations/v0.1/
[set-namespace]: https://catalog.kpt.dev/set-namespace/v0.2/