// Copyright 2022-2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fn

import (
	"reflect"
	"sort"
	"testing"

	"github.com/go-errors/errors"
	"github.com/google/go-cmp/cmp"
	schema "github.com/kptdev/kpt/api/schema/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGVK(t *testing.T) {
	input := []byte(`
apiVersion: apps/v3
kind: StatefulSet
metadata:
  name: my-config
spec:
  volumeClaimTemplates:
    - metadata:
        labels:
          testkey: testvalue
`)

	inputNoGroup := []byte(`
apiVersion: v3
kind: StatefulSet
metadata:
  name: my-config
spec:
  volumeClaimTemplates:
    - metadata:
        labels:
          testkey: testvalue
`)

	inputMismatch := []byte(`
apiVersion: apps/v1
kind: Service
metadata:
  name: my-config
spec:
  volumeClaimTemplates:
    - metadata:
        labels:
          testkey: testvalue
`)
	resource, _ := ParseKubeObject(input)
	resourceNoGroup, _ := ParseKubeObject(inputNoGroup)
	resourceMismatch, _ := ParseKubeObject(inputMismatch)
	testCases := map[string]struct {
		resource         *KubeObject
		resourceNoGroup  *KubeObject
		resourceDiffKind *KubeObject
		group            string
		version          string
		kind             string
		// this is the expected result for the resource with matched group, version, kind
		expect bool
		// this is the expected result for the resource with no group, matched version and kind
		expectNoGroup bool
		// this is the expected result for the resource with mismatch version and kind
		expectMismatch bool
	}{
		"IsGVK provided with no version, should match all versions": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "apps",
			version: "",
			kind:    "StatefulSet",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with no group, should match all groups": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "",
			version: "v3",
			kind:    "StatefulSet",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with no kind, should match all kinds": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "apps",
			version: "v3",
			kind:    "",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with no fields, should match all resources": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "",
			version: "",
			kind:    "",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: true,
		},
		"IsGVK provided with all fields, can only match if the resource has the same field value or the field is nil": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "apps",
			version: "v3",
			kind:    "StatefulSet",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with only kind, can only match if the kind is the same or kind is nil": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "",
			version: "",
			kind:    "StatefulSet",

			expect:         true,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with only group, can only match if the group is the same or group is nil": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "appWrong",
			version: "",
			kind:    "",

			expect:         false,
			expectNoGroup:  true,
			expectMismatch: false,
		},
		"IsGVK provided with only version, can only match if the version is the same or version is nil": {
			resource:         resource,
			resourceNoGroup:  resourceNoGroup,
			resourceDiffKind: resourceMismatch,

			group:   "",
			version: "v1",
			kind:    "",

			expect:         false,
			expectNoGroup:  false,
			expectMismatch: true,
		},
	}

	for testName, data := range testCases {
		t.Run(testName+"/resource", func(t *testing.T) {
			assert.Equal(t, resource.IsGVK(data.group, data.version, data.kind), data.expect)
		})
		t.Run(testName+"/resourceNoGroup", func(t *testing.T) {
			assert.Equal(t, resourceNoGroup.IsGVK(data.group, data.version, data.kind), data.expectNoGroup)
		})
		t.Run(testName+"/resourceMismatch", func(t *testing.T) {
			assert.Equal(t, resourceMismatch.IsGVK(data.group, data.version, data.kind), data.expectMismatch)
		})
	}
}

func TestIsLocalConfig(t *testing.T) {
	kptFile := []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configPath: fn-config.yaml
`)
	item := []byte(`
apiVersion: v1
kind: Service
metadata:
 name: whatever
 labels:
   app: myApp
`)
	rl := &ResourceList{}
	kptFileItem, _ := ParseKubeObject(kptFile)
	serviceItem, _ := ParseKubeObject(item)
	rl.Items = []*KubeObject{kptFileItem, serviceItem}
	for _, o := range rl.Items.WhereNot(IsLocalConfig) {
		assert.Equal(t, o.GetString("kind"), "Service")
	}
}

func TestSetSlice(t *testing.T) {
	var original = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: write-pods
  namespace: default
subjects:
- kind: User
  apiGroup: testing.group
`)

	var other = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: default
subjects:
- kind: User
  apiGroup: rbac.authorization.k8s.io
- kind: Admin
  apiGroup: rbac.authorization.k8s.io
