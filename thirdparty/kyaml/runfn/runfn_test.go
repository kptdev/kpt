// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package runfn

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
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
metadata:
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/example.com/image:version
    config.kubernetes.io/local-config: "true"
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

	KptfileData = `apiVersion: kpt.dev/v1alpha2
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
			instance: RunFns{Functions: []*yaml.RNode{{}}},
			expected: RunFns{Output: os.Stdout, Input: os.Stdin, Functions: []*yaml.RNode{{}}},
		},
		{
			name:     "explicit functions -- yes functions from input",
			instance: RunFns{Functions: []*yaml.RNode{{}}},
			expected: RunFns{Output: os.Stdout, Input: os.Stdin, Functions: []*yaml.RNode{{}}},
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

func TestRunFns_sortFns(t *testing.T) {
	testCases := []struct {
		name           string
		nodes          []*yaml.RNode
		expectedImages []string
		expectedErrMsg string
	}{
		{
			name: "multiple functions in the same file are ordered by index",
			nodes: []*yaml.RNode{
				yaml.MustParse(`
metadata:
  annotations:
    config.kubernetes.io/path: functions.yaml
    config.kubernetes.io/index: 1
    config.kubernetes.io/function: |
      container:
        image: a
`),
				yaml.MustParse(`
metadata:
  annotations:
    config.kubernetes.io/path: functions.yaml
    config.kubernetes.io/index: 0
    config.kubernetes.io/function: |
      container:
        image: b
`),
			},
			expectedImages: []string{"b", "a"},
		},
		{
			name: "non-integer value in index annotation is an error",
			nodes: []*yaml.RNode{
				yaml.MustParse(`
metadata:
  annotations:
    config.kubernetes.io/path: functions.yaml
    config.kubernetes.io/index: 0
    config.kubernetes.io/function: |
      container:
        image: a
`),
				yaml.MustParse(`
metadata:
  annotations:
    config.kubernetes.io/path: functions.yaml
    config.kubernetes.io/index: abc
    config.kubernetes.io/function: |
      container:
        image: b
