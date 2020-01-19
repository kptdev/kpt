## fn

Run local containers against Resource configuration

### Commands

[run], [sink], [source]

### Synopsis

Functions are executables packaged in container images which accept a collection of
Resource configuration as input, and emit a collection of Resource configuration as output.

| Command   | Description                                             |
|-----------|---------------------------------------------------------|
| [run]     | run containers locally to generate and transform config |
| [source]  | explicitly specify an input source to pipe to `run`     |
| [sink]    | explicitly specify an output sink to pipe to `run`      |

Functions may be used to:

- Generate configuration from Templates, DSLs, CRD-style abstractions, key-value pairs, etc. -- e.g.
  expand Helm charts, JSonnet, etc.
- Inject fields or otherwise modifying configuration -- e.g. add init-containers, side-cars, etc
- Rollout configuration changes across an organization -- e.g. similar to
  https://github.com/reactjs/react-codemod
- Validate configuration -- e.g. ensure Organizational policies are enforced

Functions may be run either imperatively with `kpt run DIR/ --image` or declaratively with
`kpt run DIR/` and specifying them in files.

Functions specified in files must contain an annotation to mark them as function declarations:

      annotations:
        config.kubernetes.io/function: |
          container:
            image: gcr.io/example.com/image:version
        config.kubernetes.io/local-config: "true"

Functions may be run at different times depending on the function and the organizational needs:

- as part of the build and development process
- as pre-commit checks
- as PR checks
- as pre-release checks
- as pre-rollout checks

#### Writing functions

There are several projects that may be used to quickly develop kpt functions:

| Name                | Language     | Examples        |
|---------------------|--------------|-----------------|
| [kpt-functions-sdk] | Typescript   |                 |
| [kustomize/kyaml]   | Golang       | [kyaml-example] |

### Examples

    # run the function defined by gcr.io/example.com/my-fn as a local container
    # against the configuration in DIR
    kpt fn run DIR/ --image gcr.io/example.com/my-fn

    # run the functions declared in files under FUNCTIONS_DIR/
    kpt fn run DIR/ --fn-path FUNCTIONS_DIR/

    # run the functions declared in files under DIR/
    kpt fn run DIR/

### Also See Command Groups

[cfg], [pkg]

### 

[run]: run.md
[source]: source.md
[sink]: sink.md

[cfg]: ../cfg/README.md
[pkg]: ../pkg/README.md
[kpt functions sdk]: https://github.com/GoogleContainerTools/kpt-functions-sdk
[kyaml]: https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml
[kyaml-example]: https://github.com/kubernetes-sigs/kustomize/blob/master/functions/examples/injection-tshirt-sizes/image/main.go