`)

	var expected = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: write-pods
  namespace: default
subjects:
- kind: User
  apiGroup: rbac.authorization.k8s.io
- kind: Admin
  apiGroup: rbac.authorization.k8s.io
`
	originalObj, err := ParseKubeObject(original)
	require.NoError(t, err)
	otherObj, err := ParseKubeObject(other)
	require.NoError(t, err)
	slicesToAdd := otherObj.GetSlice("subjects")
	require.NoError(t, originalObj.SetSlice(slicesToAdd, "subjects"))
	assert.Equal(t, originalObj.String(), expected)
}

func TestIsNamespaceScoped(t *testing.T) {
	testdata := map[string]struct {
		input    []byte
		expected bool
	}{
		"k8s resource, namespace scoped but unset": {
			input: []byte(`
apiVersion: v1
kind: ResourceQuota
metadata:
  name: example
spec:
  hard:
    limits.cpu: '10'
`),
			expected: true,
		},
		"k8s resource, cluster scoped": {
			input: []byte(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: 
  name: example
subjects:
- kind: ServiceAccount
  name: example
  apiGroup: rbac.authorization.k8s.io
`),
			expected: false,
		},
		"custom resource, namespace set": {
			input: []byte(`
apiVersion: kpt-test
kind: KptTestResource
metadata: 
  name: example
  namespace: example
`),
			expected: true,
		},
		"custom resource, namespace unset": {
			input: []byte(`
apiVersion: kpt-test
kind: KptTestResource
metadata: 
  name: example
`),
			expected: false,
		},
	}
	for description, data := range testdata {
		o, _ := ParseKubeObject(data.input)
		if o.IsNamespaceScoped() != data.expected {
			t.Errorf("%v failed, resource namespace scope: got %v, want  %v", description, o.IsNamespaceScoped(), data.expected)
		}
	}
}

var noFnConfigResourceList = []byte(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
`)

func TestNilFnConfigResourceList(t *testing.T) {
	rl, err := ParseResourceList(noFnConfigResourceList)
	if err != nil {
		t.Fatalf("Error parsing resource list: %v", err)
	}
	if rl.FunctionConfig == nil {
		t.Errorf("Empty functionConfig in ResourceList should still be initialized to avoid nil pointer error")
	}
	if !rl.FunctionConfig.IsEmpty() {
		t.Errorf("The dummy fnConfig should be surfaced and checkable.")
	}
	// Check that FunctionConfig should be able to call KRM methods even if its Nil"
	{
		if rl.FunctionConfig.GetKind() != "" {
			t.Errorf("Nil KubeObject cannot call GetKind()")
		}
		if rl.FunctionConfig.GetAPIVersion() != "" {
			t.Errorf("Nil KubeObject cannot call GetAPIVersion()")
		}
		if rl.FunctionConfig.GetName() != "" {
			t.Errorf("Nil KubeObject cannot call GetName()")
		}
		if rl.FunctionConfig.GetNamespace() != "" {
			t.Errorf("Nil KubeObject cannot call GetNamespace()")
		}
		if rl.FunctionConfig.GetAnnotation("X") != "" {
			t.Errorf("Nil KubeObject cannot call GetAnnotation()")
		}
		if rl.FunctionConfig.GetLabel("Y") != "" {
			t.Errorf("Nil KubeObject cannot call GetLabel()")
		}
	}
	// Check that nil FunctionConfig can use SubObject methods.
	{
		_, found, err := rl.FunctionConfig.NestedString("not-exist")
		if found || err != nil {
			t.Errorf("Nil KubeObject shall not have the field path `not-exist` exist, and not expect errors")
		}
	}
	// Check that nil FunctionConfig should be editable.
	{
		err = rl.FunctionConfig.SetKind("CustomFn")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetKind() != "CustomFn" {
			t.Errorf("Nil KubeObject cannot call SetKind()")
		}
		err = rl.FunctionConfig.SetAPIVersion("kpt.fn/v1")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetAPIVersion() != "kpt.fn/v1" {
			t.Errorf("Nil KubeObject cannot call SetAPIVersion()")
		}
		err = rl.FunctionConfig.SetName("test")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetName() != "test" {
			t.Errorf("Nil KubeObject cannot call SetName()")
		}
		err = rl.FunctionConfig.SetNamespace("test-ns")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetNamespace() != "test-ns" {
			t.Errorf("Nil KubeObject cannot call SetNamespace()")
		}
		err = rl.FunctionConfig.SetAnnotation("k", "v")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetAnnotation("k") != "v" {
			t.Errorf("Nil KubeObject cannot call SetAnnotation()")
		}
		err = rl.FunctionConfig.SetLabel("k", "v")
		assert.NoError(t, err)
		if rl.FunctionConfig.GetLabel("k") != "v" {
			t.Errorf("Nil KubeObject cannot call SetLabel()")
		}
		if rl.FunctionConfig.IsEmpty() {
			t.Errorf("The modified fnConfig is not nil.")
		}
	}
}

var deploymentResourceList = []byte(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: nginx-deployment
    labels:
      app: nginx
    annotations:
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'resources.yaml'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'resources.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 3
    selector:
      matchLabels:
        app: nginx
    paused: true
    strategy:
      type: Recreate
    template:
      metadata:
        labels:
          app: nginx
      spec:
        nodeSelector:
          disktype: ssd
        containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
          - containerPort: 80
    fakeStringSlice:
    - test1
    - test2
`)

const StrategyTypeRecreate = "Recreate"

func TestGetNestedFields(t *testing.T) {
	rl, _ := ParseResourceList(deploymentResourceList)
	deployment := rl.Items[0]
	// Style 1, using concatenated fields in  NestedType function.
	if intVal, _, _ := deployment.NestedInt64("spec", "replicas"); intVal != 3 {
		t.Errorf("deployment .spec.replicas expected to be 3, got %v", intVal)
	}
	if boolVal, _, _ := deployment.NestedBool("spec", "paused"); boolVal != true {
		t.Errorf("deployment .spec.paused expected to be true, got %v", boolVal)
	}
	if stringVal, _, _ := deployment.NestedString("spec", "strategy", "type"); stringVal != StrategyTypeRecreate {
		t.Errorf("deployment .spec.strategy.type expected to be `Recreate`, got %v", stringVal)
	}
	if stringMapVal, _, _ := deployment.NestedStringMap("spec", "template", "spec", "nodeSelector"); !reflect.DeepEqual(stringMapVal, map[string]string{"disktype": "ssd"}) {
		t.Errorf("deployment .spec.template.spec.nodeSelector expected to get {`disktype`: `ssd`}, got %v", stringMapVal)
	}
	if sliceVal, _, _ := deployment.NestedSlice("spec", "template", "spec", "containers"); sliceVal[0].GetString("name") != "nginx" {
		t.Errorf("deployment .spec.template.spec.containers[0].name expected to get `nginx`, got %v", sliceVal[0].GetString("name"))
	}
	if stringSliceVal, _, _ := deployment.NestedStringSlice("spec", "fakeStringSlice"); !reflect.DeepEqual(stringSliceVal, []string{"test1", "test2"}) {
		t.Errorf("deployment .spec.fakeStringSlice expected to get [`test1`, `test2`], got %v", stringSliceVal)
	}
	// Style 2, get each struct layer by type.
	spec := deployment.GetMap("spec")
	if intVal := spec.GetInt("replicas"); intVal != 3 {
		t.Errorf("deployment .spec.replicas expected to be 3, got %v", intVal)
	}

	strategy, _, _ := deployment.NestedSubObject("spec", "strategy")
	if stringVal := strategy.GetString("type"); stringVal != StrategyTypeRecreate {
		t.Errorf("deployment .spec.strategy.type expected to be `Recreate`, got %v", stringVal)
	}
	if boolVal := spec.GetBool("paused"); boolVal != true {
		t.Errorf("deployment .spec.paused expected to be true, got %v", boolVal)
	}
	if stringVal := spec.GetMap("strategy").GetString("type"); stringVal != StrategyTypeRecreate {
		t.Errorf("deployment .spec.strategy.type expected to be `Recreate`, got %v", stringVal)
	}
	tmplSpec := spec.GetMap("template").GetMap("spec")
	if stringMapVal := tmplSpec.GetMap("nodeSelector"); stringMapVal.GetString("disktype") != "ssd" {
		t.Errorf("deployment .spec.template.spec.nodeSelector expected to get {`disktype`: `ssd`}, got %v", stringMapVal.GetString("disktype"))
	}
	if sliceVal := tmplSpec.GetSlice("containers"); sliceVal[0].GetString("name") != "nginx" {
		t.Errorf("deployment .spec.template.spec.containers[0].name expected to get `nginx`, got %v", sliceVal[0].GetString("name"))
	}
}

func TestSetNestedFields(t *testing.T) {
	var err error
	o := NewEmptyKubeObject()
	err = o.SetNestedString("some starlark script...", "source")
	assert.NoError(t, err)
	if stringVal, _, _ := o.NestedString("source"); stringVal != "some starlark script..." {
		t.Errorf("KubeObject .source expected to get \"some starlark script...\", got %v", stringVal)
	}
	err = o.SetNestedStringMap(map[string]string{"tag1": "abc", "tag2": "test1"}, "tags")
	assert.NoError(t, err)
	if stringMapVal, _, _ := o.NestedString("tags", "tag2"); stringMapVal != "test1" {
		t.Errorf("KubeObject .tags.tag2 expected to get `test1`, got %v", stringMapVal)
	}
	err = o.SetNestedStringSlice([]string{"label1", "label2"}, "labels")
	assert.NoError(t, err)
	if stringSliceVal, _, _ := o.NestedStringSlice("labels"); !reflect.DeepEqual(stringSliceVal, []string{"label1", "label2"}) {
		t.Errorf("KubeObject .labels expected to get [`label1`, `label2`], got %v", stringSliceVal)
	}
}

func TestInternalAnnotationsUntouchable(t *testing.T) {
	var err error
	o := NewEmptyKubeObject()
	// Verify the "upstream-identifier" annotation cannot be changed via SetNestedStringMap
	err = o.SetNestedStringMap(map[string]string{"owner": "kpt"}, "metadata", "annotations")
	assert.NoError(t, err)
	if stringMapVal, _, _ := o.NestedStringMap("metadata", "annotations"); !reflect.DeepEqual(stringMapVal, map[string]string{"owner": "kpt"}) {
		t.Errorf("annotations cannot be set via SetNestedStringMap, got %v", stringMapVal)
	}
	err = o.SetNestedStringMap(map[string]string{UpstreamIdentifier: "apps|Deployment|default|dp"}, "metadata", "annotations")
	if !errors.Is(ErrAttemptToTouchUpstreamIdentifier{}, err) {
		t.Errorf("set internal annotation via SetNestedStringMap() failed, expect %e, got %e", ErrAttemptToTouchUpstreamIdentifier{}, err)
	}

	// Verify the "upstream-identifier" annotation cannot be changed via SetAnnotation
	err = o.SetAnnotation("owner", "kpt")
	assert.NoError(t, err)
	if o.GetAnnotation("owner") != "kpt" {
		t.Errorf("annotations cannot be set via SetAnnotation(), got %v", o.GetAnnotation("owner"))
	}
	if err = o.SetAnnotation(UpstreamIdentifier, "apps|Deployment|default|dp"); err == nil {
		t.Errorf("set internal annotation via SetAnnotation() expect panic (%v), got pass",
			ErrAttemptToTouchUpstreamIdentifier{})
	}

	// Verify the "upstream-identifier" annotation cannot be changed via SetNestedField
	err = o.SetNestedField(map[string]string{UpstreamIdentifier: "apps|Deployment|default|dp"}, "metadata", "annotations")
	if !errors.Is(ErrAttemptToTouchUpstreamIdentifier{}, err) {
		t.Errorf("set internal annotation via SetNestedField() failed, expect %e, got %e", ErrAttemptToTouchUpstreamIdentifier{}, err)
	}
}

func generate(t *testing.T) *KubeObject {
	doc := `
apiVersion: v1
kind: ConfigMap
data:
  foo: bar
  foo2: bar2
`

	o, err := ParseKubeObject([]byte(doc))
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}

	return o
}

func TestUpsertMap(t *testing.T) {
	o := generate(t)
	data := o.UpsertMap("data")

	entries, err := data.obj.Entries()
	if err != nil {
		t.Fatalf("entries failed: %v", err)
	}
	var got []string
	for k := range entries {
		got = append(got, k)
	}
	sort.Strings(got)

	want := []string{"foo", "foo2"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected expect (-want, +got): %s", diff)
	}
}

func TestGetMap(t *testing.T) {
	o := generate(t)
	got := o.GetMap("data")
	if got == nil {
		t.Errorf("unexpected value for GetMap(%q); got %v, want non-nil", "data", got)
	}
	got = o.GetMap("notExists")
	if got != nil {
		t.Errorf("unexpected value for GetMap(%q); got %v, want nil", "notExists", got)
	}
}

func TestSelectors(t *testing.T) {
	deployment := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo
`
	service := `
apiVersion: apps/v1
kind: Service
metadata:
  name: example
  namespace: my-namespace
  labels:
    foo: baz
  annotations:
    foo: bar
`
	kptfile := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example-2
  annotations:
    bar: foo
`
	d, err := ParseKubeObject([]byte(deployment))
	assert.NoError(t, err)
	s, err := ParseKubeObject([]byte(service))
	assert.NoError(t, err)
	k, err := ParseKubeObject([]byte(kptfile))
	assert.NoError(t, err)
	input := KubeObjects{d, s, k}

	// select all resources with labels foo=baz
	items := input.Where(HasLabels(map[string]string{"foo": "baz"}))
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo
---
apiVersion: apps/v1
kind: Service
metadata:
  name: example
  namespace: my-namespace
  labels:
    foo: baz
  annotations:
    foo: bar`)

	// select all deployments
	items = input.Where(IsGVK("apps", "v1", "Deployment"))
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo`)

	// exclude all services and meta resources
	items = input.WhereNot(IsGVK("apps", "v1", "Service")).WhereNot(IsMetaResource())
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo`)

	// include resources with the label abc: def
	items = input.Where(HasLabels(map[string]string{"abc": "def"}))
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo`)

	// exclude all resources with the annotation foo=bar
	items = input.WhereNot(HasAnnotations(map[string]string{"foo": "bar"}))
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
  annotations:
    bar: foo
---
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example-2
  annotations:
    bar: foo`)

	// include resources named 'example' that are Not in namespace default
	items = input.Where(IsName("example")).WhereNot(IsNamespace("default"))
	assert.Equal(t, items.String(), `apiVersion: apps/v1
kind: Service
metadata:
  name: example
  namespace: my-namespace
  labels:
    foo: baz
  annotations:
    foo: bar`)

	// add the label "g=h" to all resources with annotation "bar=foo"
	for _, obj := range input.Where(HasAnnotations(map[string]string{"bar": "foo"})) {
		err = obj.SetLabel("g", "h")
		assert.NoError(t, err)
	}
	assert.Equal(t, input.String(), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
  namespace: default
  labels:
    abc: def
    foo: baz
    g: h
  annotations:
    bar: foo
---
apiVersion: apps/v1
kind: Service
metadata:
  name: example
  namespace: my-namespace
  labels:
    foo: baz
  annotations:
    foo: bar
---
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example-2
  annotations:
    bar: foo
  labels:
    g: h`)
}

func TestGetRootKptfile(t *testing.T) {
	nestedPkgResourceList := []byte(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: ghost-root
    annotations:
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/seqindent: 'wide'
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: ghost-app
    labels:
      app.kubernetes.io/name: ghost-app
    annotations:
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'ghost-app/Kptfile'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'ghost-app/Kptfile'
      internal.config.kubernetes.io/seqindent: 'wide'
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: mariadb
    labels:
      app.kubernetes.io/name: mariadb
    annotations:
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'mariadb/Kptfile'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'mariadb/Kptfile'
      internal.config.kubernetes.io/seqindent: 'wide'
`)
	rl, _ := ParseResourceList(nestedPkgResourceList)
	kptfile := rl.Items.GetRootKptfile()
	assert.NotEmpty(t, kptfile)
	assert.Equal(t, "ghost-root", kptfile.GetName())
}

