package setters

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestDefExists(t *testing.T) {
	var tests = []struct {
		name           string
		inputKptfile   string
		setterName     string
		expectedResult bool
	}{
		{
			name: "def exists",
			inputKptfile: `
apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: 123
          setBy: me
 `,
			setterName:     "gcloud.project.projectNumber",
			expectedResult: true,
		},
		{
			name: "def doesn't exist",
			inputKptfile: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
 `,
			setterName:     "gcloud.project.projectNumber",
			expectedResult: false,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			defer os.RemoveAll(dir)
			err = ioutil.WriteFile(filepath.Join(dir, "Kptfile"), []byte(test.inputKptfile), 0600)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedResult, DefExists(dir, test.setterName))
		})
	}
}
