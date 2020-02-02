## fn

Generate, transform, validate configuration files using containerized functions

### Synopsis

Functions are executables ([that you can write](#developing-functions)) packaged in container images which accept a collection of
Resource configuration as input, and emit a collection of Resource configuration as output.

| Command  | Description                                             |
|----------|---------------------------------------------------------|
| [run]    | run containers locally to generate and transform config |
| [source] | explicitly specify an input source to pipe to `run`     |
| [sink]   | explicitly specify an output sink to pipe to `run`      |

**Data Flow**:  local configuration or stdin -> kpt [fn] (runs a container) -> local configuration or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

Functions may be used to:

* Generate configuration from templates, DSLs, CRD-style abstractions, key-value pairs, etc.-- e.g.

  expand Helm charts, JSonnet, Jinja, etc.

* Inject fields or otherwise modify configuration -- e.g.add init-containers, side-cars, etc
* Rollout configuration changes across an organization -- e.g.similar to

  https://github.com/reactjs/react-codemod

* Validate configuration -- e.g.ensure organizational policies are enforced

Functions may be run either imperatively with `kpt run DIR/ --image` or declaratively with
`kpt run DIR/` and specifying them in files.

Functions specified in files must contain an annotation to mark them as function declarations:

      annotations:
        config.kubernetes.io/function: |
          container:
            image: gcr.io/example.com/image:version
        config.kubernetes.io/local-config: "true"

Functions may be run at different times depending on the function and the organizational needs:

* as part of the build and development process
* as pre-commit checks
* as PR checks
* as pre-release checks
* as pre-rollout checks

#### Functions Catalog

[KPT Functions Catalog][catalog] repository documents a catalog of KPT functions implemented using different toolchains.

#### Developing Functions

Two projects that aid kpt function development:

| Language   | Documentation               | Examples                    |
|------------|-----------------------------|-----------------------------|
| Typescript | [kpt functions SDK][sdk-ts] | [examples][sdk-ts-examples] |
| Go         | [kustomize/kyaml][kyaml]    | [example][kyaml-example]    |

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
[sdk-ts]: https://googlecontainertools.github.io/kpt-functions-sdk/
[sdk-ts-quickstart]: https://googlecontainertools.github.io/kpt-functions-sdk/docs/develop-quickstart.html
[sdk-ts-examples]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src
[catalog]: https://googlecontainertools.github.io/kpt-functions-catalog/
[kyaml]: https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml
[kyaml-example]: https://github.com/kubernetes-sigs/kustomize/blob/master/functions/examples/injection-tshirt-sizes/image/main.go

