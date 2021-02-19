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
		args: []string{"--by-value", "3", "--put-comment", "kpt-set: ${replicas}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/${filePath}
fieldPath: spec.replicas
value: 3 # kpt-set: ${replicas}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # kpt-set: ${replicas}
 `,
	},
	{
		name: "put pattern group of setters",
		args: []string{"--by-value", "nginx-deployment", "--put-comment", "kpt-set: ${image}-${kind}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: nginx-deployment # kpt-set: ${image}-${kind}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment # kpt-set: ${image}-${kind}
spec:
  replicas: 3
 `,
	},
	{
		name: "put pattern by value",
		args: []string{"--by-value", "dev/my-project/nginx", "--put-comment", "kpt-set: ${env}/${project}/${name}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev/my-project/nginx
spec:
  replicas: 3
 `,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: dev/my-project/nginx # kpt-set: ${env}/${project}/${name}

Mutated 1 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev/my-project/nginx # kpt-set: ${env}/${project}/${name}
spec:
  replicas: 3
 `,
	},
	{
		name: "put comment by capture groups simple case",
		args: []string{"--by-value-regex", "my-project-(.*)", "--put-comment", "kpt-set: ${project}-${1}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-project-deployment
  namespace: my-project-namespace
 `,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: my-project-deployment # kpt-set: ${project}-deployment

${baseDir}/${filePath}
fieldPath: metadata.namespace
value: my-project-namespace # kpt-set: ${project}-namespace

Mutated 2 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-project-deployment # kpt-set: ${project}-deployment
  namespace: my-project-namespace # kpt-set: ${project}-namespace
 `,
	},
	{
		name: "put value and comment by regex capture groups",
		args: []string{"--by-value-regex", `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
			"--put-value", "${1}-prod-${2}-us-central-1-${3}", "--put-comment", "kpt-set: ${1}-${environment}-${2}-${region}-${3}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1-dev-bar1-us-east-1-baz1
  namespace: foo2-dev-bar2-us-east-1-baz2
 `,
		out: `${baseDir}/${filePath}
fieldPath: metadata.name
value: foo1-prod-bar1-us-central-1-baz1 # kpt-set: foo1-${environment}-bar1-${region}-baz1

${baseDir}/${filePath}
fieldPath: metadata.namespace
value: foo2-prod-bar2-us-central-1-baz2 # kpt-set: foo2-${environment}-bar2-${region}-baz2

Mutated 2 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1-prod-bar1-us-central-1-baz1 # kpt-set: foo1-${environment}-bar1-${region}-baz1
  namespace: foo2-prod-bar2-us-central-1-baz2 # kpt-set: foo2-${environment}-bar2-${region}-baz2
 `,
	},
	{
		name: "put value and comment by regex capture groups error",
		args: []string{"--by-value-regex", `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
			"--put-value", "${1}-prod-${2}-us-central-1-${3}", "--put-comment", "kpt-set: ${1}-${environment}-${2}-${region}-${3}-extra-${4}"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1-dev-bar1-us-east-1-baz1
  namespace: foo2-dev-bar2-us-east-1-baz2
 `,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1-dev-bar1-us-east-1-baz1
  namespace: foo2-dev-bar2-us-east-1-baz2
 `,
		errMsg: "unable to resolve capture groups",
	},
}
