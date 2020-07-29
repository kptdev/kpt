package cmdfix_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdfix"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/stretchr/testify/assert"
)

func TestFixSettersV1(t *testing.T) {
	var tests = []struct {
		name            string
		input           string
		notTrackedByGit bool
		err             string
		args            []string
		expectedOut     string
		openAPIFile     string
		expectedOutput  string
		expectedOpenAPI string
	}{
		{
			name: "upgrade-delete-partial-setters",
			input: `
apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
spec:
  profile: asm # {"type":"string","x-kustomize":{"setter":{"name":"profile","value":"asm"}}}
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
 `,

			openAPIFile: `apiVersion: kustomization.dev/v1alpha1
kind: Kustomization`,

			expectedOut: `processing resource configs to identify possible fixes... 
created setter with name cluster
created setter with name profile
created setter with name project
created 3 setters in total
created substitution with name subst-project-cluster
created 1 substitution in total
`,

			expectedOutput: `apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  cluster: "someproj/someclus" # {"$kpt-set":"subst-project-cluster"}
spec:
  profile: asm # {"$openapi":"profile"}
  cluster: "someproj/someclus" # {"$kpt-set":"subst-project-cluster"}
`,

			expectedOpenAPI: `apiVersion: kustomization.dev/v1alpha1
kind: Kustomization
openAPI:
  definitions:
    io.k8s.cli.setters.cluster:
      type: string
      x-k8s-cli:
        setter:
          name: cluster
          value: someclus
    io.k8s.cli.setters.profile:
      type: string
      x-k8s-cli:
        setter:
          name: profile
          value: asm
    io.k8s.cli.setters.project:
      type: string
      x-k8s-cli:
        setter:
          name: project
          value: someproj
    io.k8s.cli.substitutions.subst-project-cluster:
      x-k8s-cli:
        substitution:
          name: subst-project-cluster
          pattern: ${project}/${cluster}
          values:
          - marker: ${project}
            ref: '#/definitions/io.k8s.cli.setters.project'
          - marker: ${cluster}
            ref: '#/definitions/io.k8s.cli.setters.cluster'
`,
		},

		{
			name: "upgrade-delete-partial-setters-dryRun",
			args: []string{"--dry-run"},
			openAPIFile: `apiVersion: kustomization.dev/v1alpha1
kind: Kustomization`,
			input: `apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
spec:
  profile: asm # {"type":"string","x-kustomize":{"setter":{"name":"profile","value":"asm"}}}
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
`,
			expectedOut: `processing resource configs to identify possible fixes...  (dry-run)
created setter with name cluster (dry-run)
created setter with name profile (dry-run)
created setter with name project (dry-run)
created 3 setters in total (dry-run)
created substitution with name subst-project-cluster (dry-run)
created 1 substitution in total (dry-run)
`,
			expectedOutput: `apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
spec:
  profile: asm # {"type":"string","x-kustomize":{"setter":{"name":"profile","value":"asm"}}}
  cluster: "someproj/someclus" # {"type":"string","x-kustomize":{"partialSetters":[{"name":"project","value":"someproj"},{"name":"cluster","value":"someclus"}]}}
`,
		},

		{
			name:            "not-tracked-by-git",
			notTrackedByGit: true,
			input: `
apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  clusterName: "project-id/us-east1-d/cluster-name"
spec:
  profile: asm # {"type":"string","x-kustomize":{"setter":{"name":"profilesetter","value":"asm"}}}
  hub: gcr.io/asm-testing
 `,

			err: "kpt packages must be tracked by git",
		},

		{
			name: "no-openAPI-file-error",
			input: `
apiVersion: install.istio.io/v1alpha2
kind: IstioControlPlane
metadata:
  clusterName: "project-id/us-east1-d/cluster-name"
spec:
  profile: asm # {"type":"string","x-kustomize":{"setter":{"name":"profilesetter","value":"asm"}}}
  hub: gcr.io/asm-testing
 `,

			err: "Kptfile:",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			openAPIFileName := "Kptfile"

			dir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			defer os.RemoveAll(dir)

			err = ioutil.WriteFile(filepath.Join(dir, "deploy.yaml"), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if test.openAPIFile != "" {
				err = ioutil.WriteFile(filepath.Join(dir, openAPIFileName), []byte(test.openAPIFile), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			if !test.notTrackedByGit {
				gitRunner := gitutil.NewLocalGitRunner(dir)
				if !assert.NoError(t, gitRunner.Run("init", ".")) {
					t.FailNow()
				}
				if !assert.NoError(t, gitRunner.Run("add", ".")) {
					t.FailNow()
				}
				if !assert.NoError(t, gitRunner.Run("commit", "-m", "commit local package -- ds1")) {
					t.FailNow()
				}
			}
			out := &bytes.Buffer{}
			r := cmdfix.NewRunner("kpt")
			r.Command.SetArgs(append([]string{dir}, test.args...))
			r.Command.SetOut(out)
			err = r.Command.Execute()
			if test.err == "" {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			} else {
				if !assert.Contains(t, err.Error(), test.err) {
					t.FailNow()
				}
				return
			}

			if test.expectedOpenAPI != "" {
				actualOpenAPI, err := ioutil.ReadFile(filepath.Join(dir, openAPIFileName))
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, test.expectedOpenAPI, string(actualOpenAPI))
			}
			assert.Equal(t, test.expectedOut, out.String())
		})
	}
}
