// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package apply

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/klog/v2"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/jsonpath"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

type inventoryInfo struct {
	name      string
	namespace string
	id        string
	set       object.ObjMetadataSet
}

func (i inventoryInfo) toUnstructured() *unstructured.Unstructured {
	invMap := make(map[string]interface{})
	for _, objMeta := range i.set {
		invMap[objMeta.String()] = ""
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      i.name,
				"namespace": i.namespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: i.id,
				},
			},
			"data": invMap,
		},
	}
}

func (i inventoryInfo) toWrapped() inventory.Info {
	return inventory.WrapInventoryInfoObj(i.toUnstructured())
}

func newTestApplier(
	t *testing.T,
	invInfo inventoryInfo,
	resources object.UnstructuredSet,
	clusterObjs object.UnstructuredSet,
	statusPoller poller.Poller,
) *Applier {
	tf := newTestFactory(t, invInfo, resources, clusterObjs)
	defer tf.Cleanup()

	infoHelper := &fakeInfoHelper{
		factory: tf,
	}

	invClient := newTestInventory(t, tf)

	applier, err := NewApplierBuilder().
		WithFactory(tf).
		WithInventoryClient(invClient).
		WithStatusPoller(statusPoller).
		Build()
	require.NoError(t, err)

	// Inject the fakeInfoHelper to allow generating Info
	// objects that use the FakeRESTClient as the UnstructuredClient.
	applier.infoHelper = infoHelper

	return applier
}

func newTestDestroyer(
	t *testing.T,
	invInfo inventoryInfo,
	clusterObjs object.UnstructuredSet,
	statusPoller poller.Poller,
) *Destroyer {
	tf := newTestFactory(t, invInfo, object.UnstructuredSet{}, clusterObjs)
	defer tf.Cleanup()

	invClient := newTestInventory(t, tf)

	destroyer, err := NewDestroyer(tf, invClient)
	require.NoError(t, err)
	destroyer.StatusPoller = statusPoller

	return destroyer
}

func newTestInventory(
	t *testing.T,
	tf *cmdtesting.TestFactory,
) inventory.Client {
	// Use an Client with a fakeInfoHelper to allow generating Info
	// objects that use the FakeRESTClient as the UnstructuredClient.
	invClient, err := inventory.ClusterClientFactory{}.NewClient(tf)
	require.NoError(t, err)
	return invClient
}

func newTestFactory(
	t *testing.T,
	invInfo inventoryInfo,
	resourceSet object.UnstructuredSet,
	clusterObjs object.UnstructuredSet,
) *cmdtesting.TestFactory {
	tf := cmdtesting.NewTestFactory().WithNamespace(invInfo.namespace)

	mapper, err := tf.ToRESTMapper()
	require.NoError(t, err)

	objMap := make(map[object.ObjMetadata]resourceInfo)
	for _, r := range resourceSet {
		objMeta := object.UnstructuredToObjMetadata(r)
		objMap[objMeta] = resourceInfo{
			resource: r,
			exists:   false,
		}
	}
	for _, r := range clusterObjs {
		objMeta := object.UnstructuredToObjMetadata(r)
		objMap[objMeta] = resourceInfo{
			resource: r,
			exists:   true,
		}
	}
	var objs []resourceInfo
	for _, obj := range objMap {
		objs = append(objs, obj)
	}

	handlers := []handler{
		&nsHandler{},
		&genericHandler{
			resources: objs,
			mapper:    mapper,
		},
	}

	tf.UnstructuredClient = newFakeRESTClient(t, handlers)
	tf.FakeDynamicClient = fakeDynamicClient(t, mapper, invInfo, objs...)

	return tf
}

type resourceInfo struct {
	resource *unstructured.Unstructured
	exists   bool
}

// newFakeRESTClient creates a new client that uses a set of handlers to
// determine how to handle requests. For every request it will iterate through
// the handlers until it can find one that knows how to handle the request.
// This is to keep the main structure of the fake client manageable while still
// allowing different behavior for different testcases.
func newFakeRESTClient(t *testing.T, handlers []handler) *fake.RESTClient {
	return &fake.RESTClient{
		NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			klog.V(5).Infof("FakeRESTClient: handling %s request for %q", req.Method, req.URL)
			for _, h := range handlers {
				resp, handled, err := h.handle(t, req)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
					return nil, nil
				}
				if handled {
					return resp, nil
				}
			}
			t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
			return nil, nil
		}),
	}
}

