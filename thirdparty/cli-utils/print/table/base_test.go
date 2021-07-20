// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var (
	endColumnDef = ColumnDef{
		ColumnName:   "end",
		ColumnHeader: "END",
		ColumnWidth:  3,
		PrintResourceFunc: func(w io.Writer, width int, r Resource) (i int,
			err error) {
			return fmt.Fprint(w, "end")
		},
	}
)

func TestBaseTablePrinter_PrintTable(t *testing.T) {
	testCases := map[string]struct {
		columnDefinitions []ColumnDefinition
		resources         []Resource
		expectedOutput    string
	}{
		"no resources": {
			columnDefinitions: []ColumnDefinition{
				MustColumn("resource"),
				endColumnDef,
			},
			resources: []Resource{},
			expectedOutput: `
RESOURCE                                  END
`,
		},
		"with resource": {
			columnDefinitions: []ColumnDefinition{
				MustColumn("resource"),
				endColumnDef,
			},
			resources: []Resource{
				&fakeResource{
					resourceStatus: &pe.ResourceStatus{
						Identifier: object.ObjMetadata{
							Namespace: "default",
							Name:      "Foo",
							GroupKind: schema.GroupKind{
								Group: "apps",
								Kind:  "Deployment",
							},
						},
					},
				},
			},
			expectedOutput: `
RESOURCE                                  END
Deployment/Foo                            end
`,
		},
		"sub resources": {
			columnDefinitions: []ColumnDefinition{
				MustColumn("resource"),
				endColumnDef,
			},
			resources: []Resource{
				&fakeResource{
					resourceStatus: &pe.ResourceStatus{
						Identifier: object.ObjMetadata{
							Namespace: "default",
							Name:      "Foo",
							GroupKind: schema.GroupKind{
								Group: "apps",
								Kind:  "Deployment",
							},
						},
						GeneratedResources: []*pe.ResourceStatus{
							{
								Identifier: object.ObjMetadata{
									Namespace: "default",
									Name:      "Bar",
									GroupKind: schema.GroupKind{
										Group: "apps",
										Kind:  "ReplicaSet",
									},
								},
							},
						},
					},
				},
			},
			expectedOutput: `
RESOURCE                                  END
Deployment/Foo                            end
└─ ReplicaSet/Bar                         end
`,
		},
		"trim long content": {
			columnDefinitions: []ColumnDefinition{
				MustColumn("resource"),
				endColumnDef,
			},
			resources: []Resource{
				&fakeResource{
					resourceStatus: &pe.ResourceStatus{
						Identifier: object.ObjMetadata{
							Namespace: "default",
							Name:      "VeryLongNameThatShouldBeTrimmed",
							GroupKind: schema.GroupKind{
								Group: "apps",
								Kind:  "Deployment",
							},
						},
					},
				},
			},
			expectedOutput: `
RESOURCE                                  END
Deployment/VeryLongNameThatShouldBeTrimm  end
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()

			printer := &BaseTablePrinter{
				IOStreams: ioStreams,
				Columns:   tc.columnDefinitions,
			}

			resourceStates := &fakeResourceStates{
				resources: tc.resources,
			}

			printer.PrintTable(resourceStates, 0)

			assert.Equal(t,
				strings.TrimSpace(tc.expectedOutput),
				strings.TrimSpace(outBuffer.String()))
		})
	}
}

type fakeResourceStates struct {
	resources []Resource
}

func (r *fakeResourceStates) Resources() []Resource {
	return r.resources
}

func (r *fakeResourceStates) Error() error {
	return nil
}

type fakeResource struct {
	resourceStatus *pe.ResourceStatus
}

func (r *fakeResource) Identifier() object.ObjMetadata {
	return r.resourceStatus.Identifier
}

func (r *fakeResource) ResourceStatus() *pe.ResourceStatus {
	return r.resourceStatus
}

func (r *fakeResource) SubResources() []Resource {
	var resources []Resource
	for _, res := range r.resourceStatus.GeneratedResources {
		resources = append(resources, &fakeResource{
			resourceStatus: res,
		})
	}
	return resources
}
