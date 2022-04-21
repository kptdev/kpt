// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/cli-utils/pkg/apply/cache"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/testutil"
)

type resourceInfo struct {
	group      string
	apiVersion string
	kind       string
	name       string
	namespace  string
	uid        types.UID
	generation int64
}

// Tests that the correct "applied" objects are sent
// to the TaskContext correctly, since these are the
// applied objects added to the final inventory.
func TestApplyTask_BasicAppliedObjects(t *testing.T) {
	testCases := map[string]struct {
		applied []resourceInfo
	}{
		"apply single namespaced resource": {
			applied: []resourceInfo{
				{
					group:      "apps",
					apiVersion: "apps/v1",
					kind:       "Deployment",
					name:       "foo",
					namespace:  "default",
					uid:        types.UID("my-uid"),
					generation: int64(42),
				},
			},
		},
		"apply multiple clusterscoped resources": {
			applied: []resourceInfo{
				{
					group:      "custom.io",
					apiVersion: "custom.io/v1beta1",
					kind:       "Custom",
					name:       "bar",
					uid:        types.UID("uid-1"),
					generation: int64(32),
				},
				{
					group:      "custom2.io",
					apiVersion: "custom2.io/v1",
					kind:       "Custom2",
					name:       "foo",
					uid:        types.UID("uid-2"),
					generation: int64(1),
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			eventChannel := make(chan event.Event)
			defer close(eventChannel)
			resourceCache := cache.NewResourceCacheMap()
			taskContext := taskrunner.NewTaskContext(eventChannel, resourceCache)

			objs := toUnstructureds(tc.applied)

			oldAO := applyOptionsFactoryFunc
			applyOptionsFactoryFunc = func(string, chan<- event.Event, common.ServerSideOptions, common.DryRunStrategy,
				dynamic.Interface, discovery.OpenAPISchemaInterface) applyOptions {
				return &fakeApplyOptions{}
			}
			defer func() { applyOptionsFactoryFunc = oldAO }()

			restMapper := testutil.NewFakeRESTMapper(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}, schema.GroupVersionKind{
				Group:   "anothercustom.io",
				Version: "v2",
				Kind:    "AnotherCustom",
			})

			applyTask := &ApplyTask{
				Objects:    objs,
				Mapper:     restMapper,
				InfoHelper: &fakeInfoHelper{},
			}

			applyTask.Start(taskContext)
			<-taskContext.TaskChannel()

			// The applied resources should be stored in the TaskContext
			// for the final inventory.
			expectedIDs := object.UnstructuredSetToObjMetadataSet(objs)
			actual := taskContext.InventoryManager().SuccessfulApplies()
			if !actual.Equal(expectedIDs) {
				t.Errorf("expected (%s) inventory resources, got (%s)", expectedIDs, actual)
			}

			im := taskContext.InventoryManager()

			for _, id := range expectedIDs {
				assert.Falsef(t, im.IsFailedApply(id), "ApplyTask should NOT mark object as failed: %s", id)
				assert.Falsef(t, im.IsSkippedApply(id), "ApplyTask should NOT mark object as skipped: %s", id)
			}
		})
	}
}

