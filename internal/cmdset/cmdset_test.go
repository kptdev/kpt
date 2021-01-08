package cmdset

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
)

func TestSetCommand(t *testing.T) {
	var tests = []struct {
		name              string
		inputOpenAPI      string
		input             string
		args              []string
		out               string
		expectedOpenAPI   string
		expectedResources string
		errMsg            string
	}{
		{
			name: "set replicas",
			args: []string{"--value", "replicas=4", "--description", "hi there", "--set-by", "pw"},
			out:  "set 1 field(s)\n",
			inputOpenAPI: `
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
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hi there
      x-k8s-cli:
        setter:
          name: replicas
          value: "4"
          setBy: pw
          isSet: true
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4 # {"$kpt-set":"${replicas}"}
 `,
		},
		{
			name: "set list values",
			args: []string{"--value", "list=[10, 11]"},
			out:  "set 1 field(s)\n",
			inputOpenAPI: `
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.list:
      type: array
      x-k8s-cli:
        setter:
          name: list
          listValues:
          - 0
 `,
			input: `
apiVersion: example.com/v1beta1
kind: Example
spec:
  list: # {"$kpt-set":"${list}"}
  - 0
 `,
			expectedOpenAPI: `
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.list:
      type: array
      x-k8s-cli:
        setter:
          name: list
          listValues:
          - "10"
          - "11"
          isSet: true
 `,
			expectedResources: `
apiVersion: example.com/v1beta1
kind: Example
spec:
  list: # {"$kpt-set":"${list}"}
  - "10"
  - "11"
 `,
		},
		{
			name: "incorrect value input",
			args: []string{"--value", "4"},
			out:  "set 1 field(s)\n",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
 `,
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
			errMsg: `input to value flag must follow the format "SETTER_NAME=SETTER_VALUE"`,
		},
		{
			name: "setter doesn't exist",
			args: []string{"--value", "non-existent-setter=4"},
			out:  "set 1 field(s)\n",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
 `,
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"${replicas}"}
 `,
			errMsg: `setter "non-existent-setter" is not found`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			f := filepath.Join(baseDir, kptfile.KptFileName)
			err = ioutil.WriteFile(f, []byte(test.inputOpenAPI), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			r, err := ioutil.TempFile(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.Remove(r.Name())
			err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			runner := NewSetRunner("")
			out := &bytes.Buffer{}
			runner.Command.SetOut(out)
			runner.Command.SetArgs(append([]string{baseDir}, test.args...))
			err = runner.Command.Execute()
			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			if test.errMsg == "" && !assert.Contains(t, out.String(), strings.TrimSpace(test.out)) {
				t.FailNow()
			}

			actualResources, err := ioutil.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedResources),
				strings.TrimSpace(string(actualResources))) {
				t.FailNow()
			}

			actualOpenAPI, err := ioutil.ReadFile(f)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedOpenAPI),
				strings.TrimSpace(string(actualOpenAPI))) {
				t.FailNow()
			}
		})
	}
}
