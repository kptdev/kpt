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

var listTestCases = []testCase{
	// List Field Test Cases

	//
	// Test Case
	//
	{`Replace list`,
		`
list:
- 1
- 2
- 3`,
		`
list:
- 2
- 3
- 4`,
		`
list:
- 1
- 2
- 3`,
		`
list:
- 2
- 3
- 4`, nil},

	//
	// Test Case
	//
	{`Add an updated list`,
		`
apiVersion: apps/v1
list: # old value
- 1
- 2
- 3
`,
		`
apiVersion: apps/v1
list: # new value
- 2
- 3
- 4
`,
		`
apiVersion: apps/v1`,
		`
apiVersion: apps/v1
list:
- 2
- 3
- 4
`, nil},

	//
	// Test Case
	//
	{`Add keep an omitted field`,
		`
apiVersion: apps/v1
kind: Deployment`,
		`
apiVersion: apps/v1
kind: StatefulSet`,
		`
apiVersion: apps/v1
list: # not present in sources
- 2
- 3
- 4
`,
		`
apiVersion: apps/v1
list: # not present in sources
  - 2
  - 3
  - 4
kind: StatefulSet
`, nil},

	//
	// Test Case
	//
	// TODO(#36): consider making this an error
	{`Change an updated field`,
		`
apiVersion: apps/v1
list: # old value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1
list: # new value
- 2
- 3
- 4`,
		`
apiVersion: apps/v1
list: # conflicting value
- a
- b
- c`,
		`
apiVersion: apps/v1
list: # conflicting value
  - 2
  - 3
  - 4
`, nil},

	//
	// Test Case
	//
	{`Ignore a field -- set`,
		`
apiVersion: apps/v1
list: # ignore value
- 1
- 2
- 3
`,
		`
apiVersion: apps/v1
list: # ignore value
- 1
- 2
- 3`, `
apiVersion: apps/v1
list:
- 2
- 3
- 4
`, `
apiVersion: apps/v1
list:
- 2
- 3
- 4
`, nil},

	//
	// Test Case
	//
	{`Ignore a field -- empty`,
		`
apiVersion: apps/v1
list: # ignore value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1
list: # ignore value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1
`,
		`
apiVersion: apps/v1
`, nil},

	//
	// Test Case
	//
	{`Explicitly clear a field`,
		`
apiVersion: apps/v1`,
		`
apiVersion: apps/v1
list: null # clear`,
		`
apiVersion: apps/v1
list: # value to clear
- 1
- 2
- 3`,
		`
apiVersion: apps/v1`, nil},

	//
	// Test Case
	//
	{`Implicitly clear a field`,
		`
apiVersion: apps/v1
list: # clear value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1`,
		`
apiVersion: apps/v1
list: # old value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1`, nil},

	//
	// Test Case
	//
	// TODO(#36): consider making this an error
	{`Implicitly clear a changed field`,
		`
apiVersion: apps/v1
list: # old value
- 1
- 2
- 3`,
		`
apiVersion: apps/v1`,
		`
apiVersion: apps/v1
list: # old value
- a
- b
- c`,
		`
apiVersion: apps/v1`, nil},
}