func TestApplyTask_FetchGeneration(t *testing.T) {
	testCases := map[string]struct {
		rss []resourceInfo
	}{
		"single namespaced resource": {
			rss: []resourceInfo{
				{
					group:      "apps",
					apiVersion: "apps/v1",
					kind:       "Deployment",
					name:       "foo",
					namespace:  "default",
					uid:        types.UID("my-uid"),
					generation: int64(42),
				},
			},
		},
		"multiple clusterscoped resources": {
			rss: []resourceInfo{
				{
					group:      "custom.io",
					apiVersion: "custom.io/v1beta1",
					kind:       "Custom",
					name:       "bar",
					uid:        types.UID("uid-1"),
					generation: int64(32),
				},
				{
					group:      "custom2.io",
					apiVersion: "custom2.io/v1",
					kind:       "Custom2",
					name:       "foo",
					uid:        types.UID("uid-2"),
					generation: int64(1),
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			eventChannel := make(chan event.Event)
			defer close(eventChannel)
			resourceCache := cache.NewResourceCacheMap()
			taskContext := taskrunner.NewTaskContext(eventChannel, resourceCache)

			objs := toUnstructureds(tc.rss)

			oldAO := applyOptionsFactoryFunc
			applyOptionsFactoryFunc = func(string, chan<- event.Event, common.ServerSideOptions, common.DryRunStrategy,
				dynamic.Interface, discovery.OpenAPISchemaInterface) applyOptions {
				return &fakeApplyOptions{}
			}
			defer func() { applyOptionsFactoryFunc = oldAO }()
			applyTask := &ApplyTask{
				Objects:    objs,
				InfoHelper: &fakeInfoHelper{},
			}
			applyTask.Start(taskContext)

			<-taskContext.TaskChannel()

			for _, info := range tc.rss {
				id := object.ObjMetadata{
					GroupKind: schema.GroupKind{
						Group: info.group,
						Kind:  info.kind,
					},
					Name:      info.name,
					Namespace: info.namespace,
				}
				uid, _ := taskContext.InventoryManager().AppliedResourceUID(id)
				assert.Equal(t, info.uid, uid)

				gen, _ := taskContext.InventoryManager().AppliedGeneration(id)
				assert.Equal(t, info.generation, gen)
			}
		})
	}
}

func TestApplyTask_DryRun(t *testing.T) {
	testCases := map[string]struct {
		objs            []*unstructured.Unstructured
		expectedObjects []object.ObjMetadata
		expectedEvents  []event.Event
	}{
		"simple dry run": {
			objs: []*unstructured.Unstructured{
				toUnstructured(map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "foo",
						"namespace": "default",
					},
				}),
			},
			expectedObjects: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "foo",
					Namespace: "default",
				},
			},
			expectedEvents: []event.Event{},
		},
		"dry run with CRD and CR": {
			objs: []*unstructured.Unstructured{
				toUnstructured(map[string]interface{}{
					"apiVersion": "apiextensions.k8s.io/v1",
					"kind":       "CustomResourceDefinition",
					"metadata": map[string]interface{}{
						"name": "foo",
					},
					"spec": map[string]interface{}{
						"group": "custom.io",
						"names": map[string]interface{}{
							"kind": "Custom",
						},
						"versions": []interface{}{
							map[string]interface{}{
								"name": "v1alpha1",
							},
						},
					},
				}),
				toUnstructured(map[string]interface{}{
					"apiVersion": "custom.io/v1alpha1",
					"kind":       "Custom",
					"metadata": map[string]interface{}{
						"name": "bar",
					},
				}),
			},
			expectedObjects: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "custom.io",
						Kind:  "Custom",
					},
					Name: "bar",
				},
			},
			expectedEvents: []event.Event{},
		},
	}

	for tn, tc := range testCases {
		for i := range common.Strategies {
			drs := common.Strategies[i]
			t.Run(tn, func(t *testing.T) {
				eventChannel := make(chan event.Event)
				resourceCache := cache.NewResourceCacheMap()
				taskContext := taskrunner.NewTaskContext(eventChannel, resourceCache)

				restMapper := testutil.NewFakeRESTMapper(schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				}, schema.GroupVersionKind{
					Group:   "anothercustom.io",
					Version: "v2",
					Kind:    "AnotherCustom",
				})

				ao := &fakeApplyOptions{}
				oldAO := applyOptionsFactoryFunc
				applyOptionsFactoryFunc = func(string, chan<- event.Event, common.ServerSideOptions, common.DryRunStrategy,
					dynamic.Interface, discovery.OpenAPISchemaInterface) applyOptions {
					return ao
				}
				defer func() { applyOptionsFactoryFunc = oldAO }()

				applyTask := &ApplyTask{
					Objects:        tc.objs,
					InfoHelper:     &fakeInfoHelper{},
					Mapper:         restMapper,
					DryRunStrategy: drs,
				}

				var events []event.Event
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					for msg := range eventChannel {
						events = append(events, msg)
					}
				}()

				applyTask.Start(taskContext)
				<-taskContext.TaskChannel()
				close(eventChannel)
				wg.Wait()

				assert.Equal(t, len(tc.expectedObjects), len(ao.objects))
				for i, obj := range ao.objects {
					actual, err := object.InfoToObjMeta(obj)
					if err != nil {
						continue
					}
					assert.Equal(t, tc.expectedObjects[i], actual)
				}

				assert.Equal(t, len(tc.expectedEvents), len(events))
				for i, e := range events {
					assert.Equal(t, tc.expectedEvents[i].Type, e.Type)
				}
			})
		}
	}
}

