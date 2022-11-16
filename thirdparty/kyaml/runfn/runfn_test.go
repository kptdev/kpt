// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package runfn

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	ValueReplacerYAMLData = `apiVersion: v1
kind: ValueReplacer
stringMatch: Deployment
replace: StatefulSet
`

	ValueReplacerFnConfigYAMLData = `apiVersion: v1
kind: ValueReplacer
metadata:
  name: fn-config
stringMatch: Deployment
replace: ReplicaSet
`

	KptfileData = `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: kptfile
  annotations:
    foo: bar
`
)

func TestRunFns_Execute__initDefault(t *testing.T) {
	// droot: This is not a useful test at all, so skipping this
	t.Skip()
	b := &bytes.Buffer{}
	var tests = []struct {
		instance RunFns
		expected RunFns
		name     string
	}{
		{
			instance: RunFns{},
			name:     "empty",
			expected: RunFns{Output: os.Stdout, Input: os.Stdin},
		},
		{
			name:     "explicit output",
			instance: RunFns{Output: b},
			expected: RunFns{Output: b, Input: os.Stdin},
		},
		{
			name:     "explicit input",
			instance: RunFns{Input: b},
			expected: RunFns{Output: os.Stdout, Input: b},
		},
		{
			name:     "explicit functions -- no functions from input",
			instance: RunFns{Function: nil},
			expected: RunFns{Output: os.Stdout, Input: os.Stdin, Function: nil},
		},
		{
			name:     "explicit functions -- yes functions from input",
			instance: RunFns{Function: nil},
			expected: RunFns{Output: os.Stdout, Input: os.Stdin, Function: nil},
		},
		{
			name:     "explicit functions in paths -- no functions from input",
			instance: RunFns{FnConfigPath: "/foo"},
			expected: RunFns{
				Output:       os.Stdout,
				Input:        os.Stdin,
				FnConfigPath: "/foo",
			},
		},
		{
			name:     "functions in paths -- yes functions from input",
			instance: RunFns{FnConfigPath: "/foo"},
			expected: RunFns{
				Output:       os.Stdout,
				Input:        os.Stdin,
				FnConfigPath: "/foo",
			},
		},
		{
			name:     "explicit directories in mounts",
			instance: RunFns{StorageMounts: []runtimeutil.StorageMount{{MountType: "volume", Src: "myvol", DstPath: "/local/"}}},
			expected: RunFns{
				Output:        os.Stdout,
				Input:         os.Stdin,
				StorageMounts: []runtimeutil.StorageMount{{MountType: "volume", Src: "myvol", DstPath: "/local/"}},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, (&tt.instance).init())
			(&tt.instance).functionFilterProvider = nil
			if !assert.Equal(t, tt.expected, tt.instance) {
				t.FailNow()
			}
		})
	}
}

