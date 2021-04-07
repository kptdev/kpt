A kpt function (also known as a _KRM function_) is a containerized program that can perform CRUD
operations on KRM resources stored on the local filesystem. kpt functions are the extensible
mechanism to automate mutation and validation of KRM resources. Some example use cases:

- Enforce all `Namespace` resources to have a `cost-center` label.
- Add a label to resources based on some filtering criteria
- Use a `Team` custom resource to generate a `Namespace` and associated organization-mandated
  defaults (e.g. `RBAC`, `ResourceQuota`, etc.) when bootstraping a new team
- Bulk transformation of all `PodSecurityPolicy` resources to improve the security posture.

Since functions are containerized, they can encapsulate different toolchains, languages, and
runtimes. For example, the function container image can encapsulate:

- A Go binary built using kpt's official library
- Wrap an existing KRM tool such as `kubeval`
- Invoke a bash script performing low-level operations
- The interpreter for "executable configuration" such as `Starlark` or `Rego`

To astute readers, this model will sound familiar: functions are the client-side analog to
Kubernetes controllers: </br>

|                  | Client-side              | Server-side       |
| ---------------- | ------------------------ | ----------------- |
| **Orchestrator** | kpt                      | Kubernetes        |
| **Data**         | YAML files on filesystem | resources on etcd |
| **Programs**     | functions                | controllers       |

</br> Just as Kubernetes system orchestrates server-side programs, kpt CLI orchestrates client-side
programs operating on configuration. By standardizing the input and output of the function
containers, and how the containers are executed, kpt can provide the following guarantees:

- Functions are interoperable
- Functions can be chained together in pipeline
- Functions are hermetic. For correctness, security and speed, it's desirable to be able to run
  functions hermetically without any privileges; preventing out-of-band access to the host
  filesystem and networking.

We will discuss the Functions Specification Standard in detail in Chapter 5. At a high level, a
function execution looks like this:

![img](/static/images/func.svg)

where:

- `input items`: The input list of KRM resources to operate on.
- `output items`: The output list obtained from adding, removing, or modifying items in the input.
- `functionConfig`: An optional meta resource used to parameterize this invocation of the function.
- `results`: An optional meta resource emitted by the function for observability and debugging
  purposes.

Naturally, functions can be chained together in a pipeline:

![img](/static/images/pipeline.svg)

There are two different commands that execute a function corresponding to two fundamentally
different workflows:

- `kpt fn render`: Executes the pipeline of functions declared in the package and its subpackages.
  This is a declarative way to run functions.
- `kpt fn eval`: Executes a given function on the package. The image to run and the `functionConfig`
  is specified as CLI argument. This is an imperative way to run functions. Since the user
  explicitly ask for the function to be executed, an imperative invocation can be more privileged
  and low-level than an declarative invocation. For example, it can optionally operate on meta
  resources or have access to the host system.

We will discuss how to run functions in Chapter 4 and how to develop functions in Chapter 5.
