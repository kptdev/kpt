// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdeval

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

// TestRunFnCommand_preRunE verifies that preRunE correctly parses the commandline
// flags and arguments into the RunFns structure to be executed.
func TestRunFnCommand_preRunE(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedFnConfig string
		expectedFn       *runtimeutil.FunctionSpec
		expectedStruct   *runfn.RunFns
		expectedExecArgs []string
		err              string
		path             string
		input            io.Reader
		output           io.Writer
		fnConfigPath     string
		network          bool
		mount            []string
	}{
		{
			name: "config map",
			args: []string{"eval", "dir", "--image", "foo:bar", "--", "a=b", "c=d", "e=f"},
			path: "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {a: b, c: d, e: f}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name:   "config map stdin / stdout",
			args:   []string{"eval", "-", "--image", "foo:bar", "--", "a=b", "c=d", "e=f"},
			input:  os.Stdin,
			output: &bytes.Buffer{},
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {a: b, c: d, e: f}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name:   "config map dry-run",
			args:   []string{"eval", "dir", "--image", "foo:bar", "-o", "stdout", "--", "a=b", "c=d", "e=f"},
			output: &bytes.Buffer{},
			path:   "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {a: b, c: d, e: f}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name: "config map no args",
			args: []string{"eval", "dir", "--image", "foo:bar"},
			path: "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name:    "network enabled",
			args:    []string{"eval", "dir", "--image", "foo:bar", "--network"},
			path:    "dir",
			network: true,
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name:    "with network name",
			args:    []string{"eval", "dir", "--image", "foo:bar", "--network"},
			path:    "dir",
			network: true,
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name: "custom kind",
			args: []string{"eval", "dir", "--image", "foo:bar", "--", "Foo", "g=h"},
			path: "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {g: h}
kind: Foo
apiVersion: v1
`,
		},
		{
			name: "custom kind '=' in data",
			args: []string{"eval", "dir", "--image", "foo:bar", "--", "Foo", "g=h", "i=j=k"},
			path: "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {g: h, i: j=k}
kind: Foo
apiVersion: v1
`,
		},
		{
			name: "custom kind with storage mounts",
			args: []string{
				"eval", "dir", "--mount", "type=bind,src=/mount/path,dst=/local/",
				"--mount", "type=volume,src=myvol,dst=/local/",
				"--mount", "type=tmpfs,dst=/local/",
				"--image", "foo:bar", "--", "Foo", "g=h", "i=j=k"},
			path:  "dir",
			mount: []string{"type=bind,src=/mount/path,dst=/local/", "type=volume,src=myvol,dst=/local/", "type=tmpfs,dst=/local/"},
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {g: h, i: j=k}
kind: Foo
apiVersion: v1
`,
		},
		{
			name: "results_dir",
			args: []string{"eval", "dir", "--results-dir", "foo/", "--image", "foo:bar", "--", "a=b", "c=d", "e=f"},
			path: "dir",
			expectedStruct: &runfn.RunFns{
				Path:                  "dir",
				ResultsDir:            "foo/",
				ImagePullPolicy:       fnruntime.AlwaysPull,
				Env:                   []string{},
				ContinueOnEmptyResult: true,
				Ctx:                   context.TODO(),
			},
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {a: b, c: d, e: f}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name: "config map multi args",
			args: []string{"eval", "dir", "dir2", "--image", "foo:bar", "--", "a=b", "c=d", "e=f"},
			err:  "0 or 1 arguments supported",
		},
		{
			name: "config map not image",
			args: []string{"eval", "dir", "--", "a=b", "c=d", "e=f"},
			err:  "must specify --image",
		},
		{
			name: "config map bad data",
			args: []string{"eval", "dir", "--image", "foo:bar", "--", "a=b", "c", "e=f"},
			err:  "must have keys and values separated by",
		},
		{
			name: "envs",
			args: []string{"eval", "dir", "--env", "FOO=BAR", "-e", "BAR", "--image", "foo:bar"},
			path: "dir",
			expectedStruct: &runfn.RunFns{
				Path:                  "dir",
				ImagePullPolicy:       fnruntime.AlwaysPull,
				Env:                   []string{"FOO=BAR", "BAR"},
				ContinueOnEmptyResult: true,
				Ctx:                   context.TODO(),
			},
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name: "as current user",
			args: []string{"eval", "dir", "--as-current-user", "--image", "foo:bar"},
			path: "dir",
			expectedStruct: &runfn.RunFns{
				Path:                  "dir",
				AsCurrentUser:         true,
				ImagePullPolicy:       fnruntime.AlwaysPull,
				Env:                   []string{},
				ContinueOnEmptyResult: true,
				Ctx:                   context.TODO(),
			},
			expectedFn: &runtimeutil.FunctionSpec{
				Container: runtimeutil.ContainerSpec{
					Image: "gcr.io/kpt-fn/foo:bar",
				},
			},
			expectedFnConfig: `
metadata:
  name: function-input
data: {}
kind: ConfigMap
apiVersion: v1
`,
		},
		{
			name: "--fn-config flag",
			args: []string{"eval", "dir", "--fn-config", "a/b/c", "--image", "foo:bar"},
			err:  "missing function config file: a/b/c",
		},
		{
			name: "--fn-config with function arguments",
			args: []string{"eval", "dir", "--fn-config", "a/b/c", "--image", "foo:bar", "--", "a=b", "c=d", "e=f"},
			err:  "function arguments can only be specified without function config file",
		},
		{
			name: "exec args",
			args: []string{"eval", "dir", "--exec", "execPath arg1 'arg2 arg3'", "--", "a=b", "c=d", "e=f"},
			path: "dir",
			expectedFn: &runtimeutil.FunctionSpec{
				Exec: runtimeutil.ExecSpec{
					Path: "execPath",
				},
			},
			expectedExecArgs: []string{"arg1", "arg2 arg3"},
			expectedFnConfig: `
metadata:
  name: function-input
data: {a: b, c: d, e: f}
kind: ConfigMap
apiVersion: v1
`,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			r := GetEvalFnRunner(context.TODO(), "kpt")
			// Don't run the actual command
			r.Command.Run = nil
			r.Command.RunE = func(cmd *cobra.Command, args []string) error { return nil }
			r.Command.SilenceErrors = true
			r.Command.SilenceUsage = true

			// hack due to https://github.com/spf13/cobra/issues/42
			root := &cobra.Command{Use: "root"}
			root.AddCommand(r.Command)
			root.SetArgs(tt.args)

			// error case
			err := r.Command.Execute()
			if tt.err != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), tt.err) {
					t.FailNow()
				}
				// don't check anything else in error case
				return
			}

			// non-error case
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// check if Input was set
			if !assert.Equal(t, tt.input, r.RunFns.Input) {
				t.FailNow()
			}

			// check if Output was set
			if !assert.Equal(t, tt.output, r.RunFns.Output) {
				t.FailNow()
			}

			// check if Path was set
			if !assert.Equal(t, tt.path, r.RunFns.Path) {
				t.FailNow()
			}

			// check if Network was set
			if tt.network {
				if !assert.Equal(t, tt.network, r.RunFns.Network) {
					t.FailNow()
				}
			} else {
				if !assert.Equal(t, false, r.RunFns.Network) {
					t.FailNow()
				}
			}

			if !assert.Equal(t, tt.fnConfigPath, r.RunFns.FnConfigPath) {
				t.FailNow()
			}

			if !assert.Equal(t, r.RunFns, r.RunFns) {
				t.FailNow()
			}

			if !assert.Equal(t, toStorageMounts(tt.mount), r.RunFns.StorageMounts) {
				t.FailNow()
			}

			// check if function config was set
			if tt.expectedFnConfig != "" {
				actual := strings.TrimSpace(r.RunFns.FnConfig.MustString())
				if !assert.Equal(t, strings.TrimSpace(tt.expectedFnConfig), actual) {
					t.FailNow()
				}
			}

			// check if function was set
			if tt.expectedFn != nil {
				if !assert.NotNil(t, r.RunFns.Function) {
					t.FailNow()
				}
				if !assert.EqualValues(t, tt.expectedFn, r.RunFns.Function) {
					t.FailNow()
				}
			}

			// check if exec arguments were set
			if len(tt.expectedExecArgs) != 0 {
				if !assert.EqualValues(t, tt.expectedExecArgs, r.RunFns.ExecArgs) {
					t.FailNow()
				}
			}

			if tt.expectedStruct != nil {
				r.RunFns.Function = nil
				r.RunFns.FnConfig = nil
				tt.expectedStruct.FnConfigPath = tt.fnConfigPath
				if !assert.Equal(t, *tt.expectedStruct, r.RunFns) {
					t.FailNow()
				}
			}

		})
	}
}
