package pipeline

import (
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFilterMetaData(t *testing.T) {
	tests := map[string]struct {
		resources    []string
		expected     []string
	}{
		"no resources": {
			resources: nil,
			expected: nil,
		},

		"nothing to filter": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},

		"filter out metadata": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: config.kpt.dev/v1
Kind: FunctionPermission
Metadata:
  Name: functionPermission
Spec:
  Allow:
  - imageName: gcr.io/my-project/*…..
  Permissions:
  - network
  - mount
  Disallow:
  - Name: gcr.io/my-project/*`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
				`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: mysql
setterDefinitions:
  replicas:
    description: "replica setter"
    type: integer
setterValues:
  replicas: 5`,
				`
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
sources:
  - "."`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},
	}

	for _, test := range tests {
		var nodes []*yaml.RNode

		for _, r := range test.resources {
			res, err := yaml.Parse(r)
			assert.NoError(t, err)
			nodes = append(nodes, res)
		}

		filteredRes := filterMetaData(nodes)
		if len(filteredRes) != len(test.expected) {
			t.Fatal("length of filtered resources not equal to expected")
		}

		for i, r := range filteredRes {
			res, err := r.String()
			assert.NoError(t, err)
			assert.Equal(t, test.expected[i], res)
		}
	}
}

func TestResolveSources(t *testing.T) {
	// empty directory
	emptyDir, err := ioutil.TempDir("", "kpt")
	defer testutil.AssertNoError(t, os.RemoveAll(emptyDir))
	testutil.AssertNoError(t, err)

	// package with subpackages
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(dir)

	err = ioutil.WriteFile(dir+"/Kptfile", []byte(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
`), 0600)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// first subdirectory of dir
	subDir1, err := ioutil.TempDir(dir, "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(subDir1)

	err = ioutil.WriteFile(subDir1+"/Kptfile", []byte(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
`), 0600)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// second subdirectory of dir
	subDir2, err := ioutil.TempDir(dir, "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(subDir2)

	err = ioutil.WriteFile(subDir2+"/Kptfile", []byte(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
`), 0600)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// sorting for testing purposes
	subDirs := []string{subDir1, subDir2}
	sort.Strings(subDirs)

	tests := map[string]struct {
		p			*pkg
		expected 	[]string
	}{
		"package without pipeline": {
			p: &pkg{
				path: emptyDir,
				pipeline: nil,
			},
			// pipeline sources defaults to "./*"
			expected: []string{emptyDir},
		},

		"empty directory, current directory only": {
			p: &pkg{
				path: emptyDir,
				pipeline: &Pipeline{
					Sources: []string{"."},
				},
			},
			expected: []string{emptyDir},
		},

		"empty directory, all sources": {
			p: &pkg{
				path: emptyDir,
				pipeline: &Pipeline{
					Sources: []string{"./*"},
				},
			},
			expected: []string{emptyDir},
		},

		"directory with subpackages, current directory only": {
			p: &pkg{
				path: dir,
				pipeline: &Pipeline{
					Sources: []string{"."},
				},
			},
			expected: []string{dir},
		},

		"directory with subpackages, all sources": {
			p: &pkg{
				path: dir,
				pipeline: &Pipeline{
					Sources: []string{"./*"},
				},
			},
			expected: append([]string{dir}, subDirs...),
		},
	}

	for _, test := range tests {
		actual := test.p.resolveSources()
		if len(actual) != len(test.expected) {
			t.Fatal("number of package paths not equal to expected")
		}
		for i, path := range actual {
			assert.Equal(t, test.expected[i], path)
		}
	}
}
