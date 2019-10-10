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

var elementTestCases = []testCase{
	{`merge Element -- keep field in dest`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v0
  command: ['run.sh']
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command:
  - run.sh
`,
	},

	{`merge Element -- add field to dest`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command: ['run.sh']
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v0
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command:
  - run.sh
`,
	},

	{`merge Element -- add list, empty in dest`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command: ['run.sh']
`,
		`
kind: Deployment
items: {}
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command:
  - run.sh
`,
	},

	{`merge Element -- add list, missing from dest`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command: ['run.sh']
`,
		`
kind: Deployment
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
  command:
  - run.sh
`,
	},

	{`merge Element -- add Element first`,
		`
kind: Deployment
items:
- name: bar
  image: bar:v1
  command: ['run2.sh']
- name: foo
  image: foo:v1
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v0
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
	},

	{`merge Element -- add Element second`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command: ['run2.sh']
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v0
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
	},

	//
	// Test Case
	//
	{`keep list -- list missing from src`,
		`
kind: Deployment
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command: ['run2.sh']
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
	},

	//
	// Test Case
	//
	{`keep Element -- element missing in src`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v0
- name: bar
  image: bar:v1
  command: ['run2.sh']
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
	},

	//
	// Test Case
	//
	{`keep element -- empty list in src`,
		`
kind: Deployment
items: {}
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
	},

	//
	// Test Case
	//
	{`remove Element -- null in src`,
		`
kind: Deployment
items: null
`,
		`
kind: Deployment
items:
- name: foo
  image: foo:v1
- name: bar
  image: bar:v1
  command:
  - run2.sh
`,
		`
kind: Deployment
`,
	},
}
