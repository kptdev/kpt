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

import { addLabel, Configs } from 'kpt-functions';
import { isNamespace } from './gen/io.k8s.api.core.v1';

export const LABEL_NAME = 'label_name';
export const LABEL_VALUE = 'label_value';

export async function labelNamespace(configs: Configs) {
  const labelName = configs.getFunctionConfigValueOrThrow(LABEL_NAME);
  const labelValue = configs.getFunctionConfigValueOrThrow(LABEL_VALUE);
  configs.get(isNamespace).forEach(n => addLabel(n, labelName, labelValue));
}

labelNamespace.usage = `
Adds a label to all Namespaces.

Configured using a ConfigMap with the following keys:

${LABEL_NAME}: Label name to add to Namespaces.
${LABEL_VALUE}: Label value to add to Namespaces.

Example:

To add a label 'color: orange' to Namespaces:

apiVersion: v1
kind: ConfigMap
data:
  ${LABEL_NAME}: color
  ${LABEL_VALUE}: orange
metadata:
  name: my-config
`;