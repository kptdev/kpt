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

import { Configs, kubernetesObjectResult } from 'kpt-functions';
import { isRoleBinding } from './gen/io.k8s.api.rbac.v1';

const SUBJECT_NAME = 'subject_name';

export async function validateRolebinding(configs: Configs) {
  // Get the subject name parameter.
  const subjectName = configs.getFunctionConfigValueOrThrow(SUBJECT_NAME);

  // Iterate over all RoleBinding objects in the input, and filter those that have a subject
  // we're looking for.
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

validateRolebinding.usage = `
Disallows RBAC RoleBinding objects with the given subject name.

Configured using a ConfigMap with the following key:

${SUBJECT_NAME}: RoleBinding subjects.name to disallow.

Example:

apiVersion: v1
kind: ConfigMap
data:
  ${SUBJECT_NAME}: alice
metadata:
  name: my-config
`;
