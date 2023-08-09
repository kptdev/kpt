// Copyright 2022 The kpt Authors
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

package pull

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/printer"
	fakeprint "github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCmd(t *testing.T) {
	pkgRevName := "repo-fjdos9u2nfe2f32"
	ns := "ns"

	scheme, err := createScheme()
	if err != nil {
		t.Fatalf("error creating scheme: %v", err)
	}

	testCases := map[string]struct {
		resources map[string]string
		output    string
	}{
		"simple package": {
			resources: map[string]string{
				"Kptfile": strings.TrimSpace(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
    name: bar
    annotations:
        config.kubernetes.io/local-config: "true"
info:
    description: sample description			  
				`),
				"cm.yaml": strings.TrimSpace(`
apiVersion: v1
kind: ConfigMap
metadata:
    name: game-config
    namespace: default
data:
    foo: bar
				`),
			},
			output: `
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: porch.kpt.dev/v1alpha1
  kind: KptRevisionMetadata
  metadata:
    name: repo-fjdos9u2nfe2f32
    namespace: ns
    creationTimestamp: null
    resourceVersion: "999"
    annotations:
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: '.KptRevisionMetadata'
      config.kubernetes.io/path: '.KptRevisionMetadata'
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: bar
    annotations:
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      config.kubernetes.io/path: 'Kptfile'
  info:
    description: sample description
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: game-config
    namespace: default
    annotations:
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'cm.yaml'
      config.kubernetes.io/path: 'cm.yaml'
  data:
    foo: bar			
			`,
		},
		"package with subdirectory": {
			resources: map[string]string{
				"Kptfile": strings.TrimSpace(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
    name: bar
    annotations:
        config.kubernetes.io/local-config: "true"
info:
    description: sample description			  
				`),
				"sub/cm.yaml": strings.TrimSpace(`
apiVersion: v1
kind: ConfigMap
metadata:
    name: game-config
    namespace: default
data:
    foo: bar
				`),
			},
			output: `
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: porch.kpt.dev/v1alpha1
  kind: KptRevisionMetadata
  metadata:
    name: repo-fjdos9u2nfe2f32
    namespace: ns
    creationTimestamp: null
    resourceVersion: "999"
    annotations:
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: '.KptRevisionMetadata'
      config.kubernetes.io/path: '.KptRevisionMetadata'
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: bar
    annotations:
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      config.kubernetes.io/path: 'Kptfile'
  info:
    description: sample description
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: game-config
    namespace: default
    annotations:
      config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'sub/cm.yaml'
      config.kubernetes.io/path: 'sub/cm.yaml'
  data:
    foo: bar			
			`,
		},
	}

	for tn := range testCases {
		tc := testCases[tn]
		t.Run(tn, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&porchapi.PackageRevisionResources{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pkgRevName,
						Namespace: "ns",
					},
					Spec: porchapi.PackageRevisionResourcesSpec{
						PackageName: "foo",
						Resources:   tc.resources,
					},
				}).
				Build()
			output := &bytes.Buffer{}
			ctx := fakeprint.CtxWithPrinter(output, output)
			r := &runner{
				ctx: ctx,
				cfg: &genericclioptions.ConfigFlags{
					Namespace: &ns,
				},
				client:  c,
				printer: printer.FromContextOrDie(ctx),
			}
			cmd := &cobra.Command{}
			err = r.runE(cmd, []string{pkgRevName})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(strings.TrimSpace(tc.output), strings.TrimSpace(output.String())); diff != "" {
				t.Errorf("Unexpected result (-want, +got): %s", diff)
			}
		})
	}
}
