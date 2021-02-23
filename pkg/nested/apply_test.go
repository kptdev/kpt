// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package nested

import (
	"context"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"os"
	"path/filepath"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/provider"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
	"testing"
)

func TestApplier(t *testing.T) {
	testCases := map[string]struct {
		manifests           map[string]string
		subpackageManifests map[string]string
		namespace           string
		enforceNamespace    bool
		validate            bool

		infosCount int
		namespaces []string
	}{
		"multiple manifests with subpackages": {
			manifests: map[string]string{
				"dep.yaml": depManifest,
				"cm.yaml":  cmManifest,
				"Kptfile":  kptFile,
			},
			subpackageManifests: map[string]string{
				"cm.yaml": subpackageManifest,
				"Kptfile": subpackageKptfile,
			},
			namespace:        "default",
			enforceNamespace: true,

			infosCount: 4,
			namespaces: []string{"default", "default", "default", "default"},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pr := newProvider()

			dir, err := ioutil.TempDir("", "nested-package-test")
			assert.NoError(t, err)
			if tc.subpackageManifests != nil {
				err = os.Mkdir(filepath.Join(dir, "subpackage"), 0700)
				assert.NoError(t, err)
			}
			for filename, content := range tc.manifests {
				p := filepath.Join(dir, filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}
			for filename, content := range tc.subpackageManifests {
				p := filepath.Join(filepath.Join(dir), "subpackage", filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}

			loader := NewLoader(pr.Factory())
			ninv, err := loader.Read(nil, []string{dir})
			assert.NoError(t, err)

			applier, err := NewApplier(pr)
			assert.NoError(t, err)

			events := applier.Apply(context.TODO(), ninv, apply.Options{})
			for e := range events {
				if e.Type == event.ErrorType {
					t.Errorf("unexpected error %v", e.ErrorEvent.Err)
				}
			}
		})
	}
}

func TestApplierWithTestData(t *testing.T) {
	pr := newProvider()
	loader := NewLoader(pr.Factory())
	ninv, err := loader.Read(nil, []string{"testdata/d"})
	assert.NoError(t, err)

	applier, err := NewApplier(pr)
	assert.NoError(t, err)

	events := applier.Apply(context.TODO(), ninv, apply.Options{})
	for e := range events {
		if e.Type == event.ErrorType {
			t.Errorf("unexpected error %v", e.ErrorEvent.Err)
		}
	}
}

func newProvider() provider.Provider {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(&factory.CachingRESTClientGetter{
		Delegate: kubeConfigFlags,
	})
	f := util.NewFactory(matchVersionKubeConfigFlags)
	p := live.NewResourceGroupProvider(f)
	return p
}
