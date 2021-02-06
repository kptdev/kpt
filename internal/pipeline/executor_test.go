package pipeline

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFilterMetaResources(t *testing.T) {
	tests := map[string]struct {
		resources []string
		expected  []string
	}{
		"no resources": {
			resources: nil,
			expected:  nil,
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
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-func-config
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/kpt-fn-contrib/helm-inflator:unstable
data:
  name: chart
  local-chart-path: /source`,
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

	for name := range tests {
		test := tests[name]
		t.Run(name, func(t *testing.T) {
			var nodes []*yaml.RNode

			for _, r := range test.resources {
				res, err := yaml.Parse(r)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				nodes = append(nodes, res)
			}

			filteredRes := filterMetaResources(nodes)
			if len(filteredRes) != len(test.expected) {
				t.Fatal("length of filtered resources not equal to expected")
			}

			for i, r := range filteredRes {
				res, err := r.String()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, test.expected[i], res)
			}
		})
	}
}

// creates a directory and writes a Kptfile
func writePkg(path string) (*pkg, error) {
	dir, err := ioutil.TempDir(path, "")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(filepath.Join(
		dir, kptfile.KptFileName), []byte(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
`), 0600)
	if err != nil {
		return nil, err
	}
	p, err := newPkg(dir)
	return p, err
}

func TestResolveSource(t *testing.T) {
	emptyPkg, err := writePkg("")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(emptyPkg.Path())

	// package with subpackages
	p, err := writePkg("")
	defer os.RemoveAll(p.Path())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// subdirectories of dir
	subp1, err := writePkg(p.Path())
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	subp2, err := writePkg(p.Path())
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(subp1.Path())
	defer os.RemoveAll(subp2.Path())

	// sorting for testing purposes
	subDirs := []string{subp1.Path(), subp2.Path()}
	sort.Strings(subDirs)

	tests := map[string]struct {
		source   string
		pkgPath  string
		expected []string
	}{
		"empty directory, current directory only": {
			pkgPath:  emptyPkg.Path(),
			source:   sourceCurrentPkg,
			expected: []string{emptyPkg.Path()},
		},

		"empty directory, all sources": {
			pkgPath:  emptyPkg.Path(),
			source:   sourceAllSubPkgs,
			expected: []string{emptyPkg.Path()},
		},

		"directory with subpackages, current directory only": {
			pkgPath:  p.Path(),
			source:   sourceCurrentPkg,
			expected: []string{p.Path()},
		},

		"directory with subpackages, all sources": {
			pkgPath:  p.Path(),
			source:   sourceAllSubPkgs,
			expected: append([]string{p.Path()}, subDirs...),
		},
	}

	for name := range tests {
		test := tests[name]
		t.Run(name, func(t *testing.T) {
			actual, err := resolveSource(test.source, test.pkgPath)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			if len(actual) != len(test.expected) {
				t.Fatal("number of package paths not equal to expected")
			}
			for i, path := range actual {
				assert.Equal(t, test.expected[i], path)
			}
		})
	}
}
