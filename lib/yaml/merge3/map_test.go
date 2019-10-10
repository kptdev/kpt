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

package merge3_test

var mapTestCases = []testCase{
	//
	// Test Case
	//
	{`Add the annotations map field`,
		`
kind: Deployment`,
		`
kind: Deployment
metadata:
  annotations:
    d: e # add these annotations
`,
		`
kind: Deployment`,
		`
kind: Deployment
metadata:
  annotations:
    d: e # add these annotations`, nil},

	//
	// Test Case
	//
	{`Add an annotation to the field`,
		`
kind: Deployment
metadata:
  annotations:
    a: b`,
		`
kind: Deployment
metadata:
  annotations:
    a: b
    d: e  # add these annotations`,
		`
kind: Deployment
metadata:
  annotations:
    g: h  # keep these annotations`,
		`
kind: Deployment
metadata:
  annotations:
    g: h # keep these annotations
    d: e # add these annotations`, nil},

	//
	// Test Case
	//
	{`Add an annotation to the field, field missing from dest`,
		`
kind: Deployment
metadata:
  annotations:
    a: b # ignored because unchanged`,
		`
kind: Deployment
metadata:
  annotations:
    a: b # ignore because unchanged
    d: e`,
		`
kind: Deployment`,
		`
kind: Deployment
metadata:
  annotations:
    d: e`, nil},

	//
	// Test Case
	//
	{`Update an annotation on the field, field messing rom the dest`,
		`
kind: Deployment
metadata:
  annotations:
    a: b
    d: c`,
		`
kind: Deployment
metadata:
  annotations:
    a: b
    d: e  # set these annotations`,
		`
kind: Deployment
metadata:
  annotations:
    g: h  # keep these annotations`,
		`
kind: Deployment
metadata:
  annotations:
    g: h # keep these annotations
    d: e # set these annotations`, nil},

	//
	// Test Case
	//
	{`Add an annotation to the field, field missing from dest`,
		`
kind: Deployment
metadata:
  annotations:
    a: b # ignored because unchanged`,
		`
kind: Deployment
metadata:
  annotations:
    a: b # ignore because unchanged
    d: e`,
		`
kind: Deployment`,
		`
kind: Deployment
metadata:
  annotations:
    d: e`, nil},

	//
	// Test Case
	//
	{`Remove an annotation`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations: {}`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    c: d
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    c: d`, nil},

	//
	// Test Case
	//
	// TODO(#36) support ~annotations~: {} deletion
	{`Specify a field as empty that isn't present in the source`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations: null`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo`, nil},

	//
	// Test Case
	//
	{`Remove an annotation`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    c: d
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    c: d`, nil},

	//
	// Test Case
	//
	{`Remove annotations field`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`, nil},

	//
	// Test Case
	//
	{`Remove annotations field, but keep in dest`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    foo: bar # keep this annotation even though the parent field was removed`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    foo: bar # keep this annotation even though the parent field was removed`, nil},

	//
	// Test Case
	//
	{`Remove annotations, but they are already empty`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a: b`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations: {}`,
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`, nil},
}
