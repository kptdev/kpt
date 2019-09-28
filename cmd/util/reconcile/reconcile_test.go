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

package reconcile

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/testutil"
	"lib.kpt.dev/yaml"
)

func TestCmd_init(t *testing.T) {
	instance := Cmd{}
	instance.init()
	api, err := yaml.Parse(`apiVersion: apps/v1
kind: 
`)
	if !assert.NoError(t, err) {
		return
	}
	filter := instance.filterProvider("example.com:version", api)
	assert.Equal(t, &filters.ContainerFilter{Image: "example.com:version", Config: api}, filter)
}

func TestCmd_Execute(t *testing.T) {
	g, _, clean := testutil.SetupDefaultRepoAndWorkspace(t)
	defer clean()
	if !assert.NoError(t, os.Chdir(filepath.Dir(g.RepoDirectory))) {
		return
	}
	c := filepath.Base(g.RepoDirectory)
	if !assert.NoError(t, os.Chdir(filepath.Dir(g.RepoDirectory))) {
		return
	}

	// write a test filter
	f := `apiVersion: gcr.io/example.com/image:version
kind: ValueReplacer
stringMatch: Deployment
replace: StatefulSet
`
	if !assert.NoError(t, ioutil.WriteFile(
		filepath.Join(g.RepoDirectory, "filter.yaml"), []byte(f), 0600)) {
		return
	}

	instance := Cmd{
		PkgPath: c,
		filterProvider: func(s string, node *yaml.RNode) kio.Filter {
			// parse the filter from the input
			filter := yaml.YFilter{}
			b := &bytes.Buffer{}
			e := yaml.NewEncoder(b)
			if !assert.NoError(t, e.Encode(node.YNode())) {
				t.FailNow()
			}
			e.Close()
			d := yaml.NewDecoder(b)
			if !assert.NoError(t, d.Decode(&filter)) {
				t.FailNow()
			}

			return filters.Modifier{
				Filters: []yaml.YFilter{{Filter: yaml.Lookup("kind")}, filter},
			}
		},
	}
	err := instance.Execute()
	if !assert.NoError(t, err) {
		return
	}
	b, err := ioutil.ReadFile(
		filepath.Join(g.RepoDirectory, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, string(b), "kind: StatefulSet")
}

func Test_getContainerName(t *testing.T) {
	// make sure gcr.io works
	n, err := yaml.Parse(`apiVersion: gcr.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c := getContainerName(n)
	assert.Equal(t, "gcr.io/foo/bar:something", c)

	// make sure regional gcr.io works
	n, err = yaml.Parse(`apiVersion: us.gcr.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c = getContainerName(n)
	assert.Equal(t, "us.gcr.io/foo/bar:something", c)

	// container from annotation
	n, err = yaml.Parse(`apiVersion: v1
kind: MyThing
metadata:
  annotations:
    kpt.dev/container: gcr.io/foo/bar:something
`)
	if !assert.NoError(t, err) {
		return
	}
	c = getContainerName(n)
	assert.Equal(t, "gcr.io/foo/bar:something", c)

	// doesn't have a container
	n, err = yaml.Parse(`apiVersion: v1
kind: MyThing
metadata:
`)
	if !assert.NoError(t, err) {
		return
	}
	c = getContainerName(n)
	assert.Equal(t, "", c)

	// make sure docker.io works
	n, err = yaml.Parse(`apiVersion: docker.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c = getContainerName(n)
	assert.Equal(t, "docker.io/foo/bar:something", c)
}
