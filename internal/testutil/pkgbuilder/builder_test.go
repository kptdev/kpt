package pkgbuilder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildKptfile(t *testing.T) {
	var repos ReposInfo
	pkg := &pkg{}
	pkg.Kptfile = &Kptfile{
		Pipeline: &Pipeline{
			Functions: []Function{
				{Image: "example.com/fn1"},
				{ConfigPath: "config1"},
				{Image: "example.com/fn2"},
			},
		},
	}

	got := buildKptfile(pkg, "test1", repos)
	want := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
pipeline:
  mutators:
  - image: example.com/fn1
  - configPath: config1
  - image: example.com/fn2
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("buildKptfile returned unexpected diff (-want +got):\n%s", diff)
	}
}
