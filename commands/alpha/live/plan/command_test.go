package plan

import (
	"bytes"
	"testing"

	kptplanner "github.com/kptdev/kpt/pkg/live/planner"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestPrintTextSkip(t *testing.T) {
	testCases := map[string]struct {
		plan     *kptplanner.Plan
		expected string
	}{
		"skipped resource with reason": {
			plan: &kptplanner.Plan{
				Actions: []kptplanner.Action{
					{
						Type:       kptplanner.Skip,
						Group:      "apps",
						Kind:       "Deployment",
						Name:       "foo",
						Namespace:  "default",
						SkipReason: "some skip reason",
					},
				},
			},
			expected: "kpt will perform the following actions:\n\x1b[33m\t= apps/Deployment default/foo\n\x1b[0m\t\tsome skip reason\n\n",
		},
		"skipped resource without reason": {
			plan: &kptplanner.Plan{
				Actions: []kptplanner.Action{
					{
						Type:      kptplanner.Skip,
						Group:     "apps",
						Kind:      "Deployment",
						Name:      "foo",
						Namespace: "default",
					},
				},
			},
			expected: "kpt will perform the following actions:\n\x1b[33m\t= apps/Deployment default/foo\n\x1b[0m\n",
		},
	}

	for tn := range testCases {
		tc := testCases[tn]
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			ioStreams := genericclioptions.IOStreams{
				Out:    &buf,
				ErrOut: &buf,
			}
			var objs []*unstructured.Unstructured
			err := printText(tc.plan, objs, ioStreams)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, buf.String())
		})
	}
}
