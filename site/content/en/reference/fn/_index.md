---
title: "Fn"
linkTitle: "fn"
type: docs
weight: 4
description: >
   Generate, transform, and validate configuration files.
---
<!--mdtogo:Short
    Generate, transform, and validate configuration files.
-->

<!--mdtogo:Long-->
| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

Functions are executables ([that you can write](#developing-functions))
packaged in container images which accept a collection of Resource
configuration as input, and emit a collection of Resource configuration as output.
<!--mdtogo-->

Functions may be used to:

* Generate configuration from templates, DSLs, CRD-style abstractions,
  key-value pairs, etc.-- e.g. expand Helm charts, JSonnet, Jinja, etc.
* Inject fields or otherwise modify configuration -- e.g.add init-containers,
  side-cars, etc
* Rollout configuration changes across an organization -- e.g.similar to
  https://github.com/reactjs/react-codemod
* Validate configuration -- e.g.ensure organizational policies are enforced

Functions may be run at different times depending on the function and
the organizational needs:

* as part of the build and development process
* as pre-commit checks
* as PR checks
* as pre-release checks
* as pre-rollout checks

### Examples
<!--mdtogo:Examples-->
```sh
# run the function defined by gcr.io/example.com/my-fn as a local container
# against the configuration in DIR
kpt fn run DIR/ --image gcr.io/example.com/my-fn
```

```sh
# run the functions declared in files under FUNCTIONS_DIR/
kpt fn run DIR/ --fn-path FUNCTIONS_DIR/
```

```sh
# run the functions declared in files under DIR/
kpt fn run DIR/
```
<!--mdtogo-->

#### Functions Catalog

[KPT Functions Catalog][catalog] documents a catalog of kpt
functions implemented using different toolchains.

#### Developing Functions

| Language   | Documentation               | Examples                    |
|------------|-----------------------------|-----------------------------|
| Typescript | [KPT Functions SDK][sdk-ts] | [examples][sdk-ts-examples] |
| Go         | [kustomize/kyaml][kyaml]    | [example][kyaml-example]    |

## Next Steps

* Learn how to [run functions].
* Find out how to structure a pipeline of functions from the [functions concepts] page.
* See more examples of functions in the functions [catalog].

[run]: run
[source]: source
[sink]: sink
[cfg]: ../cfg
[pkg]: ../pkg
[sdk-ts]: ../../guides/producer/functions/ts
[sdk-ts-quickstart]: ../../guides/producer/functions/ts/#develper-quickstart
[sdk-ts-examples]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src
[catalog]: ../../guides/consumer/function/catalog
[kyaml]: https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml
[kyaml-example]: https://github.com/kubernetes-sigs/kustomize/blob/master/functions/examples/injection-tshirt-sizes/image/main.go
[run functions]: ../../guides/consumer/function/
[functions concepts]: ../../concepts/functions/
