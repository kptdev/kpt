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

var scalarTestCases = []testCase{
	{`replace scalar -- different value in dest`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
field: value0
`,
		`
kind: Deployment
field: value1
`,
	},

	{`replace scalar -- missing from dest`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
`,
		`
kind: Deployment
field: value1
`,
	},

	//
	// Test Case
	//
	{`keep scalar -- same value in src and dest`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
field: value1
`,
	},

	//
	// Test Case
	//
	{`keep scalar -- unspecified in src`,
		`
kind: Deployment
`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
field: value1
`,
	},

	//
	// Test Case
	//
	{`remove scalar -- null in src`,
		`
kind: Deployment
field: null
`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
`,
	},

	//
	// Test Case
	//
	{`remove scalar -- empty in src`,
		`
kind: Deployment
field: {}
`,
		`
kind: Deployment
field: value1
`,
		`
kind: Deployment
field: {}
`,
	},

	//
	// Test Case
	//
	{`remove scalar -- null in src, missing in dest`,
		`
kind: Deployment
field: null
`,
		`
kind: Deployment
`,
		`
kind: Deployment
`,
	},

	//
	// Test Case
	//
	{`merge an empty value`,
		`
kind: Deployment
field: {}
`,
		`
kind: Deployment
`,
		`
kind: Deployment
`,
	},
}
