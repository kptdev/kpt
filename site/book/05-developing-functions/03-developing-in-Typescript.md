kpt project provides an opinionated Typescript SDK for implementing functions.
It provides the following features:

- **General-purpose language:** Similar to Go, Typescript is a general-purpose
  programming language that provides:
  - Proper abstractions and language features
  - An extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation
- **Idiomatic:** The TS SDK provides a different level of abstraction compared
  to the Go library we saw previously. Instead of exposing the low-level YAML
  AST, resources are marshalled into native Typescript objects. As a result, you
  get a more idiomatic and high-level abstraction. Ideally, you should work with
  native data structure/object in each language and not think about YAML which
  is just a file format. Even though resources in configuration files are
  round-tripped to Typescript objects, the kpt orchestrator ensures that
  YAML-specific constructs such as comments are preserved. The obvious
  limitation of this high-level abstraction is that you cannot develop functions
  for manipulating YAML-specific constructs like comments.
- **Type-safety:** Kubernetes configuration is typed, and its schema is defined
  using the OpenAPI spec. Typescript has a sophisticated type system that
  accommodates the complexity of Kubernetes resources. The TS SDK enables
  generating Typescript classes for core and CRD types.
- **Batteries-included:** The TS SDK provides a simple, powerful API for
  querying and manipulating resources inspired by [document-oriented databases].
  It provides the scaffolding required to develop, build, test, and publish
  functions, allowing you to focus on implementing your business-logic.

## Quickstart

This quickstart will get you started developing a kpt function with the TS SDK.

### System Requirements

Currently supported platforms: amd64 Linux/Mac

- Install [kpt][download-kpt] and its dependencies
- Install [node][download-node]
  - The SDK requires `npm` version 6 or higher.
  - If installing node from binaries (i.e. without a package manager), follow
    these [installation instructions][install-node].

### Explore the Demo Functions Package

1. Get the `demo-functions` package:

   ```shell
   $ git clone --depth 1 https://github.com/GoogleContainerTools/kpt-functions-sdk.git
   ```

   All subsequent commands are run from the `demo-functions` directory:

   ```shell
   $ cd kpt-functions-sdk/ts/demo-functions
   ```

1. Install all dependencies:

   ```shell
   $ npm install
   ```

1. Run the following in a separate terminal to continuously build your function
   as you make changes:

   ```shell
   $ npm run watch
   ```

1. Explore the [`label_namespace`][label-namespace] transformer function. This
   function takes a given `label_name` and `label_value` to add the appropriate
   label to `Namespace` objects using the SDK's `addLabel` function.

   ```typescript
   import { addLabel, Configs } from "kpt-functions";
   import { isNamespace } from "./gen/io.k8s.api.core.v1";

   export const LABEL_NAME = "label_name";
   export const LABEL_VALUE = "label_value";

   export async function labelNamespace(configs: Configs) {
     const labelName = configs.getFunctionConfigValueOrThrow(LABEL_NAME);
     const labelValue = configs.getFunctionConfigValueOrThrow(LABEL_VALUE);
     configs
       .get(isNamespace)
       .forEach((n) => addLabel(n, labelName, labelValue));
   }
   ```

1. Run the `label_namespace` function on the `example-configs` directory:

   ```shell
   $ export CONFIGS=../../example-configs
   ```

   ```shell
   $ kpt fn eval $CONFIGS --exec "node dist/label_namespace_run.js" -- label_name=color label_value=orange
   ```

   As the name suggests, this function added the given label to all `Namespace`
   objects in the `example-configs` directory:

   ```shell
   $ git diff $CONFIGS
   ```

1. Try modifying the function in `src/label_namespace.ts` to perform other
   operations and rerun the function on `example-configs` to see the changes.

