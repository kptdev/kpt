package engine

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCRDName(t *testing.T) {
	r, err := NewOCIRegistry("gcr.io/example-google-project-id")
	if err != nil {
		t.Fatalf("error from NewOCIRegistry: %v", err)
	}

	ref, err := r.referenceForCRD(schema.GroupVersionKind{Group: "fn.kpt.dev", Version: "v1alpha1", Kind: "RenderHelmChart"})
	if err != nil {
		t.Fatalf("error from referenceForCRD: %v", err)
	}

	got := ref.String()
	want := "gcr.io/example-google-project-id/crds/fn.kpt.dev/renderhelmchart:v1alpha1"

	if got != want {
		t.Errorf("unexpected value from referenceForCRD; got %q, want %q", got, want)
	}
}
