// Copyright 2020 The kpt Authors
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

package merge_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/util/merge"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMerge3_Nested_packages(t *testing.T) {
	annotationSetter := yaml.SetAnnotation("foo", "bar")
	labelSetter := yaml.SetLabel("bar", "foo")

	testCases := []struct {
		name               string
		includeSubPackages bool
		original           *pkgbuilder.RootPkg
		upstream           *pkgbuilder.RootPkg
		local              *pkgbuilder.RootPkg
		expected           *pkgbuilder.RootPkg
	}{
		{
			name:               "subpackages are merged if included",
			includeSubPackages: true,
			original:           createPkg(),
			upstream:           createPkg(annotationSetter),
			local:              createPkg(labelSetter),
			expected:           createPkg(labelSetter, annotationSetter),
		},
		{
			name:               "subpackages are not merged if not included",
			includeSubPackages: false,
			original:           createPkg(),
			upstream:           createPkg(annotationSetter),
			local:              createPkg(labelSetter),
			expected: createPkgMultipleMutators(
				[]yaml.Filter{
					labelSetter,
					annotationSetter,
				},
				[]yaml.Filter{
					labelSetter,
				},
			),
		},
		{
			name:               "local copy defines the package boundaries if different from upstream",
			includeSubPackages: false,
			original: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
		{
			name:               "upstream changes not included if in a different package",
			includeSubPackages: false,
			original: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a"). // No Kptfile
									WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			original := test.original.ExpandPkg(t, testutil.EmptyReposInfo)
			updated := test.upstream.ExpandPkg(t, testutil.EmptyReposInfo)
			local := test.local.ExpandPkg(t, testutil.EmptyReposInfo)
			expected := test.expected.ExpandPkg(t, testutil.EmptyReposInfo)
			err := merge.Merge3{
				OriginalPath:       original,
				UpdatedPath:        updated,
				DestPath:           local,
				MergeOnPath:        true,
				IncludeSubPackages: test.includeSubPackages,
			}.Merge()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			diffs, err := copyutil.Diff(local, expected)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !assert.Empty(t, diffs.List()) {
				t.FailNow()
			}
		})
	}
}

func createPkg(mutators ...yaml.Filter) *pkgbuilder.RootPkg {
	return createPkgMultipleMutators(mutators, mutators)
}

func createPkgMultipleMutators(packageMutators, subPackageMutators []yaml.Filter) *pkgbuilder.RootPkg {
	return pkgbuilder.NewRootPkg().
		WithKptfile().
		WithResource(pkgbuilder.DeploymentResource, packageMutators...).
		WithSubPackages(
			pkgbuilder.NewSubPkg("a").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
			pkgbuilder.NewSubPkg("b").
				WithResource(pkgbuilder.DeploymentResource, packageMutators...).
				WithSubPackages(
					pkgbuilder.NewSubPkg("c").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
				),
		)
}

func TestMerge3_Merge_path(t *testing.T) {
	testCases := map[string]struct {
		origin   string
		update   string
		local    string
		expected string
		errMsg   string
	}{
		`Most common: add namespace and name-prefix on local, merge upstream changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 4
`},

		`Add namespace and name-prefix on local manually without adding annotations, adds new resource`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
`},

		`Conflict: User fetches package, copies a resource in same file, adds different name suffix`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-1
  namespace: my-space
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-2
  namespace: my-space
spec:
  replicas: 3
`,
			errMsg: `found duplicate "local" resources in file "f1.yaml"`},

		`Publisher changes name in upstream but want to maintain original identity, no local customizations, fetch upstream changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 4
`},

		`Publisher changes name in upstream but want to maintain original identity, consumer adds name-prefix on local, fetch upstream changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new
  namespace: my-space
spec:
  replicas: 4
`},

		`Publisher changes name in upstream but don't want to maintain original identity which is equivalent 
to delete existing resource and add new one, consumer adds name-prefix on local`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-new
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-new
spec:
  replicas: 4
`},

		`Publisher changes name multiple times in upstream but maintains original identity, no local customizations,
fetch upstream changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 4`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new-again
spec:
  replicas: 5`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 5
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new-again
spec:
  replicas: 5
`},

		`Publisher changes name multiple times in upstream but maintains original identity, consumer adds name-prefix 
on local, fetch upstream changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new
spec:
  replicas: 4`,
			update: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new-again
spec:
  replicas: 5`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: dev-nginx-deployment
  namespace: my-space
spec:
  replicas: 5
`,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: default/nginx-deployment
  name: nginx-deployment-new-again
  namespace: my-space
spec:
  replicas: 5
`},
		`Publisher adds metadata.annotations in upstream in a non-identity kustomization resource, consumer adds changes to resource body
on local, fetch upstream changes`: {
			origin: `
apiVersion: kustomize.config.k8s.io/v1beta1
metadata:
  labels:
    color: blue
commonLabels:
  app: dev`,
			update: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  labels:
    color: blue
  annotations:
    id.example.org: abcd
commonLabels:
  app: dev`,
			local: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  labels:
    color: blue
commonLabels:
  app: dev
  tier: backend
`,
			expected: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  labels:
    color: blue
  annotations:
    id.example.org: abcd
commonLabels:
  app: dev
  tier: backend
`},
		`Publisher adds commonLabels in upstream in a non-identity kustomization resource, consumer adds changes to resource body
on local, fetch upstream changes`: {
			origin: `
commonLabels:
  app: dev`,
			update: `
commonLabels:
  tier: backend
  app: dev`,
			local: `
commonLabels:
  app: dev
  tier: backend
  db: mysql
`,
			expected: `
commonLabels:
  app: dev
  tier: backend
  db: mysql
`},

		`Version changes are just like any other changes`: {
			origin: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3`,
			update: `
apiVersion: apps/v2
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4`,
			local: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
`,
			expected: `
apiVersion: apps/v2
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
`},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// setup the local directory
			dir := t.TempDir()

			err := os.MkdirAll(filepath.Join(dir, "localDir"), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = os.MkdirAll(filepath.Join(dir, "updatedDir"), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = os.MkdirAll(filepath.Join(dir, "originalDir"), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = os.WriteFile(filepath.Join(dir, "originalDir", "f1.yaml"), []byte(strings.TrimSpace(tc.origin)), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = os.WriteFile(filepath.Join(dir, "updatedDir", "f1.yaml"), []byte(strings.TrimSpace(tc.update)), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = os.WriteFile(filepath.Join(dir, "localDir", "f1.yaml"), []byte(strings.TrimSpace(tc.local)), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = merge.Merge3{
				OriginalPath: filepath.Join(dir, "originalDir"),
				UpdatedPath:  filepath.Join(dir, "updatedDir"),
				DestPath:     filepath.Join(dir, "localDir"),
				MergeOnPath:  true,
			}.Merge()
			if tc.errMsg == "" {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			} else {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), tc.errMsg) {
					t.FailNow()
				}
				return
			}

			b, err := os.ReadFile(filepath.Join(dir, "localDir", "f1.yaml"))
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(string(b))) {
				t.FailNow()
			}
		})
	}
}
