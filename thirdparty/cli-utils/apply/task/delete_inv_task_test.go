// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply/cache"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestDeleteInvTask(t *testing.T) {
	testCases := map[string]struct {
		err     error
		isError bool
	}{
		"no error case": {
			err:     nil,
			isError: false,
		},
		"error is returned in result": {
			err:     apierrors.NewResourceExpired("unused message"),
			isError: true,
		},
		"inventory not found is not error and not returned": {
			err: apierrors.NewNotFound(schema.GroupResource{Resource: "simples"},
				"unused-resource-name"),
			isError: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := inventory.NewFakeClient(object.ObjMetadataSet{})
			client.Err = tc.err
			eventChannel := make(chan event.Event)
			resourceCache := cache.NewResourceCacheMap()
			context := taskrunner.NewTaskContext(eventChannel, resourceCache)

			task := DeleteInvTask{
				TaskName:  taskName,
				InvClient: client,
				InvInfo:   localInv,
				DryRun:    common.DryRunNone,
			}
			if taskName != task.Name() {
				t.Errorf("expected task name (%s), got (%s)", taskName, task.Name())
			}
			task.Start(context)
			result := <-context.TaskChannel()
			if tc.isError {
				if tc.err != result.Err {
					t.Errorf("running DeleteInvTask expected error (%s), got (%s)", tc.err, result.Err)
				}
			} else {
				if result.Err != nil {
					t.Errorf("unexpected error running DeleteInvTask: %s", result.Err)
				}
			}
		})
	}
}
