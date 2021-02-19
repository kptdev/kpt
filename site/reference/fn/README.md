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
| ----------------------- | ------------------------ |
| local files or stdin    | local files or stdout    |

Functions are executables (that you can [write][Functions Developer Guide])
which accept a collection of Resource configuration as input, and emit a
collection of Resource configuration as output.

Functions can be packaged as container images, starlark scripts, or binary
executables.

<!--mdtogo-->

Functions may be used to:

- Generate configuration from templates, DSLs, CRD-style abstractions,
  key-value pairs, etc.-- e.g. expand Helm charts, JSonnet, Jinja, etc.
- Inject fields or otherwise modify configuration -- e.g.add init-containers,
  side-cars, etc
- Rollout configuration changes across an organization -- e.g.similar to
  <https://github.com/reactjs/react-codemod>
- Validate configuration -- e.g.ensure organizational policies are enforced

Functions may be run at different times depending on the function and
the organizational needs:

- as part of the build and development process
- as pre-commit checks
- as PR checks
- as pre-release checks
- as pre-rollout checks

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

#### Using Functions

The [catalog] documents config functions implemented using different toolchains
like starlark, typescript, and golang.

#### Developing Functions

See the [Functions Developer Guide] for more on producing functions.

| Language   | Documentation               | Examples                    |
| ---------- | --------------------------- | --------------------------- |
| Typescript | [Typescript SDK]            | [examples][sdk-ts-examples] |
| Go         | [Golang Libraries]          | [example][golang-example]   |
| Shell      | Use builtin shell functions | [example][shell-example]    |
| Starlark   | [Starlark SDK]              | [example][starlark-example] |

## Next Steps

- Learn how to [run functions].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.

[Functions Developer Guide]: ../../guides/producer/functions/
[Typescript SDK]: ../../guides/producer/functions/ts/
[sdk-ts-examples]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/
[Golang Libraries]: ../../guides/producer/functions/golang/
[golang-example]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go/set-namespace/
[shell-example]: https://github.com/kubernetes-sigs/kustomize/blob/master/functions/examples/template-heredoc-cockroachdb/image/cockroachdb-template.sh
[Starlark SDK]: ../../guides/producer/functions/starlark/
[starlark-example]: https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/functions/starlark/set_namespace.star
[run functions]: ../../guides/consumer/function/
[functions concepts]: ../../concepts/functions/