`),
			},
			expectedErrMsg: "strconv.Atoi: parsing \"abc\": invalid syntax",
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			packageBuff := &kio.PackageBuffer{
				Nodes: test.nodes,
			}

			err := sortFns(packageBuff)
			if test.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Equal(t, test.expectedErrMsg, err.Error())
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var images []string
			for _, n := range packageBuff.Nodes {
				spec := runtimeutil.GetFunctionSpec(n)
				images = append(images, spec.Container.Image)
			}

			assert.Equal(t, test.expectedImages, images)
		})
	}
}

func TestCmd_Execute(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	fn, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithFakePrinter(nil, nil),
		Path:                   dir,
		functionFilterProvider: getFilterProvider(t),
		Functions:              []*yaml.RNode{fn},
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := ioutil.ReadFile(
		filepath.Join(dir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "kind: StatefulSet")
}

func TestCmd_Execute_includeMetaResources(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	fn, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}

	// write a Kptfile to the directory of configuration
	if !assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, v1alpha2.KptFileName), []byte(KptfileData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithFakePrinter(nil, nil),
		Path:                   dir,
		functionFilterProvider: getMetaResourceFilterProvider(),
		IncludeMetaResources:   true,
		Functions:              []*yaml.RNode{fn},
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := ioutil.ReadFile(
		filepath.Join(dir, v1alpha2.KptFileName))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "foo: baz")
}

func TestCmd_Execute_notIncludeMetaResources(t *testing.T) {
	dir := setupTest(t)
	defer os.RemoveAll(dir)

	// write a test filter to the directory of configuration
	if !assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "filter.yaml"), []byte(ValueReplacerYAMLData), 0600)) {
		return
	}

	// write a Kptfile to the directory of configuration
	if !assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, v1alpha2.KptFileName), []byte(KptfileData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithFakePrinter(nil, nil),
		Path:                   dir,
		functionFilterProvider: getMetaResourceFilterProvider(),
		fnResults:              fnresult.NewResultList(),
	}
	if !assert.NoError(t, instance.Execute()) {
		t.FailNow()
	}
	b, err := ioutil.ReadFile(
		filepath.Join(dir, v1alpha2.KptFileName))
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
	tmpF, err := ioutil.TempFile("", "filter*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	os.RemoveAll(tmpF.Name())
	if !assert.NoError(t, ioutil.WriteFile(tmpF.Name(), []byte(ValueReplacerFnConfigYAMLData), 0600)) {
		return
	}

	fn, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}

	// run the functions, providing the path to the directory of filters
	instance := RunFns{
		Ctx:          fake.CtxWithFakePrinter(nil, nil),
		FnConfigPath: tmpF.Name(),
		Path:         dir,
		Functions:    []*yaml.RNode{fn},
		fnResults:    fnresult.NewResultList(),
	}
	instance.functionFilterProvider = getFnConfigPathFilterProvider(t, &instance)
	// initialize the defaults
	assert.NoError(t, instance.init())

	err = instance.Execute()
	if !assert.NoError(t, err) {
		return
	}
	b, err := ioutil.ReadFile(
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

	fn, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	instance := RunFns{
		Ctx:                    fake.CtxWithFakePrinter(nil, nil),
		Output:                 out, // write to out
		Path:                   dir,
		functionFilterProvider: getFilterProvider(t),
		Functions:              []*yaml.RNode{fn},
		fnResults:              fnresult.NewResultList(),
	}
	// initialize the defaults
	assert.NoError(t, instance.init())

	if !assert.NoError(t, instance.Execute()) {
		return
	}
	b, err := ioutil.ReadFile(
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
	fn, err := yaml.Parse(ValueReplacerYAMLData)
	if err != nil {
		t.Fatal(err)
	}

	read, err := kio.LocalPackageReader{PackagePath: dir}.Read()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	input := &bytes.Buffer{}
	if !assert.NoError(t, kio.ByteWriter{Writer: input}.Write(read)) {
		t.FailNow()
	}

	outDir, err := ioutil.TempDir("", "kustomize-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, ioutil.WriteFile(
		filepath.Join(dir, "filter.yaml"), []byte(ValueReplacerYAMLData), 0600)) {
		return
	}

	instance := RunFns{
		Ctx:                    fake.CtxWithFakePrinter(nil, nil),
		Input:                  input, // read from input
		Path:                   outDir,
		functionFilterProvider: getFilterProvider(t),
		Functions:              []*yaml.RNode{fn},
		fnResults:              fnresult.NewResultList(),
	}
	// initialize the defaults
	assert.NoError(t, instance.init())

	if !assert.NoError(t, instance.Execute()) {
		return
	}
	b, err := ioutil.ReadFile(
		filepath.Join(outDir, "java", "java-deployment.resource.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Contains(t, string(b), "kind: StatefulSet")
}

func getGeneratorFilterProvider(t *testing.T) func(runtimeutil.FunctionSpec, *yaml.RNode, currentUserFunc) (kio.Filter, error) {
	return func(f runtimeutil.FunctionSpec, node *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error) {
		return kio.FilterFunc(func(items []*yaml.RNode) ([]*yaml.RNode, error) {
			if f.Container.Image == "generate" {
				node, err := yaml.Parse("kind: generated")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				return append(items, node), nil
			}
			return items, nil
		}), nil
	}
}
func TestRunFns_ContinueOnEmptyResult(t *testing.T) {
	fn1, err := yaml.Parse(`
kind: fakefn
metadata:
  annotations:
    config.kubernetes.io/function: |
      container:
        image: pass
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	fn2, err := yaml.Parse(`
kind: fakefn
metadata:
  annotations:
    config.kubernetes.io/function: |
      container:
        image: generate
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	var test = []struct {
		ContinueOnEmptyResult bool
		ExpectedOutput        string
	}{
		{
			ContinueOnEmptyResult: false,
			ExpectedOutput:        "",
		},
		{
			ContinueOnEmptyResult: true,
			ExpectedOutput: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - kind: generated
`,
		},
	}
	for i := range test {
		ouputBuffer := bytes.Buffer{}
		r := RunFns{
			Ctx:                    fake.CtxWithFakePrinter(nil, nil),
			Input:                  bytes.NewReader([]byte{}),
			Output:                 &ouputBuffer,
			Functions:              []*yaml.RNode{fn1, fn2},
			functionFilterProvider: getGeneratorFilterProvider(t),
			ContinueOnEmptyResult:  test[i].ContinueOnEmptyResult,
		}
		if !assert.NoError(t, r.Execute()) {
			t.FailNow()
		}
		assert.Equal(t, test[i].ExpectedOutput, ouputBuffer.String())
	}
}

// setupTest initializes a temp test directory containing test data
func setupTest(t *testing.T) string {
	dir, err := ioutil.TempDir("", "kustomize-kyaml-test")
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
