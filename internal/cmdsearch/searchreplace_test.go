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

var searchReplaceCases = []test{
	{
		name: "search by value",
		args: []string{"--by-value", "3"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 3
${filePath}:  spec.foo: 3
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
	},
	{
		name: "search replace by value",
		args: []string{"--by-value", "3", "--put-literal", "4"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
foo:
  bar: 3
 `,
		out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 4
${filePath}:  foo.bar: 4
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
foo:
  bar: 4
 `,
	},
	{
		name: "search replace multiple deployments",
		args: []string{"--by-value", "3", "--put-literal", "4"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 4
${filePath}:  spec.replicas: 4
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 4
 `,
	},
	{
		name: "search replace multiple deployments different value",
		args: []string{"--by-value", "3", "--put-literal", "4"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 5
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 4
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 5
 `,
	},
	{
		name: "search by regex",
		args: []string{"--by-value-regex", "nginx-*"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.name: nginx-deployment
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
	},
	{
		name: "search replace by regex",
		args: []string{"--by-value-regex", "nginx-*", "--put-literal", "ubuntu-deployment"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.name: ubuntu-deployment
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ubuntu-deployment
spec:
  replicas: 3
 `,
	},
	{
		name: "search by path",
		args: []string{"--by-path", "spec.replicas"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 3
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
	},
	{
		name: "search by path expression 1",
		args: []string{"--by-path", "a.*.c"},
		input: `
a:
  b1:
    c: thing1 # MATCHES
    d: cat
  b2:
    c: thing2 # MATCHES
    d: dog
  b3:
    d:
    - f:
        c: thing3
        d: beep
    - f:
        g:
          c: thing4
          d: boop
    - d: mooo
 `,
		out: `${baseDir}/
matched 2 field(s)
${filePath}:  a.b1.c: thing1
${filePath}:  a.b2.c: thing2
`,
		expectedResources: `
a:
  b1:
    c: thing1 # MATCHES
    d: cat
  b2:
    c: thing2 # MATCHES
    d: dog
  b3:
    d:
    - f:
        c: thing3
        d: beep
    - f:
        g:
          c: thing4
          d: boop
    - d: mooo
 `,
	},
	{
		name: "search by path expression 2",
		args: []string{"--by-path", "a.**.c", "--put-literal", "new-thing"},
		input: `
a:
  b1:
    c: thing1 # MATCHES
    d: cat
  b2:
    c: thing2 # MATCHES
    d: dog
  b3:
    d:
    - f:
        c: thing3 # MATCHES
        d: beep
    - f:
        g:
          c: thing4 # MATCHES
          d: boop
    - d: mooo
 `,
		out: `${baseDir}/
matched 4 field(s)
${filePath}:  a.b1.c: new-thing
${filePath}:  a.b2.c: new-thing
${filePath}:  a.b3.d[0].f.c: new-thing
${filePath}:  a.b3.d[1].f.g.c: new-thing
`,
		expectedResources: `
a:
  b1:
    c: new-thing # MATCHES
    d: cat
  b2:
    c: new-thing # MATCHES
    d: dog
  b3:
    d:
    - f:
        c: new-thing # MATCHES
        d: beep
    - f:
        g:
          c: new-thing # MATCHES
          d: boop
    - d: mooo
 `,
	},
	{
		name: "search by array path",
		args: []string{"--by-path", "spec.foo[1]"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - b
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.foo[1]: b
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - b
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
	},
	{
		name: "search replace by array path regex",
		args: []string{"--by-path", "spec.foo[1]", "--put-literal", "c"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - b
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.foo[1]: c
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - c
 `,
	},
	{
		name: "search replace by array path out of bounds",
		args: []string{"--by-path", "spec.foo[2]", "--put-literal", "c"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - b
 `,
		out: `${baseDir}/
matched 0 field(s)
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - a
  - b
 `,
	},
	{
		name: "search replace by array objects path",
		args: []string{"--by-path", "spec.foo[1].c", "--put-literal", "thing-new"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - c: thing0
  - c: thing1
  - c: thing2
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.foo[1].c: thing-new
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo:
  - c: thing0
  - c: thing-new
  - c: thing2
 `,
	},
	{
		name: "replace by path and value",
		args: []string{"--by-path", "spec.replicas", "--by-value", "3", "--put-literal", "4"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 4
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
	},
	{
		name: "add non-existing field",
		args: []string{"--by-path", "metadata.namespace", "--put-literal", "myspace"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.namespace: myspace
`,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: myspace
spec:
  replicas: 3
 `,
	},
	{
		name: "put literal error",
		args: []string{"--put-literal", "something"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		errMsg: `at least one of ["by-value", "by-value-regex", "by-path"] must be provided`,
	},
	{
		name: "error when both by-value and by-regex provided",
		args: []string{"--by-value", "nginx-deployment", "--by-value-regex", "nginx-*"},
		input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		errMsg: `only one of ["by-value", "by-value-regex"] can be provided`,
	},
}
