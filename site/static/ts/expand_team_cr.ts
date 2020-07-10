/**
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

expandTeamCr.usage = `
Generates per-environment Namespaces and RoleBindings from the 'Team' custom resource.

Configured using a custom resource of kind Team, e.g.:

apiVersion: anthos.cft.dev/v1alpha1
kind: Team
metadata:
  name: payments
spec:
  roles:
  - role: sre
    users:
    - jane@clearify.co
  - groups:
    - payments-developers@clearify.co
    role: developer
    users:
    - basic@clearify.co

This configuration creates 2 Namespaces (payments-prod, payments-dev)
and corresponding Rolebinding objects in each of these Namespaces.
`;
