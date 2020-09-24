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

package manifestreader

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func TestKyamlPathManifestReader_Read_resource(t *testing.T) {
	testCases := []struct {
		name                   string
		manifestName           string
		manifest               string
		readerNamespace        string
		readerEnforceNamespace bool
		expectError            bool
		expectedNamespace      string
		expectedAnnoCount      int
	}{
		{
			name:         "namespaced resource with namespace",
			manifestName: "deployment.yaml",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
`,
			readerNamespace:        "default",
			readerEnforceNamespace: false,
			expectedNamespace:      "default",
			expectedAnnoCount:      0,
		},
		{
			name:         "namespaced resource without namespace",
			manifestName: "my-dep.yaml",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    foo: bar
`,
			readerNamespace:        "bar",
			readerEnforceNamespace: false,
			expectedNamespace:      "bar",
			expectedAnnoCount:      1,
		},
		{
			name:         "clusterscoped resource",
			manifestName: "my-ns.yaml",
			manifest: `
apiVersion: v1
kind: Namespace
metadata:
  name: foo
  annotations:
    HuskerDu: New Day Rising
`,
			readerNamespace:        "bar",
			readerEnforceNamespace: true,
			expectedNamespace:      "",
			expectedAnnoCount:      1,
		},
		{
			name:         "enforce namespace",
			manifestName: "my-dep.yaml",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  namespace: default
`,
			readerNamespace:        "bar",
			readerEnforceNamespace: true,
			expectError:            true,
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("default")
			defer tf.Cleanup()

			d, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			err = ioutil.WriteFile(
				filepath.Join(d, test.manifestName), []byte(test.manifest), 0600)
			assert.NoError(t, err)
			err = ioutil.WriteFile(
				filepath.Join(d, kptfile.KptFileName), []byte(`Kptfile`), 0600)
			assert.NoError(t, err)

			infos, err := (&KyamlPathManifestReader{
				Path: d,
				ReaderOptions: manifestreader.ReaderOptions{
					Factory:          tf,
					Namespace:        test.readerNamespace,
					EnforceNamespace: test.readerEnforceNamespace,
				},
			}).Read()

			if test.expectError {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Equal(t, 1, len(infos))
			inf := infos[0]

			assert.Equal(t, test.expectedNamespace, inf.Namespace)

			u := inf.Object.(*unstructured.Unstructured)
			annos, found, err := unstructured.NestedStringMap(u.Object, "metadata", "annotations")
			assert.NoError(t, err)

			if !found {
				annos = make(map[string]string)
			}

			_, found = annos[kioutil.IndexAnnotation]
			assert.False(t, found)
			_, found = annos[kioutil.PathAnnotation]
			assert.False(t, found)
			assert.Equal(t, test.expectedAnnoCount, len(annos))
			assert.NotEmpty(t, inf.Name)
			assert.Equal(t, test.manifestName, inf.Source)
		})
	}
}

func TestKyamlPathManifestReader_Read_packages(t *testing.T) {
	//old := kyamlext.IgnoreFileName
	//defer func() { kyamlext.IgnoreFileName = old }()
	//kyamlext.IgnoreFileName = func() string {
	//	return kptfile.KptIgnoreFileName
	//}

	testCases := []struct {
		name          string
		path          string
		expectedCount int
		expectedError bool
	}{
		//TODO: Uncomment test when the testset is merged.
		//{
		//	name:          "ignorefiles should be checked",
		//	path:          "../../testutil/testdata/dataset-with-ignorefile",
		//	expectedCount: 1,
		//},
		{
			name:          "nested packages",
			path:          "../../testutil/testdata/dataset-with-autosetters/mysql",
			expectedCount: 3,
		},
		{
			name:          "no nesting",
			path:          "../../testutil/testdata/helloworld-fn",
			expectedCount: 2,
		},
		{
			name:          "no Kptfile should lead to error",
			path:          "../../testutil/testdata/helloworld-fn-no-kptfile",
			expectedError: true,
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("default")
			defer tf.Cleanup()

			infos, err := (&KyamlPathManifestReader{
				Path: test.path,
				ReaderOptions: manifestreader.ReaderOptions{
					Factory:   tf,
					Namespace: "default",
				},
			}).Read()

			if test.expectedError {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Equal(t, test.expectedCount, len(infos))
		})
	}
}
