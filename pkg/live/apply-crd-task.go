// Copyright 2021 The kpt Authors
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

package live

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// ApplyCRDTask encapsulates information necessary to apply a
// Custom Resource Definition (CRD) as a task within a task queue.
// Implements the Task interface.
type ApplyCRDTask struct {
	factory cmdutil.Factory
	crd     *unstructured.Unstructured
}

func (a *ApplyCRDTask) Name() string {
	return "apply-rg-crd"
}

func (a *ApplyCRDTask) Action() event.ResourceAction {
	return event.ApplyAction
}

func (a *ApplyCRDTask) Identifiers() object.ObjMetadataSet {
	return object.UnstructuredSetToObjMetadataSet([]*unstructured.Unstructured{a.crd})
}

// NewApplyCRDTask returns a pointer to an ApplyCRDTask struct,
// containing fields to run the task.
func NewApplyCRDTask(factory cmdutil.Factory, crd *unstructured.Unstructured) *ApplyCRDTask {
	return &ApplyCRDTask{
		factory: factory,
		crd:     crd,
	}
}

// Start function is called to start the task running.
func (a *ApplyCRDTask) Start(taskContext *taskrunner.TaskContext) {
	go func() {
		mapper, err := a.factory.ToRESTMapper()
		if err != nil {
			taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
			return
		}
		mapping, err := mapper.RESTMapping(crdGroupKind)
		if err != nil {
			taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
			return
		}
		client, err := a.factory.UnstructuredClientForMapping(mapping)
		if err != nil {
			taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
			return
		}
		// Set the "last-applied-annotation" so future applies work correctly.
		if err := util.CreateApplyAnnotation(a.crd, unstructured.UnstructuredJSONScheme); err != nil {
			taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
			return
		}
		// Apply the CRD to the cluster and ignore already exists error.
		var clearResourceVersion = false
		var emptyNamespace = ""
		helper := resource.NewHelper(client, mapping)
		_, err = helper.Create(emptyNamespace, clearResourceVersion, a.crd)
		if err != nil {
			taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
			return
		}
		taskContext.TaskChannel() <- taskrunner.TaskResult{}
	}()
}

func (a *ApplyCRDTask) Cancel(_ *taskrunner.TaskContext) {}

func (a *ApplyCRDTask) StatusUpdate(_ *taskrunner.TaskContext, _ object.ObjMetadata) {}
