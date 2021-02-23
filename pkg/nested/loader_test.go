package nested

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

func TestPathManifestReader_Read(t *testing.T) {
	testCases := map[string]struct {
		manifests        map[string]string
		subpackageManifests map[string]string
		namespace        string
		enforceNamespace bool
		validate         bool

		infosCount int
		namespaces []string
	}{
		"namespace should be set if not already present": {
			manifests: map[string]string{
				"dep.yaml": depManifest,
			},
			namespace:        "foo",
			enforceNamespace: true,

			infosCount: 1,
			namespaces: []string{"foo"},
		},
		"multiple manifests": {
			manifests: map[string]string{
				"dep.yaml": depManifest,
				"cm.yaml":  cmManifest,
			},
			namespace:        "default",
			enforceNamespace: true,

			infosCount: 2,
			namespaces: []string{"default", "default"},
		},
		"multiple manifests with Kptfile": {
			manifests: map[string]string{
				"dep.yaml": depManifest,
				"cm.yaml":  cmManifest,
				"Kptfile": kptFile,
			},
			namespace:        "default",
			enforceNamespace: true,

			infosCount: 3,
			namespaces: []string{"default", "default", "default"},
		},
		"multiple manifests with subpackages": {
			manifests: map[string]string{
				"dep.yaml": depManifest,
				"cm.yaml":  cmManifest,
				"Kptfile": kptFile,
			},
			subpackageManifests: map[string]string{
				"cm.yaml": subpackageManifest,
				"Kptfile": subpackageKptfile,
			},
			namespace:        "default",
			enforceNamespace: true,

			infosCount: 4,
			namespaces: []string{"default", "default", "default", "default"},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()

			dir, err := ioutil.TempDir("", "path-reader-test")
			assert.NoError(t, err)
			if tc.subpackageManifests != nil {
				err = os.Mkdir(filepath.Join(dir, "subpackage"), 0700)
				assert.NoError(t, err)
			}
			for filename, content := range tc.manifests {
				p := filepath.Join(dir, filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}
			for filename, content := range tc.subpackageManifests {
				p := filepath.Join(filepath.Join(dir), "subpackage", filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}

			loader := NewLoader(tf)
			_, err = loader.Read(nil, []string{dir})
			assert.NoError(t, err)
		})
	}
}

var (
	depManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 1
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
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`

	cmManifest = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: cm3
data:
  foo: bar
`
	kptFile = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: test
inventory:
  namespace: default
  name: inventory-test
  inventoryID: inventory-test
`
	subpackageManifest = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: cm1
data:
  foo: bar
`
	subpackageKptfile = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: subpackage
inventory:
  namespace: default
  name: inventory-subpackage-test
  inventoryID: inventory-subpackage-test
`
)
