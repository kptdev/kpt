package cmdsetdependency

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

func TestSetDependencyCommand(t *testing.T) {
	var tests = []struct {
		name            string
		inputKptFile    string
		args            []string
		expectedKptfile string
		err             string
	}{
		{
			name: "Set dependency",
			args: []string{
				"https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0", "hello-world"},
			inputKptFile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			expectedKptfile: `
apiVersion: v1alpha1
kind: Kptfile
dependencies:
- name: hello-world
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/helloworld-set
    ref: v0.1.0
  updateStrategy: fast-forward
 `,
		},
		{
			name: "Set dependency error strategy",
			args: []string{
				"https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0", "hello-world", "--strategy", "some"},
			inputKptFile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			expectedKptfile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			err: `provided update strategy "some" is invalid, must be one of ["fast-forward" "force-delete-replace" "alpha-git-patch" "resource-merge"]`,
		},
		{
			name: "Set dependency rel path error",
			args: []string{
				"https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0", "non-existent-dir/hello-world"},
			inputKptFile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			expectedKptfile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			err: `parent directory non-existent-dir does not exist`,
		},
		{
			name: "Set dependency rel path",
			args: []string{
				"https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.1.0", "somedir/hello-world"},
			inputKptFile: `
apiVersion: v1alpha1
kind: Kptfile
 `,
			expectedKptfile: `
apiVersion: v1alpha1
kind: Kptfile
dependencies:
- name: somedir/hello-world
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/helloworld-set
    ref: v0.1.0
  updateStrategy: fast-forward
 `,
		},
		{
			name: "Update existing dependency",
			args: []string{
				"https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.2.0", "hello-world"},
			inputKptFile: `
apiVersion: v1alpha1
kind: Kptfile
dependencies:
  - name: hello-world
    git:
        repo: https://github.com/GoogleContainerTools/kpt
        directory: /package-examples/helloworld-set
        ref: v0.1.0
    updateStrategy: resource-merge
 `,
			expectedKptfile: `
apiVersion: v1alpha1
kind: Kptfile
dependencies:
- name: hello-world
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/helloworld-set
    ref: v0.2.0
  updateStrategy: resource-merge
 `,
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
			err = os.MkdirAll(filepath.Join(baseDir, "somedir"), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			f := filepath.Join(baseDir, kptfile.KptFileName)
			err = ioutil.WriteFile(f, []byte(test.inputKptFile), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			runner := NewSetDependencyRunner("")
			out := &bytes.Buffer{}
			runner.Command.SetOut(out)
			err = os.Chdir(baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			runner.Command.SetArgs(test.args)
			err = runner.Command.Execute()

			if test.err != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Equal(t, test.err, err.Error()) {
					t.FailNow()
				}
			} else if !assert.NoError(t, err) {
				t.FailNow()
			}

			actualOpenAPI, err := ioutil.ReadFile(f)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedKptfile),
				strings.TrimSpace(string(actualOpenAPI))) {
				t.FailNow()
			}
		})
	}
}
