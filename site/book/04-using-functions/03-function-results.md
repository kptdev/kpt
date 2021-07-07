In kpt, the counterpart to Unix philsophophy of "everything is a file" is "everything is a
Kubernetes resource". This also extends to the results of executing functions using `eval` or
`render`. In addition to providing a human-readable terminal output, these commands provide
structured results which can be consumed by other tools. This enables you to build robust UI layers
on top of kpt. For example:

- Create a custom dashboard that shows the results returned by functions
- Annotate a GitHub Pull Request with results returned by a validator function at the granularity of individuals fields

In both `render` and `eval`, structured results can be enabled using the `--results-dir` flag.

For example:

```shell
$ kpt fn render wordpress --results-dir /tmp
Package "wordpress/mysql":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"

Package "wordpress":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1"

Successfully executed 3 function(s) in 2 package(s).
For complete results, see /tmp/results.yaml
```

The results are provided as resource of kind `FunctionResultList`:

```yaml
# /tmp/results.yaml
apiVersion: kpt.dev/v1
kind: FunctionResultList
metadata:
  name: fnresults
exitCode: 0
items:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/kubeval:v0.1
    exitCode: 0
```

Let's see a more interesting result where the `kubeval` function catches a validation issue.
For example, change the value of `port` field in `service.yaml` from `80` to `"80"` and
rerun:

```shell
$ kpt fn render wordpress --results-dir /tmp
Package "wordpress/mysql":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"

Package "wordpress":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"
[FAIL] "gcr.io/kpt-fn/kubeval:v0.1"
  Results:
    [ERROR] Invalid type. Expected: integer, given: string in object "v1/Service/wordpress" in file "service.yaml" in field "spec.ports.0.port"
  Exit code: 1

For complete results, see /tmp/results.yaml
```

The results resource will now contain failure details:

```yaml
# /tmp/results.yaml
apiVersion: kpt.dev/v1
kind: FunctionResultList
metadata:
  name: fnresults
exitCode: 1
items:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/kubeval:v0.1
    exitCode: 1
    results:
      - message: "Invalid type. Expected: integer, given: string"
        severity: error
        resourceRef:
          apiVersion: v1
          kind: Service
          name: wordpress
        field:
          path: spec.ports.0.port
        file:
          path: service.yaml
```
