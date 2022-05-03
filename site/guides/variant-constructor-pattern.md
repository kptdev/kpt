As shown in the diagram below, kpt packages comes in two flavors:
`abstract` packages and concrete package instances. `abstract` package is a reususable package that is used to create concrete package instances that are deployed to a kubernetes cluster. In programming language terms, you can think of `abstract` packages as the classes and  `concrete` packages as the instances of the classes. `concrete` instances of package are also referred as `variant`  or `deployable instance` of the package. 

Figure below shows a `package catalog` on the left that has `abstract` packages and `deployable instances` on the right. A good pattern is to keep the abstract packages and instance packages in separate repos and typically `deployable instances` repo will be setup to auto deploy to a kubernetes cluster using tools such as `config-sync`, `flux`, `argocd`.

![variant constructor pkg repo diagram](img/variant-constructor-pkg-repo-diagram.png)


Resources in an `abstract` package will have placeholder values that needs to be substituated with actual values to make them ready for deployment. 
For example, name of the namespace resource below has `example` as a placeholder value. This is a part of the `abstract package`.

```yaml
apiVersion:v1
kind: Namespace
metadata:
	name: example # <-- this is a placeholder value
```

If you look at the config workflow, you will note that creating variant of an abstract package is a very common and frequent operation, so reducing the manual steps required in creating a variant can have significant benefits for the users. Next, we are going to look at the pattern and a workflow that shows how a package producer can enables automatic customization of variants.

A kpt package contains kubernetes resources. Whenever you are creating a variant of the package, first step is to ensure unique identity of the resources in that variant. For example, if the abstract package contained a `namespace` resource, then the variant needs to have namespace resource corresponding to that variant.

In a kubernetes cluster, resources are identified by their group, version, kind, namespace and name (aka GVKNN). If resource is cluster scoped, then the `metadata.name` uniquely identifies the resource in a cluster. If the resource is namespace scoped, then (namespace, name) together identifies the resource uniquely.

If you look at the steps involved in variant construction, it involves:
1. set of functions that can help in ensuring unique identity of the resources
2. Input to the functions for step 1

kpt-function-catalog provides two function that helps with customizing the identify of the resources:
1. set-namespace: sets the namespace for all resources in the package.
2. ensure-name-substring: sets the name of the resources in the package.

Now let's talk about the input to the functions. In most cases, variant's name (concrete package instance name) itself can be used to derive unique identity of the resources in the variant. For example, if I create variant of `microservice`, I will name the deployable package instance as `user-service` or `order-service`. So if the package's name is available to the functions, then they can use that to customize the name/namespace of the resources. So, starting with `kpt v1.0.0-beta.14+` , kpt makes `package's name` available in a local `ConfigMap` at a well-known path `pkg-context.yaml` that contains the package instance name and is available to functions during `kpt fn render` or `kpt fn eval`.

Examples of `pkg-context.yaml` for abstract and deployable instance, For abstract package the value of `name` field is `example` and for deployable instance it contains actual value of the package name.

```yaml
apiVersion: v1
kind: ConfigMap
data:
	name: example
```

```yaml

apiVersion: v1
kind: ConfigMap
data:
	name: my-pkg-instance
```

For `namespace-provisioning` package, we use `set-namespace` function to ensure unique identify for the resources and `set-namespace` function supports reading input from `pkg-context.yaml`.

```Kptfile

pipeline:
	mutators:
		- img: set-namespace:v0.3.4
		  configPath: pkg-context.yaml
```

kpt supports a way to create `deployable pkg instance` such that `pkg-context` is automatically populated with the deployable instance's name.

```sh
kpt pkg get <pkg-location> my-pkg-instance --for-deployment
```

So, now creating a deployable instance involves the following:

```
# pick name of the deployable instance say `some-ns`
$ kpt pkg get <> <some-ns> --for-deployement

$ kpt fn render <some-ns>

# That's it. Your pkg instance is all ready to deploy now.
```

## Summary
So with the above pattern and workflow, you can how package publisher can enable automatic customization of deployable instance of a package with minimal input i.e. package instance name.

## Future plans

Currently `--for-deployment` steps invokes built-in function to generate `pkg-context`. We would like to make it extensible for users to invoke their custom function for deploy workflow.