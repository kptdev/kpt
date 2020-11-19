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

package cmdsearch

var validateCases = []test{
	{
		name: "invalid namespace",
		args: []string{"--by-path", "metadata.namespace", "--put-literal", "1", "--validate"},
		input: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.namespace: 1
validation errors
${filePath}:  metadata.namespace: Invalid type. Expected: [string,null], given: integer
`,
		expectedResources: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
 `,
	},
	{
		name: "valid containerPort",
		args: []string{"--by-value", "foo", "--put-literal", "bar", "--validate"},
		input: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: foo
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.template.spec.containers[0].ports[0].containerPort: bar
validation errors
${filePath}:  spec.template.spec.containers.0.ports.0.containerPort: Invalid type. Expected: integer, given: string
`,
		expectedResources: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: foo
 `,
	},
	{
		name: "valid replace by value",
		args: []string{"--by-value", "nginx", "--put-literal", "ubuntu", "--validate"},
		input: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
 `,
		out: `${baseDir}/
matched 5 field(s)
${filePath}:  spec.selector.app: ubuntu
${filePath}:  spec.template.metadata.name: ubuntu
${filePath}:  spec.template.metadata.labels.app: ubuntu
${filePath}:  spec.template.spec.containers[0].name: ubuntu
${filePath}:  spec.template.spec.containers[0].image: ubuntu
`,
		expectedResources: `
apiVersion: v1
kind: ReplicationController
metadata:
  name: "bob"
spec:
  replicas: 2
  selector:
    app: ubuntu
  template:
    metadata:
      name: ubuntu
      labels:
        app: ubuntu
    spec:
      containers:
      - name: ubuntu
        image: ubuntu
        ports:
        - containerPort: 80
 `,
	},
}
