// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testyaml contains test data and libraries for formatting
// Kubernetes configuration
package testyaml

var UnformattedYaml1 = []byte(`
spec: a
status:
  conditions:
  - 3
  - 1
  - 2
apiVersion: example.com/v1beta1
kind: MyType
`)

var UnformattedYaml2 = []byte(`
spec2: a
status2:
  conditions:
  - 3
  - 1
  - 2
apiVersion: example.com/v1beta1
kind: MyType2
`)

var UnformattedJson1 = []byte(`
{
  "spec": "a",
  "status": {"conditions": [3, 1, 2]},
  "apiVersion": "example.com/v1beta1",
  "kind": "MyType"
}
`)

var FormattedYaml1 = []byte(`apiVersion: example.com/v1beta1
kind: MyType
spec: a
status:
  conditions:
  - 3
  - 1
  - 2
`)

var FormattedYaml2 = []byte(`apiVersion: example.com/v1beta1
kind: MyType2
spec2: a
status2:
  conditions:
  - 3
  - 1
  - 2
`)
