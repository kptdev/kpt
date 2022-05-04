If you look at the config workflows, you will notice that creating variant of a package is a very common and frequent operation, so reducing the manual steps required in creating a variant can have significant benefits for the package consumers. In this guide, we will look at some techniques/patterns that a package author can use to enable automatic variant construction of a package.

kpt packages comes in two flavors:  `abstract package` and `deployable package instance`. `abstract` package is a reususable package that is used to create deployable package instances that are deployed to a kubernetes cluster. In programming language terms, you can think of `abstract` packages as the classes and  `deployable` packages as the instances of the classes. `deployable` instances of package are also referred as `variant` of the package. 

Figure below shows a `package catalog` on the left that has `abstract` packages and `deployable instances` on the right. A good pattern is to keep the abstract packages and instance packages in separate repos and typically `deployable instances` repo will be setup to auto deploy to a kubernetes cluster using tools such as `config-sync`, `flux`, `argocd`.

![variant constructor pkg repo diagram](img/variant-constructor-pkg-repo-diagram.png)


Resources in an `abstract` package have placeholder values that needs to be substituated with actual values to make them ready for deployment. 
For example, name of the namespace resource below has `example` as a placeholder value. This is a part of the `abstract package`.

```yaml
apiVersion:v1
kind: Namespace
metadata:
	name: example # <-- this is a placeholder value
```


A kpt package contains kubernetes resources. Whenever you are creating a variant of the package, first step is to ensure unique identity of the resources in that variant. For example, if the abstract package contained a `namespace` resource, then the variant package will have namespace resource corresponding to that variant.

In a kubernetes cluster, resources are identified by their group, version, kind, namespace and name (aka GVKNN). If resource is cluster scoped, then the `metadata.name` uniquely identifies the resource in a cluster. If the resource is namespace scoped, then (namespace, name) together identifies the resource uniquely.

If you look at the steps involved in variant construction, it involves:
1. set of functions that can help in updating unique identity of the resources
2. Input to the functions for step 1

[kpt-function-catalog](https://catalog.kpt.dev) provides two function that helps with customizing the identify of the resources:
1. [set-namespace](https://catalog.kpt.dev/set-namespace/v0.3/): sets the namespace for all resources in the package.
2. [ensure-name-substring](https://catalog.kpt.dev/ensure-name-substring/v0.2/): sets the name of the resources in the package.

Now let's talk about the input to the functions. In most cases, variant's name (deployable package instance name) itself can be used to derive unique identity of the resources in the variant. For example, if I create a variant of `microservice` package, I will name the deployable package instance to `user-service` or `order-service`. So if the package's name is available to the functions, then they can use it to customize the name/namespace of the resources. So, starting with `kpt v1.0.0-beta.14+` , kpt makes `package's name` available in a local `ConfigMap` at a well-known path `pkg-context.yaml` that contains the package instance name and is available to functions during `kpt fn render` or `kpt fn eval`.

Examples of `pkg-context.yaml` for abstract and deployable instance, For abstract package the value of `name` field is `example` and for deployable instance it contains actual value of the package name.

```yaml
# package context for an abstract package.
# This is automatically created on `kpt pkg init`
apiVersion: v1
kind: ConfigMap
data:
	name: example
```

```yaml
# package context for a deployable instance of a package. This is automatically
# populated during `kpt pkg get`.
apiVersion: v1
kind: ConfigMap
data:
	name: my-pkg-instance
```

kpt supports a way to create `deployable pkg instance` such that `pkg-context` is automatically populated with the deployable instance's name.

```sh
$ kpt pkg get <pkg-location> my-pkg-instance --for-deployment
```

Now, let's look at how to provide the input to the functions.

So if you are using `set-namespace` function in your package, then `set-namespace` function supports reading input from `pkg-context.yaml`. Here is an example:

```Kptfile
...
pipeline:
	mutators:
		- img: set-namespace:v0.3.4
		  configPath: pkg-context.yaml
...
```

So for a package consumer, creating a deployable instance involves the following:

```
# pick name of the deployable instance say `my-pkg-instance`
$ kpt pkg get <path-to-abstract-pkg> <my-pkg-instance> --for-deployement

$ kpt fn render <my-pkg-instance>

# That's it. Your pkg instance is all ready to deploy now.
```

## Summary
So with the above pattern and workflow, you can how package publisher can enable automatic customization of deployable instance of a package with minimal input i.e. package instance name.

## Future plans

Currently `--for-deployment` steps invokes built-in function to generate `pkg-context`. We would like to make it extensible for users to invoke their custom function for deploy workflow.