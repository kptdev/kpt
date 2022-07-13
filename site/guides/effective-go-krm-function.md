# Effective GO KRM functions

This guide gives tips to effectively write a KRM function. 

This guide is for advanced kpt function users who find the [catalog.kpt.dev] cannot fulfil their needs 
and want to design their own KRM functions. 

Suggest reading [Developing in Go] first

## Prerequisites

- [Install kpt]
- [Install Docker]

## Setup 

<!--- TODO: Use scaffolding to generate the get-started package --->

We start from a "get-started-runner" package which contains a `main.go` file with some scaffolding code.


```shell
# Set your KRM function name.
export FUNCTION_NAME=<YOUR FUNCTION NAME>
export GOPATH=$(go env GOPATH)
export FUNCTION_PATH=github.com/<YOUR USERNAME>

# Create and direct to your Go working directory
mkdir -p $GOPATH/src/${FUNCTION_PATH} && cd $GOPATH/src/${FUNCTION_PATH}

# Get the "get-started" package.
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/go/get-started-runner@master ${FUNCTION_NAME}

cd ${FUNCTION_NAME}

# Initialize Go module and install the kpt KRM function SDK.
go mod init && go mod tidy -compat=1.17
```

## Write your KRM function code in Go 

In the main.go, you should have
```go
var _ fn.Runner = &FunctionX{}

type FunctionX struct {
    // TODO: Modify with your expected function config.
    FnConfigBool bool
    FnConfigInt  int
    FnConfigFoo string
}

func (r *FunctionX) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
      // TODO: Add your KRM resource mutate or validate logic.
}

func main() {
    if err := fn.AsMain(&FunctionX{}); err != nil {
        os.Exit(1)
    }
}
```
`FunctionX` implements the [`Runner`] interface that can process the input KRM resources as [`fn.ResourceList`], it initializes `fn.KubeObject` to hold the KRM resources,
so that you can use [`fn.KubeObject` and `fn.SubObject`] methods directly. After `Run`, it will convert the modified `fn.KubeObjects` to KRM resources.

### Define configures

If you need to use configurable variables, you can define them as `FunctionX` fields.
Otherwise you can skip this step and move to next. 
```go 
type FunctionX struct {
  FnConfigBool bool
  FnConfigInt  int
  FnConfigFoo string
}
```

For example, define a `SetImage` and add two variables to compare-and-swap the image value will be like:
```go
type SetImage struct {
  OldImage string // Existing image
  NewImage string // New image to replace
}
```

The `SetImage` is a KRM resource. It should be passed from the input as:
```yaml
apiVersion: config.kubernetes.io/v1
kind: ResourceList
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  # Kind is required to match the Runner struct
  kind: SetImage
  metadata:
     name: try-out
  oldImage: example
  newImage: <YOUR NEW IMAGE NAME>
items:
...
```
<!--- TODO: we should not require users to understand and provide the input ResourceList. 
We should build a test infra that users only provide the KRM resources--->

## Write the main logic in `Run`

The SDK will initialize a slice of `*fn.KubeObject` to hold your KRM resources. You will need to pass the
KRM resources from the input in `items` fields.
```go
func (r *FunctionX) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
      // TODO: Add your KRM resource mutate or validate logic.
}

func main() {
    if err := fn.AsMain(&FunctionX{}); err != nil {
        os.Exit(1)
    }
}
```

<!--- TODO: we should not require users to understand and provide the input ResourceList. 
We should build a test infra that users only provide the KRM resources--->

### Select KRM resources 
The `fn.KubeObjects` is a slice of `*fn.KubeObject`, that you can apply some select logic to easily choose
the target KRM resources. See below example on using `Where` and `WhereNot` to filter different types of resources.
```go
func (r *YourFunction) Run(context *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
	// namespaceScoped contains only namespace scoped resources  
    namespaceScoped := objects.Where(func(o *fn.KubeObject) bool { return o.IsNamespaceScoped() }) 
    // clusterScoped contains only cluster scoped resources
    clusterScoped := objects.Where(func(o *fn.KubeObject) bool { return o.IsClusterScoped() })
    // customDeployment contains all resources of Kind "CustomDeployment", in Group "fn.kpt.dev" with any Versions.  
    customDeployment := objects.Where(fn.IsGVK("fn.kpt.dev", "", "CustomDeployment") })
    // excluded contains all resources except namespace objects
    excluded := objects.WhereNot(fn.IsGVK("v1", "", "Namespace") })
}
```

### Read and write a field spec path of a KRM resource

Like [unstructured.Unstructured], `fn.KubeObject` (and `fn.SubObject`) provides a series of methods to 
let you read and write different resources types.

```go
func (r *YourFunction) Run(context *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
	// Get first deployment object.
	deployment := items.Where(fn.IsGVK("apps", "v1", "Deployment")).Where(func(o *fn.KubeObject) bool{return o.GetName() == "nginx"})[0]
	// Get the int value from deployment `spec.replicas`
	replicas := deployment.NestedInt64OrDie("spec", "replicas")
	fn.Logf("replicas is %v\n", replicas)
    // Get the boolean value from deployment `spec.paused`
    paused := deployment.NestedBoolOrDie("spec", "paused")
    fn.Logf("paused is %v\n", paused)
    // Update strategy from Recreate to RollingUpdate.
    if strategy := obj.NestedStringOrDie("spec", "strategy", "type"); strategy == "Recreate" {
    	obj.SetNestedStringOrDie("RollingUpdate", "spec", "strategy", "type")
    }
}
```

Besides the [unstructured.Unstructured] style, you can also run functions on each sub-field as a `fn.SubObject`  
```go
func (r *YourFunction) Run(context *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
    // Get first deployment object.
	deployment := items.Where(fn.IsGVK("apps", "v1", "Deployment")).Where(func(o *fn.KubeObject) bool{return o.GetName() == "nginx"})[0]
    // Get a spec as a `SubObject`
	spec := deployment.GetMap("spec")
    // Get integer from SubObject spec
    replicas = spec.GetInt("replicas")
    fn.Logf("replicas is %v\n", replicas)
    // Get the SubObject from another SubObject
    nodeSelector := spec.GetMap("template").GetMap("spec").GetMap("nodeSelector")
    if nodeSelector.GetString("disktype") != "ssd" {
       nodeSelector.SetNestedStringOrDie("ssd", "disktype")
    }
}
```

### Copy `KubeObject` to a typed struct

If you already have some struct to define a KRM resource (like `corev1.ConfigMap`), you can switch the `KubeObject` 
to the other type via `As`
```go
func (r *YourFunction) Run(context *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects) {
    deploymentObject := objects.WhereNot(fn.IsGVK("apps", "v1", "Deployment") })[0]
    deploymentSpec := deploymentObject.GetMap("spec")
	var dpSpec appsv1.DeploymentSpec
    deploymentSpec.As(&dpSpec)
    dpSpec.Size()
}
```

<!-- TODO: We need a "Test the KRM function" section, which reuqires the SDK to provide the test infra so users only provide the input resource and expected output in YAML--->

[Install kpt]:
  https://kpt.dev/installation/
[Install Docker]:
  https://docs.docker.com/get-docker/
[`fn.ResourceList`]:
  https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn#ResourceList
[`fn.KubeObject` and `fn.SubObject`]:
  https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn#KubeObject
[Golang]:
  https://go.dev/doc/gopath_code
[`Runner`]:
  https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn#Runner
[unstructured.Unstructured]:
  https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
[Developing in Go]:
  https://kpt.dev/book/05-developing-functions/02-developing-in-Go
[catalog.kpt.dev]: 
  https://catalog.kpt.dev/