func TestApplyTaskWithError(t *testing.T) {
	testCases := map[string]struct {
		objs            []*unstructured.Unstructured
		expectedObjects object.ObjMetadataSet
		expectedEvents  []event.Event
		expectedSkipped object.ObjMetadataSet
		expectedFailed  object.ObjMetadataSet
	}{
		"some resources have apply error": {
			objs: []*unstructured.Unstructured{
				toUnstructured(map[string]interface{}{
					"apiVersion": "apiextensions.k8s.io/v1",
					"kind":       "CustomResourceDefinition",
					"metadata": map[string]interface{}{
						"name": "foo",
					},
					"spec": map[string]interface{}{
						"group": "anothercustom.io",
						"names": map[string]interface{}{
							"kind": "AnotherCustom",
						},
						"versions": []interface{}{
							map[string]interface{}{
								"name": "v2",
							},
						},
					},
				}),
				toUnstructured(map[string]interface{}{
					"apiVersion": "anothercustom.io/v2",
					"kind":       "AnotherCustom",
					"metadata": map[string]interface{}{
						"name":      "bar",
						"namespace": "barbar",
					},
				}),
				toUnstructured(map[string]interface{}{
					"apiVersion": "anothercustom.io/v2",
					"kind":       "AnotherCustom",
					"metadata": map[string]interface{}{
						"name":      "bar-with-failure",
						"namespace": "barbar",
					},
				}),
			},
			expectedObjects: object.ObjMetadataSet{
				{
					GroupKind: schema.GroupKind{
						Group: "apiextensions.k8s.io",
						Kind:  "CustomResourceDefinition",
					},
					Name: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "anothercustom.io",
						Kind:  "AnotherCustom",
					},
					Name:      "bar",
					Namespace: "barbar",
				},
			},
			expectedEvents: []event.Event{
				{
					Type: event.ApplyType,
					ApplyEvent: event.ApplyEvent{
						Error: fmt.Errorf("expected apply error"),
					},
				},
			},
			expectedFailed: object.ObjMetadataSet{
				{
					GroupKind: schema.GroupKind{
						Group: "anothercustom.io",
						Kind:  "AnotherCustom",
					},
					Name:      "bar-with-failure",
					Namespace: "barbar",
				},
			},
		},
	}

	for tn, tc := range testCases {
		drs := common.DryRunNone
		t.Run(tn, func(t *testing.T) {
			eventChannel := make(chan event.Event)
			resourceCache := cache.NewResourceCacheMap()
			taskContext := taskrunner.NewTaskContext(eventChannel, resourceCache)

			restMapper := testutil.NewFakeRESTMapper(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}, schema.GroupVersionKind{
				Group:   "anothercustom.io",
				Version: "v2",
				Kind:    "AnotherCustom",
			})

			ao := &fakeApplyOptions{}
			oldAO := applyOptionsFactoryFunc
			applyOptionsFactoryFunc = func(string, chan<- event.Event, common.ServerSideOptions, common.DryRunStrategy,
				dynamic.Interface, discovery.OpenAPISchemaInterface) applyOptions {
				return ao
			}
			defer func() { applyOptionsFactoryFunc = oldAO }()

			applyTask := &ApplyTask{
				Objects:        tc.objs,
				InfoHelper:     &fakeInfoHelper{},
				Mapper:         restMapper,
				DryRunStrategy: drs,
			}

			var events []event.Event
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for msg := range eventChannel {
					events = append(events, msg)
				}
			}()

			applyTask.Start(taskContext)
			<-taskContext.TaskChannel()
			close(eventChannel)
			wg.Wait()

			assert.Equal(t, len(tc.expectedObjects), len(ao.passedObjects))
			for i, obj := range ao.passedObjects {
				actual, err := object.InfoToObjMeta(obj)
				if err != nil {
					continue
				}
				assert.Equal(t, tc.expectedObjects[i], actual)
			}

			assert.Equal(t, len(tc.expectedEvents), len(events))
			for i, e := range events {
				assert.Equal(t, tc.expectedEvents[i].Type, e.Type)
				assert.Equal(t, tc.expectedEvents[i].ApplyEvent.Error.Error(), e.ApplyEvent.Error.Error())
			}

			applyIds := object.UnstructuredSetToObjMetadataSet(tc.objs)

			im := taskContext.InventoryManager()

			// validate record of failed prunes
			for _, id := range tc.expectedFailed {
				assert.Truef(t, im.IsFailedApply(id), "ApplyTask should mark object as failed: %s", id)
			}
			for _, id := range applyIds.Diff(tc.expectedFailed) {
				assert.Falsef(t, im.IsFailedApply(id), "ApplyTask should NOT mark object as failed: %s", id)
			}
			// validate record of skipped prunes
			for _, id := range tc.expectedSkipped {
				assert.Truef(t, im.IsSkippedApply(id), "ApplyTask should mark object as skipped: %s", id)
			}
			for _, id := range applyIds.Diff(tc.expectedSkipped) {
				assert.Falsef(t, im.IsSkippedApply(id), "ApplyTask should NOT mark object as skipped: %s", id)
			}
		})
	}
}

