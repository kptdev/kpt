// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
)

// SendEventTask is an implementation of the Task interface
// that will send the provided event on the eventChannel when
// executed.
type SendEventTask struct {
	Event event.Event
}

// Start start a separate goroutine that will send the
// event and then push a TaskResult on the taskChannel to
// signal to the taskrunner that the task is completed.
func (s *SendEventTask) Start(taskContext *taskrunner.TaskContext) {
	go func() {
		taskContext.SendEvent(s.Event)
		taskContext.TaskChannel() <- taskrunner.TaskResult{}
	}()
}

// ClearTimeout doesn't do anything as SendEventTask doesn't support
// timeouts.
func (s *SendEventTask) ClearTimeout() {}