func TestEmptyGetRootKptfile(t *testing.T) {
	noKptfileInResourceList := []byte(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: v1
  kind: ConfigMap
  metadata: # kpt-merge: /kptfile.kpt.dev
    name: kptfile.kpt.dev
    annotations:
      config.kubernetes.io/local-config: "true"
      internal.kpt.dev/upstream-identifier: '|ConfigMap|default|kptfile.kpt.dev'
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'package-context.yaml'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'package-context.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  data:
    name: ghost21`)
	rl, _ := ParseResourceList(noKptfileInResourceList)
	kptfile := rl.Items.GetRootKptfile()
	assert.Empty(t, kptfile)
}

func TestGroupVersionKind(t *testing.T) {
	input := []byte(`
apiVersion: apps/v2
kind: StatefulSet
metadata:
  name: my-config
`)

	resource, _ := ParseKubeObject(input)
	got := resource.GroupVersionKind()
	want := schema.GroupVersionKind{Group: "apps", Version: "v2", Kind: "StatefulSet"}
	if got != want {
		t.Fatalf("unexpected result from GroupVersionKind(); got %v; want %v", got, want)
	}
}

func TestRNodeInteroperability(t *testing.T) {
	input := []byte(`
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-app
  namespace: my-ns
`)

	var found bool
	ko, err := ParseKubeObject(input)
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}

	// test copy to and from ResourceNode
	rn := ko.CopyToResourceNode()
	assert.Equal(t, "apps/v1", ko.GetAPIVersion())
	assert.Equal(t, "StatefulSet", ko.GetKind())
	assert.Equal(t, "my-app", ko.GetName())
	assert.Equal(t, "my-ns", ko.GetNamespace())
	assert.Equal(t, "apps/v1", rn.GetApiVersion())
	assert.Equal(t, "StatefulSet", rn.GetKind())
	assert.Equal(t, "my-app", rn.GetName())
	assert.Equal(t, "my-ns", rn.GetNamespace())

	ko2 := NewKubeObjectFromResourceNode(rn)
	assert.Equal(t, "apps/v1", rn.GetApiVersion())
	assert.Equal(t, "StatefulSet", rn.GetKind())
	assert.Equal(t, "my-app", rn.GetName())
	assert.Equal(t, "my-ns", rn.GetNamespace())
	assert.Equal(t, "apps/v1", ko2.GetAPIVersion())
	assert.Equal(t, "StatefulSet", ko2.GetKind())
	assert.Equal(t, "my-app", ko2.GetName())
	assert.Equal(t, "my-ns", ko2.GetNamespace())

	// test move to and from ResourceNode
	rn2 := ko2.MoveToResourceNode()
	_, found, _ = ko2.NestedString("apiVersion")
	assert.False(t, found)
	_, found, _ = ko2.NestedString("kind")
	assert.False(t, found)
	_, found, _ = ko2.NestedString("metadata", "name")
	assert.False(t, found)
	_, found, _ = ko2.NestedString("metadata", "namespace")
	assert.False(t, found)
	assert.Equal(t, "apps/v1", rn2.GetApiVersion())
	assert.Equal(t, "StatefulSet", rn2.GetKind())
	assert.Equal(t, "my-app", rn2.GetName())
	assert.Equal(t, "my-ns", rn2.GetNamespace())

	ko3 := MoveToKubeObject(rn2)
	assert.Equal(t, "apps/v1", ko3.GetAPIVersion())
	assert.Equal(t, "StatefulSet", ko3.GetKind())
	assert.Equal(t, "my-app", ko3.GetName())
	assert.Equal(t, "my-ns", ko3.GetNamespace())
	assert.Empty(t, rn2.GetApiVersion())
	assert.Empty(t, rn2.GetKind())
	assert.Empty(t, rn2.GetName())
	assert.Empty(t, rn2.GetNamespace())
}

func TestDeepCopy(t *testing.T) {
	input := []byte(`
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-app
  namespace: my-ns
`)

	orig, err := ParseKubeObject(input)
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}

	copy := orig.Copy()
	assert.Equal(t, orig.String(), copy.String())
	assert.NoError(t, copy.SetName("new-name"))
	assert.Equal(t, "my-app", orig.GetName())
	assert.Equal(t, "new-name", copy.GetName())
}

func TestUpsert(t *testing.T) {
	resources := []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configPath: fn-config.yaml
---
apiVersion: v1
kind: Service
metadata:
 name: whatever
 labels:
   app: myApp
`)

	toUpdate := []byte(`
apiVersion: v1
kind: Service
metadata:
 name: whatever
 labels:
   app: notMyApp
`)

	toInsert := []byte(`
apiVersion: v1
kind: Service
metadata:
 name: new
 labels:
   app: notMyApp
`)

	var objs KubeObjects
	objs, err := ParseKubeObjects(resources)
	if err != nil {
		t.Fatalf("failed to parse objects: %v", err)
	}

	toUpdateObj, err := ParseKubeObject(toUpdate)
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}
	objs.Upsert(toUpdateObj)
	assert.Equal(t, len(objs), 2)
	assert.Equal(t, "whatever", objs[1].GetMap("metadata").GetString("name"))
	assert.Equal(t, "notMyApp", objs[1].GetMap("metadata").GetMap("labels").GetString("app"))

	toInsertObj, err := ParseKubeObject(toInsert)
	if err != nil {
		t.Fatalf("failed to parse object: %v", err)
	}
	objs.Upsert(toInsertObj)
	assert.Equal(t, len(objs), 3)
	assert.Equal(t, "new", objs[2].GetMap("metadata").GetString("name"))
	assert.Equal(t, "notMyApp", objs[2].GetMap("metadata").GetMap("labels").GetString("app"))
}
