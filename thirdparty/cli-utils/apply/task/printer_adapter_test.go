// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
)

func TestKubectlPrinterAdapter(t *testing.T) {
	ch := make(chan event.Event)
	buffer := bytes.Buffer{}
	operation := "serverside-applied"

	adapter := KubectlPrinterAdapter{
		ch:        ch,
		groupName: "test-0",
	}

	toPrinterFunc := adapter.toPrinterFunc()
	resourcePrinter, err := toPrinterFunc(operation)
	assert.NoError(t, err)

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "name",
				"namespace": "namespace",
			},
		},
	}

	// Need to run this in a separate gorutine since go channels
	// are blocking.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = resourcePrinter.PrintObj(deployment, &buffer)
	}()
	msg := <-ch
	wg.Wait()

	assert.NoError(t, err)
	assert.Equal(t, event.ServersideApplied, msg.ApplyEvent.Operation)
	assert.Equal(t, deployment, msg.ApplyEvent.Resource)
}
