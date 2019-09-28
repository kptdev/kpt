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

package merge_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/fmtr"
	"lib.kpt.dev/yaml"
	. "lib.kpt.dev/yaml/walk/merge"
)

const dest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h1
    i: j
    m: n2
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args: ['c', 'a', 'b']
        env:
        - name: DEMO_GREETING
          value: "Hello from the environment"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
`

func TestMerge_map(t *testing.T) {
	dest := yaml.MustParse(dest)
	src := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h2
    k: l
    m: n1
`)

	result, err := Merge(src, dest)
	if !assert.NoError(t, err) {
		return
	}
	actual, err := result.String()
	if !assert.NoError(t, err) {
		return
	}

	expected := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h2
    i: j
    k: l
    m: n1
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args:
        - c
        - a
        - b
        env:
        - name: DEMO_GREETING
          value: "Hello from the environment"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
`
	b, err := fmtr.FormatInput(bytes.NewBufferString(expected))
	if !assert.NoError(t, err) {
		return
	}
	expected = b.String()

	b, err = fmtr.FormatInput(bytes.NewBufferString(actual))
	if !assert.NoError(t, err) {
		return
	}
	actual = b.String()

	assert.Equal(t, expected, actual)
}

func TestMerge_clear(t *testing.T) {
	dest := yaml.MustParse(dest)
	src := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations: null
`)

	result, err := Merge(src, dest)
	if !assert.NoError(t, err) {
		return
	}
	actual, err := result.String()
	if !assert.NoError(t, err) {
		return
	}

	expected := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args:
        - c
        - a
        - b
        env:
        - name: DEMO_GREETING
          value: "Hello from the environment"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
`
	b, err := fmtr.FormatInput(bytes.NewBufferString(expected))
	if !assert.NoError(t, err) {
		return
	}
	expected = b.String()

	b, err = fmtr.FormatInput(bytes.NewBufferString(actual))
	if !assert.NoError(t, err) {
		return
	}
	actual = b.String()

	assert.Equal(t, expected, actual)
}

func TestMerge_mapInverse(t *testing.T) {
	dest := yaml.MustParse(dest)
	src := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h2
    k: l
    m: n1
`)

	result, err := Merge(dest, src)
	if !assert.NoError(t, err) {
		return
	}
	actual, err := result.String()
	if !assert.NoError(t, err) {
		return
	}

	expected := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h1
    i: j
    k: l
    m: n2
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args:
        - c
        - a
        - b
        env:
        - name: DEMO_GREETING
          value: "Hello from the environment"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
`
	b, err := fmtr.FormatInput(bytes.NewBufferString(expected))
	if !assert.NoError(t, err) {
		return
	}
	expected = b.String()

	b, err = fmtr.FormatInput(bytes.NewBufferString(actual))
	if !assert.NoError(t, err) {
		return
	}
	actual = b.String()

	assert.Equal(t, expected, actual)
}

func TestMerge_listElem(t *testing.T) {
	dest := yaml.MustParse(dest)
	src := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: nginx
        env:
        - name: DEMO_GREETING
          value: "New Demo Greeting"
        - name: NEW_DEMO_VALUE
          value: "Another Env Not In The Dest"
`)

	result, err := Merge(src, dest)
	if !assert.NoError(t, err) {
		return
	}
	actual, err := result.String()
	if !assert.NoError(t, err) {
		return
	}

	expected := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h1
    i: j
    m: n2
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args:
        - c
        - a
        - b
        env:
        - name: DEMO_GREETING
          value: "New Demo Greeting"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
        - name: NEW_DEMO_VALUE
          value: "Another Env Not In The Dest"
`

	b, err := fmtr.FormatInput(bytes.NewBufferString(expected))
	if !assert.NoError(t, err) {
		return
	}
	expected = b.String()

	b, err = fmtr.FormatInput(bytes.NewBufferString(actual))
	if !assert.NoError(t, err) {
		return
	}
	actual = b.String()

	assert.Equal(t, expected, actual)
}

func TestMerge_list(t *testing.T) {
	dest := yaml.MustParse(dest)
	src := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: nginx
        args: ['e', 'd', 'f']
`)

	result, err := Merge(src, dest)
	if !assert.NoError(t, err) {
		return
	}
	actual, err := result.String()
	if !assert.NoError(t, err) {
		return
	}

	expected := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app: java
  annotations:
    a.b.c: d.e.f
    g: h1
    i: j
    m: n2
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        args:
        - e
        - d
        - f
        env:
        - name: DEMO_GREETING
          value: "Hello from the environment"
        - name: DEMO_FAREWELL
          value: "Such a sweet sorrow"
`

	b, err := fmtr.FormatInput(bytes.NewBufferString(expected))
	if !assert.NoError(t, err) {
		return
	}
	expected = b.String()

	b, err = fmtr.FormatInput(bytes.NewBufferString(actual))
	if !assert.NoError(t, err) {
		return
	}
	actual = b.String()

	assert.Equal(t, expected, actual)
}

func TestMerge_commentsKept(t *testing.T) {
	actual, err := MergeStrings(`
a:
  b:
    c: e
`,
		`
a:
  b:
    # header comment
    c: d
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    # header comment
    c: e
`, actual)

	actual, err = MergeStrings(`
a:
  b:
    c: e
`,
		`
a:
  b:
    c: d
    # footer comment
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    c: e
    # footer comment
`, actual)

	actual, err = MergeStrings(`
a:
  b:
    c: e
`,
		`
a:
  b:
    c: d # line comment
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    c: e # line comment
`, actual)
}

func TestMerge_commentsOverride(t *testing.T) {
	actual, err := MergeStrings(`
a:
  b:
    # header comment
    c: e
`,
		`
a:
  b:
    # replace comment
    c: d
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    # header comment
    c: e
`, actual)

	actual, err = MergeStrings(`
a:
  b:
    c: e
    # footer comment
`,
		`
a:
  b:
    c: d
    # replace comment
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    c: e
    # footer comment
`, actual)

	actual, err = MergeStrings(`
a:
  b:
    c: e # line comment
`,
		`
a:
  b:
    c: d # replace comment
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    c: e # line comment
`, actual)

	actual, err = MergeStrings(`
a:
  b:
    c: d # line comment
`,
		`
a:
  b:
    c: d # replace comment
`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a:
  b:
    c: d # line comment
`, actual)
}
