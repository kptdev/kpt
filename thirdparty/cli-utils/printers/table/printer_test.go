// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/table"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
)

var (
	createdOpResult = event.Created
	prunedOpResult  = event.Pruned
)

func TestActionColumnDef(t *testing.T) {
	testCases := map[string]struct {
		resource       table.Resource
		columnWidth    int
		expectedOutput string
	}{
		"unexpected implementation of Resource interface": {
			resource:       &SubResourceInfo{},
			columnWidth:    15,
			expectedOutput: "",
		},
		"neither applied nor pruned": {
			resource:       &ResourceInfo{},
			columnWidth:    15,
			expectedOutput: "",
		},
		"applied": {
			resource: &ResourceInfo{
				ResourceAction: event.ApplyAction,
				ApplyOpResult:  createdOpResult,
			},
			columnWidth:    15,
			expectedOutput: "Created",
		},
		"pruned": {
			resource: &ResourceInfo{
				ResourceAction: event.PruneAction,
				PruneOpResult:  prunedOpResult,
			},
			columnWidth:    15,
			expectedOutput: "Pruned",
		},
		"trimmed output": {
			resource: &ResourceInfo{
				ResourceAction: event.ApplyAction,
				ApplyOpResult:  createdOpResult,
			},
			columnWidth:    5,
			expectedOutput: "Creat",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := actionColumnDef.PrintResource(&buf, tc.columnWidth, tc.resource)
			if err != nil {
				t.Error(err)
			}

			if want, got := tc.expectedOutput, buf.String(); want != got {
				t.Errorf("expected %q, but got %q", want, got)
			}
		})
	}
}