func TestCmd_Execute(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	fnConfig, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}
	fn := &runtimeutil.FunctionSpec{
		Container: runtimeutil.ContainerSpec{
			Image: "gcr.io/example.com/image:version",
		},
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithDefaultPrinter(),
		Path:                   dir,
		functionFilterProvider: getFilterProvider(t),
		Function:               fn,
		FnConfig:               fnConfig,
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := os.ReadFile(
		filepath.Join(dir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "kind: StatefulSet")
}

func TestCmd_Execute_includeMetaResources(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	fnConfig, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}
	fn := &runtimeutil.FunctionSpec{
		Container: runtimeutil.ContainerSpec{
			Image: "gcr.io/example.com/image:version",
		},
	}

	// write a Kptfile to the directory of configuration
	if !assert.NoError(t, os.WriteFile(
		filepath.Join(dir, v1.KptFileName), []byte(KptfileData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithDefaultPrinter(),
		Path:                   dir,
		functionFilterProvider: getMetaResourceFilterProvider(),
		Function:               fn,
		FnConfig:               fnConfig,
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := os.ReadFile(
		filepath.Join(dir, v1.KptFileName))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "foo: baz")
}

func TestCmd_Execute_notIncludeMetaResources(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	// write a test filter to the directory of configuration
	if !assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "filter.yaml"), []byte(ValueReplacerYAMLData), 0600)) {
		return
	}

	// write a Kptfile to the directory of configuration
	if !assert.NoError(t, os.WriteFile(
		filepath.Join(dir, v1.KptFileName), []byte(KptfileData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithDefaultPrinter(),
		Path:                   dir,
		functionFilterProvider: getMetaResourceFilterProvider(),
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := os.ReadFile(
		filepath.Join(dir, v1.KptFileName))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.EqualValues(t, string(b), KptfileData)
}

type TestFilter struct {
	invoked bool
	Exit    error
}

func (f *TestFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	f.invoked = true
	return input, nil
}

func (f *TestFilter) GetExit() error {
	return f.Exit
}

func getFnConfigPathFilterProvider(t *testing.T, r *RunFns) func(runtimeutil.FunctionSpec, *yaml.RNode, currentUserFunc) (kio.Filter, error) {
	return func(f runtimeutil.FunctionSpec, node *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error) {
		// parse the filter from the input
		filter := yaml.YFilter{}
		b := &bytes.Buffer{}
		e := yaml.NewEncoder(b)
		var err error
		if r.FnConfigPath != "" {
			node, err = r.getFunctionConfig()
			if err != nil {
				t.Fatal(err)
			}
		}
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
		}, nil
	}
}

func TestCmd_Execute_setFnConfigPath(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	// write a test filter to a separate directory
	tmpF, err := os.CreateTemp("", "filter*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	os.RemoveAll(tmpF.Name())
	if !assert.NoError(t, os.WriteFile(tmpF.Name(), []byte(ValueReplacerFnConfigYAMLData), 0600)) {
		return
	}

	fnConfig, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}
	fn := &runtimeutil.FunctionSpec{
		Container: runtimeutil.ContainerSpec{
			Image: "gcr.io/example.com/image:version",
		},
	}

	// run the functions, providing the path to the directory of filters
	instance := RunFns{
		Ctx:          fake.CtxWithDefaultPrinter(),
		FnConfigPath: tmpF.Name(),
		Path:         dir,
		Function:     fn,
		FnConfig:     fnConfig,
		fnResults:    fnresult.NewResultList(),
	}
	instance.functionFilterProvider = getFnConfigPathFilterProvider(t, &instance)
	// initialize the defaults
	assert.NoError(t, instance.init())

	err = instance.Execute()
	if !assert.NoError(t, err) {
		return
	}
	b, err := os.ReadFile(
		filepath.Join(dir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, string(b), "kind: ReplicaSet")
}

// TestCmd_Execute_setOutput tests the execution of a filter using an io.Writer as output
func TestCmd_Execute_setOutput(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	fnConfig, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}
	fn := &runtimeutil.FunctionSpec{
		Container: runtimeutil.ContainerSpec{
			Image: "gcr.io/example.com/image:version",
		},
	}

	out := &bytes.Buffer{}
	instance := RunFns{
		Ctx:                    fake.CtxWithDefaultPrinter(),
		Output:                 out, // write to out
		Path:                   dir,
		functionFilterProvider: getFilterProvider(t),
		Function:               fn,
		FnConfig:               fnConfig,
		fnResults:              fnresult.NewResultList(),
	}
	// initialize the defaults
	assert.NoError(t, instance.init())

	if !assert.NoError(t, instance.Execute()) {
		return
	}
	b, err := os.ReadFile(
		filepath.Join(dir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.NotContains(t, string(b), "kind: StatefulSet")
	assert.Contains(t, out.String(), "kind: StatefulSet")
}

// TestCmd_Execute_setInput tests the execution of a filter using an io.Reader as input
func TestCmd_Execute_setInput(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)
	fnConfig, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}
	fn := &runtimeutil.FunctionSpec{
		Container: runtimeutil.ContainerSpec{
			Image: "gcr.io/example.com/image:version",
		},
	}

	read, err := kio.LocalPackageReader{PackagePath: dir, PreserveSeqIndent: true}.Read()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	input := &bytes.Buffer{}
	if !assert.NoError(t, kio.ByteWriter{Writer: input}.Write(read)) {
		t.FailNow()
	}

	outDir, err := os.MkdirTemp("", "kustomize-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "filter.yaml"), []byte(ValueReplacerYAMLData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithDefaultPrinter(),
		Input:                  input, // read from input
		Path:                   outDir,
		functionFilterProvider: getFilterProvider(t),
		Function:               fn,
		FnConfig:               fnConfig,
		fnResults:              fnresult.NewResultList(),
	}
	// initialize the defaults
	assert.NoError(t, instance.init())

	if !assert.NoError(t, instance.Execute()) {
		return
	}
	b, err := os.ReadFile(
		filepath.Join(outDir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "kind: StatefulSet")
}

// setupTest initializes a temp test directory containing test data
func setupTest(t *testing.T) string {
	dir, err := os.MkdirTemp("", "kustomize-kyaml-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	_, filename, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	ds, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "test", "testdata"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, copyutil.CopyDir(ds, dir)) {
		t.FailNow()
	}
	if !assert.NoError(t, os.Chdir(filepath.Dir(dir))) {
		t.FailNow()
	}
	return dir
}

// getFilterProvider fakes the creation of a filter, replacing the ContainerFiler with
// a filter to s/kind: Deployment/kind: StatefulSet/g.
// this can be used to simulate running a filter.
func getFilterProvider(t *testing.T) func(runtimeutil.FunctionSpec, *yaml.RNode, currentUserFunc) (kio.Filter, error) {
	return func(f runtimeutil.FunctionSpec, node *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error) {
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
		}, nil
	}
}

// getMetaResourceFilterProvider fakes the creation of a filter, replacing the
// ContainerFilter with replace the value for annotation "foo" to "baz"
func getMetaResourceFilterProvider() func(runtimeutil.FunctionSpec, *yaml.RNode, currentUserFunc) (kio.Filter, error) {
	return func(f runtimeutil.FunctionSpec, node *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error) {
		return filters.Modifier{
			Filters: []yaml.YFilter{{Filter: yaml.SetAnnotation("foo", "baz")}},
		}, nil
	}
}

func TestRunFns_mergeContainerEnv(t *testing.T) {
	testcases := []struct {
		name      string
		instance  RunFns
		inputEnvs []string
		expect    runtimeutil.ContainerEnv
	}{
		{
			name:     "all empty",
			instance: RunFns{},
			expect:   *runtimeutil.NewContainerEnv(),
		},
		{
			name:      "empty command line envs",
			instance:  RunFns{},
			inputEnvs: []string{"foo=bar"},
			expect:    *runtimeutil.NewContainerEnvFromStringSlice([]string{"foo=bar"}),
		},
		{
			name: "empty declarative envs",
			instance: RunFns{
				Env: []string{"foo=bar"},
			},
			expect: *runtimeutil.NewContainerEnvFromStringSlice([]string{"foo=bar"}),
		},
		{
			name: "same key",
			instance: RunFns{
				Env: []string{"foo=bar", "foo"},
			},
			inputEnvs: []string{"foo=bar1", "bar"},
			expect:    *runtimeutil.NewContainerEnvFromStringSlice([]string{"foo=bar", "bar", "foo"}),
		},
		{
			name: "same exported key",
			instance: RunFns{
				Env: []string{"foo=bar", "foo"},
			},
			inputEnvs: []string{"foo1=bar1", "foo"},
			expect:    *runtimeutil.NewContainerEnvFromStringSlice([]string{"foo=bar", "foo1=bar1", "foo"}),
		},
	}

	for i := range testcases {
		tc := testcases[i]
		t.Run(tc.name, func(t *testing.T) {
			envs := tc.instance.mergeContainerEnv(tc.inputEnvs)
			assert.Equal(t, tc.expect.GetDockerFlags(), runtimeutil.NewContainerEnvFromStringSlice(envs).GetDockerFlags())
		})
	}
}