// The handler interface allows different testcases to provide
// special handling of requests. It also allows a single handler
// to keep state between a set of related requests instead of keeping
// a single large event handler.
type handler interface {
	handle(t *testing.T, req *http.Request) (*http.Response, bool, error)
}

// genericHandler provides a simple handler for resources that can
// be fetched and updated. It will simply return the given resource
// when asked for and accept patch requests.
type genericHandler struct {
	resources []resourceInfo
	mapper    meta.RESTMapper
}

func (g *genericHandler) handle(t *testing.T, req *http.Request) (*http.Response, bool, error) {
	klog.V(5).Infof("genericHandler: handling %s request for %q", req.Method, req.URL)
	for _, r := range g.resources {
		gvk := r.resource.GroupVersionKind()
		mapping, err := g.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, false, err
		}
		var allPath string
		if mapping.Scope == meta.RESTScopeNamespace {
			allPath = fmt.Sprintf("/namespaces/%s/%s", r.resource.GetNamespace(), mapping.Resource.Resource)
		} else {
			allPath = fmt.Sprintf("/%s", mapping.Resource.Resource)
		}
		singlePath := allPath + "/" + r.resource.GetName()

		if req.URL.Path == singlePath && req.Method == http.MethodGet {
			if r.exists {
				bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, r.resource)))
				return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Header: cmdtesting.DefaultHeader(), Body: cmdtesting.StringBody("")}, true, nil
		}

		if req.URL.Path == singlePath && req.Method == http.MethodPatch {
			bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, r.resource)))
			return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
		}

		if req.URL.Path == singlePath && req.Method == http.MethodDelete {
			if r.exists {
				bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, r.resource)))
				return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
			}

			// We're not testing DeletePropagationOrphan, so StatusOK should be
			// safe. Otherwise, the status might be StatusAccepted.
			// https://github.com/kubernetes/apiserver/blob/v0.22.2/pkg/endpoints/handlers/delete.go#L140
			status := http.StatusOK

			// Return Status object, if resource doesn't exist.
			result := &metav1.Status{
				Status: metav1.StatusSuccess,
				Code:   int32(status),
				Details: &metav1.StatusDetails{
					Name: r.resource.GetName(),
					Kind: r.resource.GetKind(),
				},
			}
			bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, result)))
			return &http.Response{StatusCode: status, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
		}

		if req.URL.Path == allPath && req.Method == http.MethodPost {
			bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, r.resource)))
			return &http.Response{StatusCode: http.StatusCreated, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
		}
	}
	return nil, false, nil
}

func newInventoryReactor(invInfo inventoryInfo) *inventoryReactor {
	return &inventoryReactor{
		inventoryObj: invInfo.toUnstructured(),
	}
}

type inventoryReactor struct {
	inventoryObj *unstructured.Unstructured
}

func (ir *inventoryReactor) updateFakeDynamicClient(fdc *dynamicfake.FakeDynamicClient) {
	fdc.PrependReactor("create", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
		obj := *action.(clienttesting.CreateAction).GetObject().(*unstructured.Unstructured)
		ir.inventoryObj = &obj
		return true, ir.inventoryObj.DeepCopy(), nil
	})
	fdc.PrependReactor("list", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
		uList := &unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{},
		}
		if ir.inventoryObj != nil {
			uList.Items = append(uList.Items, *ir.inventoryObj.DeepCopy())
		}
		return true, uList, nil
	})
	fdc.PrependReactor("get", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, ir.inventoryObj.DeepCopy(), nil
	})
	fdc.PrependReactor("update", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
		obj := *action.(clienttesting.UpdateAction).GetObject().(*unstructured.Unstructured)
		ir.inventoryObj = &obj
		return true, ir.inventoryObj.DeepCopy(), nil
	})
}

// nsHandler can handle requests for a namespace. It will behave as if
// every requested namespace exists. It simply fetches the name of the requested
// namespace from the url and creates a new namespace type with the provided
// name for the response.
type nsHandler struct{}

