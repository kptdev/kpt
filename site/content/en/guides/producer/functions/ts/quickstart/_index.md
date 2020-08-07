---
title: "TypeScript Quickstart"
linkTitle: "TypeScript Quickstart"
weight: 6
type: docs
description: >
   Quickstart for Developing Config Functions
---

## Developer Quickstart

This quickstart will get you started developing a config function with the
TypeScript SDK, using an existing kpt-functions package. Config functions are
any functions which implement the `KptFunc` interface documented in the
[TS SDK API].

### Prerequisites

#### System Requirements

Currently supported platforms: amd64 Linux/Mac

#### Setting Up Your Local Environment

- Install [node][download-node]
- Install [docker][install-docker]
- Install [kpt][download-kpt] and add it to your $PATH

### Explore the Demo Functions Package

1. Get the `demo-functions` package:

   ```sh
   git clone --depth 1 https://github.com/GoogleContainerTools/kpt-functions-sdk.git
   ```

   All subsequent commands are run from the `demo-functions` directory:

   ```sh
   cd kpt-functions-sdk/ts/demo-functions
   ```

1. Install all dependencies:

   ```sh
   npm install
   ```

1. Run the following in a separate terminal to continuously build your
   function as you make changes:

   ```sh
   npm run watch
   ```

1. Explore the [`label_namespace`][label-namespace] transformer function. This
   function takes a given `label_name` and `label_value` to add the
   appropriate label to `Namespace` objects using the SDK's `addLabel`
   function.

  ```typescript
  import { addLabel, Configs } from 'kpt-functions';
  import { isNamespace } from './gen/io.k8s.api.core.v1';

  export const LABEL_NAME = 'label_name';
  export const LABEL_VALUE = 'label_value';

  export async function labelNamespace(configs: Configs) {
    const labelName = configs.getFunctionConfigValueOrThrow(LABEL_NAME);
    const labelValue = configs.getFunctionConfigValueOrThrow(LABEL_VALUE);
    configs.get(isNamespace).forEach(n => addLabel(n, labelName, labelValue));
  }
  ```

1. Run the `label_namespace` function on the `example-configs` directory:

   ```sh
   export CONFIGS=../../example-configs

   kpt fn source $CONFIGS |
     node dist/label_namespace_run.js -d label_name=color -d label_value=orange |
     kpt fn sink $CONFIGS
   ```

   As the name suggests, this function added the given label to all
   `Namespace` objects in the `example-configs` directory:

   ```sh
   git diff $CONFIGS
   ```

1. Try modifying the function in `src/label_namespace.ts` to perform other
   operations and rerun the function on `example-configs` to see the changes.

1. Explore validation functions like [validate-rolebinding]. Instead of
   transforming configuration, this function searches RoleBindings and returns
   a results field containing details about invalid subject names.

  ```typescript
  import { Configs, kubernetesObjectResult } from 'kpt-functions';
  import { isRoleBinding } from './gen/io.k8s.api.rbac.v1';

  const SUBJECT_NAME = 'subject_name';

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

   ```sh
   kpt fn source $CONFIGS |
     node dist/validate_rolebinding_run.js -d subject_name=alice@foo-corp.com |
     kpt fn sink $CONFIGS
   git diff $CONFIGS
   ```

1. Explore generator functions like [expand-team-cr]. This function generates
   a per-environment Namespace and RoleBinding object for each custom resource
   (CR) of type Team.

  ```typescript
  import { Configs } from 'kpt-functions';
  import { isTeam, Team } from './gen/dev.cft.anthos.v1alpha1';
  import { Namespace } from './gen/io.k8s.api.core.v1';
  import { RoleBinding, Subject } from './gen/io.k8s.api.rbac.v1';

  const ENVIRONMENTS = ['dev', 'prod'];

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
          kind: 'ClusterRole',
          name: item.role,
          apiGroup: 'rbac.authorization.k8s.io',
        },
      });
    });
  }

  function roleSubjects(item: Team.Spec.Item): Subject[] {
    const userSubjects: Subject[] = (item.users || []).map(
      (user) =>
        new Subject({
          kind: 'User',
          name: user,
        })
    );
    const groupSubjects: Subject[] = (item.groups || []).map(
      (group) =>
        new Subject({
          kind: 'Group',
          name: group,
        })
    );
    return userSubjects.concat(groupSubjects);
  }
  ```

1. Run `expand-team-cr` on `example-configs`.

   ```sh
   kpt fn source $CONFIGS |
     node dist/expand_team_cr_run.js |
     kpt fn sink $CONFIGS
   git diff $CONFIGS
   ```

## Next Steps

- Take a look at these [demo functions] to better understand
  how to use the typescript SDK.
- Read the complete [Typescript Developer Guide].
- Learn how to [run functions].

[TS SDK API]: https://googlecontainertools.github.io/kpt-functions-sdk/api/
[download-node]: https://nodejs.org/en/download/
[install-docker]: https://docs.docker.com/engine/installation/
[download-kpt]: ../../../../../installation/
[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/label_namespace.ts
[validate-rolebinding]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/validate_rolebinding.ts
[expand-team-cr]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/demo-functions/src/expand_team_cr.ts
[demo functions]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/
[Typescript Developer Guide]: ../develop/
[run functions]: ../../../../consumer/function/
