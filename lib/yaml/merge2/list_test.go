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

var listTestCases = []testCase{
	{`replace List -- different value in dest`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
items:
- 0
- 1
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
	},

	{`replace List -- missing from dest`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
	},

	//
	// Test Case
	//
	{`keep List -- same value in src and dest`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
	},

	//
	// Test Case
	//
	{`keep List -- unspecified in src`,
		`
kind: Deployment
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
	},

	//
	// Test Case
	//
	{`remove List -- null in src`,
		`
kind: Deployment
items: null
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
`,
	},

	//
	// Test Case
	//
	{`remove list -- empty in src`,
		`
kind: Deployment
items: {}
`,
		`
kind: Deployment
items:
- 1
- 2
- 3
`,
		`
kind: Deployment
items: {}
`,
	},
}
