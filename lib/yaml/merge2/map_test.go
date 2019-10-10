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

package merge2_test

var mapTestCases = []testCase{
	{`merge Map -- update field in dest`,
		`
kind: Deployment
spec:
  foo: bar1
`,
		`
kind: Deployment
spec:
  foo: bar0
  baz: buz
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	{`merge Map -- add field to dest`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
spec:
  foo: bar0
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	{`merge Map -- add list, empty in dest`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
spec: {}
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	{`merge Map -- add list, missing from dest`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	{`merge Map -- add Map first`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
spec:
  foo: bar1
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	{`merge Map -- add Map second`,
		`
kind: Deployment
spec:
  baz: buz
  foo: bar1
`,
		`
kind: Deployment
spec:
  foo: bar1
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	//
	// Test Case
	//
	{`keep map -- map missing from src`,
		`
kind: Deployment
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	//
	// Test Case
	//
	{`keep map -- empty list in src`,
		`
kind: Deployment
items: {}
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
	},

	//
	// Test Case
	//
	{`remove Map -- null in src`,
		`
kind: Deployment
spec: null
`,
		`
kind: Deployment
spec:
  foo: bar1
  baz: buz
`,
		`
kind: Deployment
`,
	},
}
