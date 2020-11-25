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

var putPatternCases = []test{
	{
		name: "put pattern single setter",
		args: []string{"--by-value", "3", "--put-pattern", "${replicas}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"`,
		out: `${baseDir}/${filePath}
fieldPath: spec.replicas
value: 3 # {"$kpt-set":"${replicas}"}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
	},
	{
		name: "put pattern group of setters",
		args: []string{"--by-value", "nginx-deployment", "--put-pattern", "${image}-${kind}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: image
          value: "nginx"
    io.k8s.cli.setters.kind:
      x-k8s-cli:
        setter:
          name: kind
          value: "deployment"`,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: nginx-deployment # {"$kpt-set":"${image}-${kind}"}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment # {"$kpt-set":"${image}-${kind}"}
spec:
  replicas: 3
 `,
	},
	{
		name: "put pattern by regex",
		args: []string{"--by-value-regex", "my-project-*", "--put-pattern", "${project}-*"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-project-deployment
  namespace: my-project-namespace
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.project:
      x-k8s-cli:
        setter:
          name: project
          value: "my-project"`,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: my-project-deployment # {"$kpt-set":"${project}-deployment"}

${baseDir}/${filePath}
fieldPath: metadata.namespace
value: my-project-namespace # {"$kpt-set":"${project}-namespace"}

Mutated 2 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-project-deployment # {"$kpt-set":"${project}-deployment"}
  namespace: my-project-namespace # {"$kpt-set":"${project}-namespace"}
spec:
  replicas: 3
 `,
	},
	{
		name: "put pattern by value",
		args: []string{"--by-value", "dev/my-project/nginx", "--put-pattern", "${env}/${project}/${name}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev/my-project/nginx
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.project:
      x-k8s-cli:
        setter:
          name: project
          value: "my-project"
    io.k8s.cli.setters.env:
      x-k8s-cli:
        setter:
          name: env
          value: "dev"
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: "nginx"
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: "my-space"`,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: dev/my-project/nginx # {"$kpt-set":"${env}/${project}/${name}"}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev/my-project/nginx # {"$kpt-set":"${env}/${project}/${name}"}
spec:
  replicas: 3
 `,
	},
	{
		name: "put pattern error",
		args: []string{"--by-value", "nginx-deployment", "--put-pattern", "${image}-${tag}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: image
          value: "nginx"
    io.k8s.cli.setters.kind:
      x-k8s-cli:
        setter:
          name: kind
          value: "deployment"`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		errMsg: `setter "tag" doesn't exist, please create setter definition and try again`,
	},
	{
		name: "put pattern list-values error",
		args: []string{"--by-value", "3", "--put-pattern", "${replicas-list}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		inputKptfile: `apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas-list:
      x-k8s-cli:
        setter:
          name: replicas-list
          value: ""
          listValues: 
           - "1"
           - "2"`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		errMsg: `setter pattern should not refer to array type setters: "replicas-list"`,
	},
}
