[starlark]: https://catalog.kpt.dev/starlark/v0.4/
[apply-replacements]: https://catalog.kpt.dev/apply-replacements/v0.1/

# Value propagation pattern

Generating a string value and propagating that value to another place
(or many other places) in your configuration is a very common pattern. 
In this guide, we will go through the recommended technique to 
do this value propagation using our [starlark] and [apply-replacements]
KRM functions. 

## String generation function

Sometimes, the value that we need to propagate is a concatenation of
other values that come from various other resource fields. In order
to generate the value we need to propagate, we can make use of the
[starlark] function.

For example, let's say we have a few context resources:

```yaml
# gcloud-config.yaml

apiVersion: v1
kind: ConfigMap
metadata:
   name: gcloud-config.kpt.dev
   annotations:
      config.kubernetes.io/local-config: "true"
data:
   domain: domain
   orgId: org-ID
   projectID: project
   region: region
   zone: zone
```

```yaml
# package-context.yaml

apiVersion: v1
kind: ConfigMap
metadata:
   name: kptfile.kpt.dev
   annotations:
      config.kubernetes.io/local-config: "true"
data:
   name: namespace
```

and a RoleBinding as follows:

```yaml
# rolebinding.yaml

apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: app-admin
  namespace: myns
subjects:
- kind: Group
  name: example-admin@example.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: app-admin
  apiGroup: rbac.authorization.k8s.io
```

For this example, our goal is to change the RoleBinding's Group name
from `example-admin@example.com` to the value `project-namespace-role@domain`.
In order to generate this value, we will need to look at various fields from our
context resources, concatenate them together, and then store the generated value somewhere. 

We can create a ConfigMap named `value-store`, which we will use to store the generated string in its
`data.group` field:

```yaml
# value-store.yaml

apiVersion: v1
kind: ConfigMap
metadata:
  name: value-store
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  group: example-group
```

Now we can use the [starlark] function to generate and store the desired string value.
This function allows you to run a Starlark script to modify or generate resources. In
our case, the functionConfig should look like the following:

```yaml
# generate-rolebinding-group.yaml

apiVersion: fn.kpt.dev/v1alpha1
kind: StarlarkRun
metadata:
  name: generate-rolebinding-group.yaml
  annotations:
    config.kubernetes.io/local-config: "true"
source: |-
  load("krmfn.star", "krmfn")
  
  def generate_group(resources, role):
    group = ""
    value_store = {}
    for r in resources:
      if krmfn.match_gvk(r, "v1", "ConfigMap") and krmfn.match_name(r, "gcloud-config.kpt.dev"):
        project = r["data"]["projectID"]
        domain = r["data"]["domain"]
      if krmfn.match_gvk(r, "v1", "ConfigMap") and krmfn.match_name(r, "kptfile.kpt.dev"):
        namespace = r["data"]["name"]
      if krmfn.match_gvk(r, "v1", "ConfigMap") and krmfn.match_name(r, "value-store"):
        value_store = r
    group = project + "-" + namespace + "-" + role + "@" + domain
    value_store["data"]["group"] = group
  generate_group(ctx.resource_list["items"], "app-admin")
```

## Value propagation function

Now that we have a function that can generate the desired value, we will need to
configure another function to propagate the value to the desired place(s). 

We can achieve this with the [apply-replacements] function. The apply-replacements
function is a wrapper for the [kustomize replacements](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/replacements/) 
feature, and can be used to copy a value from a provided source to any number of specified targets. 
In our case, the functionConfig looks like the following: 

```yaml
# propagate-rolebinding-group.yaml

apiVersion: fn.kpt.dev/v1alpha1
kind: ApplyReplacements
metadata:
   name: propagate-rolebinding-group
   annotations:
      config.kubernetes.io/local-config: "true"
replacements:
   - source:
        kind: ConfigMap
        name: value-store
        fieldPath: data.group
     targets:
        - select:
             kind: RoleBinding
             name: app-admin
          fieldPaths:
             - subjects.[kind=Group].name
```

This functionConfig will configure [apply-replacements] to copy the value
we stored in `value-store` to the RoleBinding Group. 

## Running the functions

In order to run the configured functions, we can have the following pipeline in our Kptfile:

```yaml
# Kptfile

apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/starlark:v0.4
      configPath: generate-rolebinding-group.yaml
    - image: gcr.io/kpt-fn/apply-replacements:v0.1
      configPath: propagate-rolebinding-group.yaml
```

After running these two functions with `kpt fn render`, we should see the value of our 
RoleBinding group change from `example-admin@example.com` to `project-namespace-role@domain` as desired.

## Summary

With the above pattern and workflow, you can easily generate and propagate
common values to various places of your configuration.