var (
	nsPathRegex = regexp.MustCompile(`/api/v1/namespaces/([^/]+)`)
)

func (n *nsHandler) handle(t *testing.T, req *http.Request) (*http.Response, bool, error) {
	match := nsPathRegex.FindStringSubmatch(req.URL.Path)
	if req.Method == http.MethodGet && match != nil {
		nsName := match[1]
		ns := v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		bodyRC := ioutil.NopCloser(bytes.NewReader(toJSONBytes(t, &ns)))
		return &http.Response{StatusCode: http.StatusOK, Header: cmdtesting.DefaultHeader(), Body: bodyRC}, true, nil
	}
	return nil, false, nil
}

type fakePoller struct {
	start  chan struct{}
	events []pollevent.Event
}

func newFakePoller(statusEvents []pollevent.Event) *fakePoller {
	return &fakePoller{
		events: statusEvents,
		start:  make(chan struct{}),
	}
}

// Start events being sent on the status channel
func (f *fakePoller) Start() {
	close(f.start)
}

func (f *fakePoller) Poll(ctx context.Context, _ object.ObjMetadataSet, _ polling.PollOptions) <-chan pollevent.Event {
	eventChannel := make(chan pollevent.Event)
	go func() {
		defer close(eventChannel)
		// wait until started to send the events
		<-f.start
		for _, f := range f.events {
			eventChannel <- f
		}
		// wait until cancelled to close the event channel and exit
		<-ctx.Done()
	}()
	return eventChannel
}

type fakeInfoHelper struct {
	factory *cmdtesting.TestFactory
}

// TODO(mortent): This has too much code in common with the
// infoHelper implementation. We need to find a better way to structure
// this.
func (f *fakeInfoHelper) UpdateInfo(info *resource.Info) error {
	mapper, err := f.factory.ToRESTMapper()
	if err != nil {
		return err
	}
	gvk := info.Object.GetObjectKind().GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}
	info.Mapping = mapping

	c, err := f.getClient(gvk.GroupVersion())
	if err != nil {
		return err
	}
	info.Client = c
	return nil
}

func (f *fakeInfoHelper) BuildInfo(obj *unstructured.Unstructured) (*resource.Info, error) {
	info := &resource.Info{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Source:    "unstructured",
		Object:    obj,
	}
	err := f.UpdateInfo(info)
	return info, err
}

func (f *fakeInfoHelper) getClient(gv schema.GroupVersion) (resource.RESTClient, error) {
	if f.factory.UnstructuredClientForMappingFunc != nil {
		return f.factory.UnstructuredClientForMappingFunc(gv)
	}
	if f.factory.UnstructuredClient != nil {
		return f.factory.UnstructuredClient, nil
	}
	return f.factory.Client, nil
}

// fakeDynamicClient returns a fake dynamic client.
func fakeDynamicClient(t *testing.T, mapper meta.RESTMapper, invInfo inventoryInfo, objs ...resourceInfo) *dynamicfake.FakeDynamicClient {
	fakeClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme)

	invReactor := newInventoryReactor(invInfo)
	invReactor.updateFakeDynamicClient(fakeClient)

	for i := range objs {
		obj := objs[i]
		gvk := obj.resource.GroupVersionKind()
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		r := mapping.Resource.Resource
		fakeClient.PrependReactor("get", r, func(clienttesting.Action) (bool, runtime.Object, error) {
			if obj.exists {
				return true, obj.resource, nil
			}
			return false, nil, nil
		})
		fakeClient.PrependReactor("delete", r, func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, nil
		})
	}

	return fakeClient
}

func toJSONBytes(t *testing.T, obj runtime.Object) []byte {
	objBytes, err := runtime.Encode(unstructured.NewJSONFallbackEncoder(codec), obj)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	return objBytes
}

type JSONPathSetter struct {
	Path  string
	Value interface{}
}

func (jps JSONPathSetter) Mutate(u *unstructured.Unstructured) {
	_, err := jsonpath.Set(u.Object, jps.Path, jps.Value)
	if err != nil {
		panic(fmt.Sprintf("failed to mutate unstructured object: %v", err))
	}
}
