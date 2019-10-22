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

package command_test

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/command"
)

func TestCommandBuilder_Build(t *testing.T) {
	dir := copyTestData(t, ".")
	defer os.RemoveAll(dir)
	root := &cobra.Command{
		Use: "test-kpt",
	}
	err := command.CommandBuilder{
		PkgPath: dir,
		RootCmd: root,
	}.BuildCommands()
	if !assert.NoError(t, err) {
		return
	}
	root.SetArgs([]string{"set", "image", "nginx",
		"--new-image", "nginx:1.8.1",
		"--old-image", "nginx:1.7.9"})
	err = root.Execute()
	if !assert.NoError(t, err) {
		return
	}
	b, err := ioutil.ReadFile(filepath.Join(dir, "resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
  annotations:
    name: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.8.1
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: foo
  labels:
    app: foo
  annotations:
    name: foo
spec:
  serviceName: foo-service
  replicas: 3
  selector:
    matchLabels:
      app: foo
  template:
    metadata:
      labels:
        app: foo
    spec:
      containers:
      - name: foo
        image: foo:v1.0.0
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: foo-service
spec:
  selector:
    app: foo
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
`, string(b))
}

func TestCommandBuilder_Build_noMatch(t *testing.T) {
	dir := copyTestData(t, ".")
	defer os.RemoveAll(dir)
	root := &cobra.Command{
		Use: "test-kpt",
	}
	err := command.CommandBuilder{
		PkgPath: dir,
		RootCmd: root,
	}.BuildCommands()
	if !assert.NoError(t, err) {
		return
	}
	root.SetArgs([]string{"set", "image", "nginx",
		"--new-image", "nginx:1.8.1",
		"--old-image", "nginx:1.7.1"})
	err = root.Execute()
	if !assert.NoError(t, err) {
		return
	}
	b, err := ioutil.ReadFile(filepath.Join(dir, "resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, noChange, string(b))
}

var noChange = `# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
  annotations:
    name: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: foo
  labels:
    app: foo
  annotations:
    name: foo
spec:
  serviceName: foo-service
  replicas: 3
  selector:
    matchLabels:
      app: foo
  template:
    metadata:
      labels:
        app: foo
    spec:
      containers:
      - name: foo
        image: foo:v1.0.0
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: foo-service
spec:
  selector:
    app: foo
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
`

func copyTestData(t *testing.T, dir string) string {
	to, err := ioutil.TempDir("", "kpt-test-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	from := testDataDir(t)
	if !assert.NoError(t, os.Chdir(from)) {
		t.FailNow()
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(to, path), 0700)
		}

		sourcefile, err := os.Open(filepath.Join(from, path))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer sourcefile.Close()

		destfile, err := os.OpenFile(
			filepath.Join(to, path), os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer destfile.Close()

		_, err = io.Copy(destfile, sourcefile)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		return nil
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return to
}

func testDataDir(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(1)
	if !assert.True(t, ok, "could not get testdata directory") {
		t.FailNow()
	}
	ds, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "testdata"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return ds
}