func toUnstructured(obj map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: obj,
	}
}

func toUnstructureds(rss []resourceInfo) []*unstructured.Unstructured {
	var objs []*unstructured.Unstructured

	for _, rs := range rss {
		objs = append(objs, &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": rs.apiVersion,
				"kind":       rs.kind,
				"metadata": map[string]interface{}{
					"name":       rs.name,
					"namespace":  rs.namespace,
					"uid":        string(rs.uid),
					"generation": rs.generation,
					"annotations": map[string]interface{}{
						"config.k8s.io/owning-inventory": "id",
					},
				},
			},
		})
	}
	return objs
}

type fakeApplyOptions struct {
	objects       []*resource.Info
	passedObjects []*resource.Info
}

func (f *fakeApplyOptions) Run() error {
	var err error
	for _, obj := range f.objects {
		if strings.Contains(obj.Name, "failure") {
			err = fmt.Errorf("expected apply error")
		} else {
			f.passedObjects = append(f.passedObjects, obj)
		}
	}
	return err
}

func (f *fakeApplyOptions) SetObjects(objects []*resource.Info) {
	f.objects = objects
}

type fakeInfoHelper struct{}

func (f *fakeInfoHelper) UpdateInfo(*resource.Info) error {
	return nil
}

func (f *fakeInfoHelper) BuildInfos(objs []*unstructured.Unstructured) ([]*resource.Info, error) {
	return object.UnstructuredsToInfos(objs)
}

func (f *fakeInfoHelper) BuildInfo(obj *unstructured.Unstructured) (*resource.Info, error) {
	return object.UnstructuredToInfo(obj)
}