1. Explore validation functions like [validate-rolebinding]. Instead of
   transforming configuration, this function searches RoleBindings and returns a
   results field containing details about invalid subject names.

   ```typescript
   import { Configs, kubernetesObjectResult } from "kpt-functions";
   import { isRoleBinding } from "./gen/io.k8s.api.rbac.v1";

   const SUBJECT_NAME = "subject_name";

   export async function validateRolebinding(configs: Configs) {
     // Get the subject name parameter.
     const subjectName = configs.getFunctionConfigValueOrThrow(SUBJECT_NAME);

     // Iterate over all RoleBinding objects in the input
     // Filter those that have a subject we're looking for.
     const results = configs
       .get(isRoleBinding)
       .filter(
         (rb) =>
           rb && rb.subjects && rb.subjects.find((s) => s.name === subjectName)
       )
       .map((rb) =>
         kubernetesObjectResult(`Found banned subject '${subjectName}'`, rb)
       );

     configs.addResults(...results);
   }
   ```

1. Run `validate-rolebinding` on `example-configs`.

   ```shell
   $ kpt fn eval $CONFIGS --exec "node dist/validate_rolebinding_run.js" -- subject_name=alice@foo-corp.com
   ```

   Look at the changes made by the function:

   ```shell
   $ git diff $CONFIGS
   ```

1. Explore generator functions like [expand-team-cr]. This function generates a
   per-environment Namespace and RoleBinding object for each custom resource
   (CR) of type Team.

   ```typescript
   import { Configs } from "kpt-functions";
   import { isTeam, Team } from "./gen/dev.cft.anthos.v1alpha1";
   import { Namespace } from "./gen/io.k8s.api.core.v1";
   import { RoleBinding, Subject } from "./gen/io.k8s.api.rbac.v1";

   const ENVIRONMENTS = ["dev", "prod"];

   export async function expandTeamCr(configs: Configs) {
     // For each 'Team' custom resource in the input:
     // 1. Generate a per-enviroment Namespace.
     // 2. Generate RoleBindings in each Namespace.
     configs.get(isTeam).forEach((team) => {
       const name = team.metadata.name;

       ENVIRONMENTS.forEach((suffix) => {
         const ns = `${name}-${suffix}`;
         configs.insert(Namespace.named(ns));
         configs.insert(...createRoleBindings(team, ns));
       });
     });
   }

   function createRoleBindings(team: Team, namespace: string): RoleBinding[] {
     return (team.spec.roles || []).map((item) => {
       return new RoleBinding({
         metadata: {
           name: item.role,
           namespace,
         },
         subjects: roleSubjects(item),
         roleRef: {
           kind: "ClusterRole",
           name: item.role,
           apiGroup: "rbac.authorization.k8s.io",
         },
       });
     });
   }

   function roleSubjects(item: Team.Spec.Item): Subject[] {
     const userSubjects: Subject[] = (item.users || []).map(
       (user) =>
         new Subject({
           kind: "User",
           name: user,
         })
     );
     const groupSubjects: Subject[] = (item.groups || []).map(
       (group) =>
         new Subject({
           kind: "Group",
           name: group,
         })
     );
     return userSubjects.concat(groupSubjects);
   }
   ```

1. Run `expand-team-cr` on `example-configs`.

   ```shell
   $ kpt fn eval $CONFIGS --exec "node dist/expand_team_cr_run.js"
   ```

   Look at the changes made by the function:

   ```shell
   $ git diff $CONFIGS
   ```

## Next Steps

- Read the complete [Typescript SDK Developer Guide].
- Take a look at these [example functions] to better understand how to use the
  TS SDK.

[download-kpt]: /book/01-getting-started/01-system-requirements
[download-node]: https://nodejs.org/en/download/
[install-node]: https://github.com/nodejs/help/wiki/Installation/
[ts sdk api]: https://googlecontainertools.github.io/kpt-functions-sdk/api/
[label-namespace]:
  https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/label_namespace.ts
[validate-rolebinding]:
  https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/validate_rolebinding.ts
[expand-team-cr]:
  https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/expand_team_cr.ts
[example functions]:
  https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/
[document-oriented databases]:
  https://en.wikipedia.org/wiki/Document-oriented_database
[typescript sdk developer guide]: /sdk/ts-guide
