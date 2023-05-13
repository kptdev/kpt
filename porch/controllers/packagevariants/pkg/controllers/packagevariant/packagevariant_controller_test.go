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

package packagevariant

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestValidatePackageVariant(t *testing.T) {
	packageVariantHeader := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv`

	testCases := map[string]struct {
		packageVariant string
		expectedErr    string
	}{
		"empty spec": {
			packageVariant: packageVariantHeader,
			expectedErr:    "missing required field spec.upstream; missing required field spec.downstream",
		},

		"missing package names": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    revision: v1
    repo: blueprints
  downstream:
    repo: deployments
`,
			expectedErr: "missing required field spec.upstream.package; missing required field spec.downstream.package",
		},

		"empty adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
`,
		},

		"invalid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  adoptionPolicy: invalid
  deletionPolicy: invalid
`,
			expectedErr: "spec.adoptionPolicy field can only be \"adoptNone\" or \"adoptExisting\"; spec.deletionPolicy can only be \"orphan\" or \"delete\"",
		},

		"valid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  adoptionPolicy: adoptExisting
  deletionPolicy: orphan
`,
		},

		"validate package context": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  packageContext:
    data:
      foo: bar
      hello: there
    removeKeys:
    - bar
    - foobar
`,
		},

		"name in package context data": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  packageContext:
    data:
      name: test
`,
			expectedErr: "spec.packageContext.data: Invalid value: map[string]string{\"name\":\"test\"}: must not contain the key \"name\"",
		},

		"name in package context removeKeys": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  packageContext:
    removeKeys:
    - name
`,
			expectedErr: "spec.packageContext.removeKeys: Invalid value: []string{\"name\"}: must not contain the key \"name\"",
		},

		"package-path in package context data": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  packageContext:
    data:
      package-path: test
`,
			expectedErr: "spec.packageContext.data: Invalid value: map[string]string{\"package-path\":\"test\"}: must not contain the key \"package-path\"",
		},
		"package-path in package context removeKeys": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  packageContext:
    removeKeys:
    - package-path
`,
			expectedErr: "spec.packageContext.removeKeys: Invalid value: []string{\"package-path\"}: must not contain the key \"package-path\"",
		},
		"valid injectors": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  injectors:
  - group: porch.kpt.dev
    version: v1alpha1
    kind: Repository
    name: foo
  - name: bar
`,
		},
		"invalid injectors": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foo
    revision: v1
    repo: blueprints
  downstream:
    package: foo
    repo: deployments
  injectors:
  - group: porch.kpt.dev
    version: v1alpha1
    kind: Repository
    name: foo
  - name: ""
`,
			expectedErr: "spec.injectors[1].name must not be empty",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageVariant), &pv))
			actualErr := combineErrors(validatePackageVariant(&pv))
			require.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestNewWorkspaceName(t *testing.T) {
	prListHeader := `apiVersion: porch.kpt.dev
kind: PackageRevisionList
metadata:
  name: my-pr-list`

	testCases := map[string]struct {
		packageRevisionList string
		expected            string
	}{
		"empty list": {
			packageRevisionList: prListHeader,
			expected:            "packagevariant-1",
		},

		"two elements with packagevariant prefix": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-1
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-2`,
			expected: "packagevariant-3",
		},

		"two elements, one with packagevariant prefix": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-1
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: foo`,
			expected: "packagevariant-2",
		},

		"two elements, neither with packagevariant prefix": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: foo-1
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: foo-2`,
			expected: "packagevariant-1",
		},

		"two elements with packagevariant prefix, one doesn't match package": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-1
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-2
    packageName: some-other-package`,
			expected: "packagevariant-2",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var prList porchapi.PackageRevisionList
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageRevisionList), &prList))
			actual := string(newWorkspaceName(&prList, "", ""))
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetDownstreamPRs(t *testing.T) {
	prListHeader := `apiVersion: porch.kpt.dev
kind: PackageRevisionList
metadata:
  name: my-pr-list`

	pvStr := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec: 
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar`

	testCases := map[string]struct {
		packageRevisionList string
		expected            []string
		fcOutput            []string
	}{

		// should return nil
		"empty list": {
			packageRevisionList: prListHeader,
			expected:            nil,
		},

		// should return the draft that we own
		"two drafts, one owned": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-1
    lifecycle: Draft
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
  spec:
    workspaceName: packagevariant-2
    lifecycle: Draft
    repository: deployments
    packageName: bar`,
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-1
status:
  publishTimestamp: null
`,
			},
		},

		// should return both drafts that we own
		"one published and two drafts, all owned": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    revision: v1
    workspaceName: packagevariant-1
    lifecycle: Published
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-2
    lifecycle: Draft
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-3
    lifecycle: Draft
    repository: deployments
    packageName: bar`,
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-2
status:
  publishTimestamp: null
`, `apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-3
status:
  publishTimestamp: null
`,
			},
		},

		// should return the latest published that we own
		"three published, latest one not owned": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    revision: v2
    workspaceName: packagevariant-2
    lifecycle: Published
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    revision: v1
    workspaceName: packagevariant-1
    lifecycle: Published
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: some-other-uid-1
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: some-other-uid-2
  spec:
    revision: v3
    workspaceName: packagevariant-3
    lifecycle: Published
    repository: deployments
    packageName: bar`,
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Published
  packageName: bar
  repository: deployments
  revision: v2
  workspaceName: packagevariant-2
status:
  publishTimestamp: null
`,
			},
		},

		// should return just the published and delete the two drafts
		"one published and two drafts, all owned, drafts from different package": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    revision: v1
    workspaceName: packagevariant-1
    lifecycle: Published
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-2
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-2
    lifecycle: Draft
    repository: deployments
    packageName: foo
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-3
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-3
    lifecycle: Draft
    repository: deployments
    packageName: foo`,
			fcOutput: []string{`deleting object: my-pr-2`, `deleting object: my-pr-3`},
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Published
  packageName: bar
  repository: deployments
  revision: v1
  workspaceName: packagevariant-1
status:
  publishTimestamp: null
`,
			},
		},
	}

	var pv api.PackageVariant
	require.NoError(t, yaml.Unmarshal([]byte(pvStr), &pv))

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var prList porchapi.PackageRevisionList
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageRevisionList), &prList))

			fc := &fakeClient{}
			reconciler := &PackageVariantReconciler{Client: fc}

			actualStr := reconciler.getDownstreamPRs(context.TODO(), &pv, &prList)
			var actual []string
			for _, pr := range actualStr {
				bytes, err := yaml.Marshal(pr)
				require.NoError(t, err)
				actual = append(actual, string(bytes))
			}

			require.Equal(t, tc.expected, actual)
			require.Equal(t, tc.fcOutput, fc.output)
		})
	}
}

func TestDeleteOrOrphan(t *testing.T) {
	prStr := `apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: some-other-uid
spec:
  lifecycle: %s
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-3
`

	pvStr := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec: 
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar
  deletionPolicy: %s`

	testCases := map[string]struct {
		deletionPolicy string
		prLifecycle    string
		expectedOutput []string
		expectedPR     string
	}{

		// should delete the PR
		"deletionPolicy delete, lifecycle draft": {
			deletionPolicy: string(api.DeletionPolicyDelete),
			prLifecycle:    string(porchapi.PackageRevisionLifecycleDraft),
			expectedOutput: []string{"deleting object: my-pr"},
		},

		// should delete the PR
		"deletionPolicy delete, lifecycle proposed": {
			deletionPolicy: string(api.DeletionPolicyDelete),
			prLifecycle:    string(porchapi.PackageRevisionLifecycleProposed),
			expectedOutput: []string{"deleting object: my-pr"},
		},

		// should propose the PR for deletion
		"deletionPolicy delete, lifecycle published": {
			deletionPolicy: string(api.DeletionPolicyDelete),
			prLifecycle:    string(porchapi.PackageRevisionLifecyclePublished),
			expectedOutput: []string{"updating object: my-pr"},
			expectedPR: `apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: some-other-uid
spec:
  lifecycle: DeletionProposed
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-3
status:
  publishTimestamp: null
`,
		},

		// should do nothing
		"deletionPolicy delete, lifecycle deletionProposed": {
			deletionPolicy: string(api.DeletionPolicyDelete),
			prLifecycle:    string(porchapi.PackageRevisionLifecycleDeletionProposed),
			expectedOutput: nil,
		},

		// should remove the pv's owner reference from the pr
		"deletionPolicy orphan, lifecycle draft": {
			deletionPolicy: string(api.DeletionPolicyOrphan),
			prLifecycle:    string(porchapi.PackageRevisionLifecycleDraft),
			expectedOutput: []string{"updating object: my-pr"},
			expectedPR: `apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: some-other-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-3
status:
  publishTimestamp: null
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal(
				[]byte(fmt.Sprintf(pvStr, tc.deletionPolicy)), &pv))

			var pr porchapi.PackageRevision
			require.NoError(t, yaml.Unmarshal(
				[]byte(fmt.Sprintf(prStr, tc.prLifecycle)), &pr))

			fc := &fakeClient{}
			reconciler := &PackageVariantReconciler{Client: fc}
			reconciler.deleteOrOrphan(context.Background(), &pr, &pv)

			require.Equal(t, tc.expectedOutput, fc.output)

			if tc.expectedPR != "" {
				prAfter, err := yaml.Marshal(&pr)
				require.NoError(t, err)
				require.Equal(t, tc.expectedPR, string(prAfter))
			}
		})
	}
}

func TestAdoptionPolicy(t *testing.T) {
	prListHeader := `apiVersion: porch.kpt.dev
kind: PackageRevisionList
metadata:
  name: my-pr-list`

	pvStr := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec: 
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar
  adoptionPolicy: %s`

	testCases := map[string]struct {
		packageRevisionList string
		adoptionPolicy      string
		expected            []string
		clientOutput        []string
	}{

		// should return the previously unowned draft, with owner references added
		"owned published, unowned draft, adoptExisting": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-1
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-1
    lifecycle: Published
    revision: v1
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-2
  spec:
    workspaceName: packagevariant-2
    lifecycle: Draft
    repository: deployments
    packageName: bar`,
			adoptionPolicy: string(api.AdoptionPolicyAdoptExisting),
			clientOutput:   []string{"updating object: my-pr-2"},
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr-2
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    controller: true
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-2
status:
  publishTimestamp: null
`,
			},
		},

		// should return just the draft that we own
		"two drafts, one owned, adoptNone": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-1
    ownerReferences:
    - apiVersion: config.porch.kpt.dev
      kind: PackageVariant
      name: my-pv
      uid: pv-uid
  spec:
    workspaceName: packagevariant-1
    lifecycle: Draft
    repository: deployments
    packageName: bar
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-2
  spec:
    workspaceName: packagevariant-2
    lifecycle: Draft
    repository: deployments
    packageName: bar`,
			adoptionPolicy: string(api.AdoptionPolicyAdoptNone),
			clientOutput:   nil,
			expected: []string{`apiVersion: porch.kpt.dev
kind: PackageRevision
metadata:
  creationTimestamp: null
  name: my-pr-1
  ownerReferences:
  - apiVersion: config.porch.kpt.dev
    kind: PackageVariant
    name: my-pv
    uid: pv-uid
spec:
  lifecycle: Draft
  packageName: bar
  repository: deployments
  workspaceName: packagevariant-1
status:
  publishTimestamp: null
`,
			},
		},

		// this should return nil and should not attempt to adopt nor
		// delete the package revision
		"unowned draft, but package name doesn't match, adoptExisting": {
			packageRevisionList: prListHeader + `
items:
- apiVersion: porch.kpt.dev
  kind: PackageRevision
  metadata:
    name: my-pr-1
  spec:
    workspaceName: packagevariant-1
    lifecycle: Draft
    repository: deployments
    packageName: foo
`,
			adoptionPolicy: string(api.AdoptionPolicyAdoptExisting),
			clientOutput:   nil,
			expected:       nil,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fc := &fakeClient{}
			reconciler := &PackageVariantReconciler{Client: fc}
			var prList porchapi.PackageRevisionList
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageRevisionList), &prList))

			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal(
				[]byte(fmt.Sprintf(pvStr, tc.adoptionPolicy)), &pv))

			actualStr := reconciler.getDownstreamPRs(context.TODO(), &pv, &prList)
			var actual []string
			for _, pr := range actualStr {
				bytes, err := yaml.Marshal(pr)
				require.NoError(t, err)
				actual = append(actual, string(bytes))
			}

			require.Equal(t, tc.expected, actual)
			require.Equal(t, tc.clientOutput, fc.output)
		})
	}
}

func TestEnsurePackageContext(t *testing.T) {

	pvBase := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec:
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar
`

	prrBase := `apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevisionResources
metadata:
  name: prr
  namespace: default
spec:
  packageName: nephio-system
  repository: nephio-packages
  resources:
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        description: Example
    package-context.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: kptfile.kpt.dev
        annotations:
          config.kubernetes.io/local-config: "true"
      data:
        name: example
`

	testCases := map[string]struct {
		spec        string
		initialData string
		expectedErr string
		expectedPRR string
	}{
		"empty package context": {
			spec:        ``,
			initialData: ``,
			expectedErr: "",
			expectedPRR: prrBase,
		},
		"add one entry": {
			spec: `  packageContext:
    data:
      foo: bar
`,
			initialData: ``,
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n",
		},
		"add two entries": {
			spec: `  packageContext:
    data:
      foo: bar
      foobar: barfoo
`,
			initialData: ``,
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n        foobar: barfoo\n",
		},
		"add one with existing": {
			spec: `  packageContext:
    data:
      foo: bar
`,
			initialData: "        hello: there\n",
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n        hello: there\n",
		},
		"change existing": {
			spec: `  packageContext:
    data:
      foo: bar
`,
			initialData: "        foo: there\n",
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n",
		},
		"remove one entry": {
			spec: `  packageContext:
    removeKeys:
    - hello
`,
			initialData: "        hello: there\n",
			expectedErr: "",
			expectedPRR: prrBase,
		},
		"remove entry, leave existing": {
			spec: `  packageContext:
    removeKeys:
    - hello
`,
			initialData: "        hello: there\n        foo: bar\n",
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n",
		},
		"remove and add entries": {
			spec: `  packageContext:
    data:
      foobar: barfoo
      there: hello
    removeKeys:
    - hello
`,
			initialData: "        hello: there\n        foo: bar\n",
			expectedErr: "",
			expectedPRR: prrBase + "        foo: bar\n        foobar: barfoo\n        there: hello\n",
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal([]byte(pvBase+tc.spec), &pv))
			var prr porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(prrBase+tc.initialData), &prr))

			actualErr := ensurePackageContext(&pv, &prr)
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}

			var expectedPRR porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(tc.expectedPRR), &expectedPRR))

			require.Equal(t, expectedPRR, prr)
		})
	}
}

func TestEnsureKRMFunctions(t *testing.T) {
	pvBase := `
apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec:
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar
  pipeline:
`[1:]

	prrBase := `
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevisionResources
metadata:
  name: prr
  namespace: default
spec:
  packageName: nephio-system
  repository: nephio-packages
  resources:
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        description: Example
`[1:]

	testCases := map[string]struct {
		initialPipeline string
		pvPipeline      string
		expectedErr     string
		expectedPrr     string
	}{
		"add one mutator with existing mutators": {
			initialPipeline: `
        mutators:
          - image: gcr.io/kpt-fn/set-labels:v0.1
            name: set-labels
            configMap:
              app: foo
          - image: gcr.io/kpt-fn/set-annotations:v0.1
            name: set-annotations`[1:],
			pvPipeline: `
    mutators:
      - image: gcr.io/kpt-fn/set-namespace:v0.1
        name: set-namespace
        configMap:
          namespace: my-ns`[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        mutators:
        - name: PackageVariant.my-pv.set-namespace.0
          image: gcr.io/kpt-fn/set-namespace:v0.1
          configMap:
            namespace: my-ns
        - image: gcr.io/kpt-fn/set-labels:v0.1
          name: set-labels
          configMap:
            app: foo
        - image: gcr.io/kpt-fn/set-annotations:v0.1
          name: set-annotations
`[1:],
		},
		"add two mutators with existing": {
			initialPipeline: `
        mutators:
          - image: gcr.io/kpt-fn/set-labels:v0.1
            name: set-labels
            configMap:
              app: foo
          - image: gcr.io/kpt-fn/set-annotations:v0.1
            name: set-annotations`[1:],
			pvPipeline: `
    mutators:
      - image: gcr.io/kpt-fn/set-namespace:v0.1
        name: set-namespace
        configMap:
          namespace: my-ns
      - image: gcr.io/kpt-fn/format:unstable
        name: format`[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        mutators:
        - name: PackageVariant.my-pv.set-namespace.0
          image: gcr.io/kpt-fn/set-namespace:v0.1
          configMap:
            namespace: my-ns
        - name: PackageVariant.my-pv.format.1
          image: gcr.io/kpt-fn/format:unstable
        - image: gcr.io/kpt-fn/set-labels:v0.1
          name: set-labels
          configMap:
            app: foo
        - image: gcr.io/kpt-fn/set-annotations:v0.1
          name: set-annotations
`[1:],
		},
		"add one mutator with none existing": {
			initialPipeline: "",
			pvPipeline: `
    mutators:
      - image: gcr.io/kpt-fn/set-namespace:v0.1
        name: set-namespace
        configMap:
          namespace: my-ns`[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        mutators:
        - name: PackageVariant.my-pv.set-namespace.0
          image: gcr.io/kpt-fn/set-namespace:v0.1
          configMap:
            namespace: my-ns
`[1:],
		},
		"add none with existing mutators": {
			initialPipeline: `
        mutators:
          - image: gcr.io/kpt-fn/set-labels:v0.1
            name: set-labels
            configMap:
              app: foo
          - image: gcr.io/kpt-fn/set-annotations:v0.1
            name: set-annotations`[1:],
			pvPipeline:  "",
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        mutators:
        - image: gcr.io/kpt-fn/set-labels:v0.1
          name: set-labels
          configMap:
            app: foo
        - image: gcr.io/kpt-fn/set-annotations:v0.1
          name: set-annotations
`[1:],
		},
		"add one mutator with existing with comments": {
			initialPipeline: `
        mutators:
          - image: gcr.io/kpt-fn/set-labels:v0.1
            # this is a comment
            name: set-labels
            configMap:
              app: foo
          - image: gcr.io/kpt-fn/set-annotations:v0.1
            name: set-annotations`[1:],
			pvPipeline: `
    mutators:
      - image: gcr.io/kpt-fn/set-namespace:v0.1
        name: set-namespace
        configMap:
          namespace: my-ns`[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        mutators:
        - name: PackageVariant.my-pv.set-namespace.0
          image: gcr.io/kpt-fn/set-namespace:v0.1
          configMap:
            namespace: my-ns
        - image: gcr.io/kpt-fn/set-labels:v0.1
          # this is a comment
          name: set-labels
          configMap:
            app: foo
        - image: gcr.io/kpt-fn/set-annotations:v0.1
          name: set-annotations
`[1:],
		},
		"add one validator with existing validators": {
			initialPipeline: `
        validators:
          - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
            name: gatekeeper-validate`[1:],
			pvPipeline: `
    validators:
      - image: gcr.io/kpt-fn/validate-name:undefined
        name: validate-name `[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        validators:
        - name: PackageVariant.my-pv.validate-name.0
          image: gcr.io/kpt-fn/validate-name:undefined
        - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
          name: gatekeeper-validate
`[1:],
		},
		"add two validators with existing validators": {
			initialPipeline: `
        validators:
          - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
            name: gatekeeper-validate`[1:],
			pvPipeline: `
    validators:
      - image: gcr.io/kpt-fn/validate-name:undefined
        name: validate-name `[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        validators:
        - name: PackageVariant.my-pv.validate-name.0
          image: gcr.io/kpt-fn/validate-name:undefined
        - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
          name: gatekeeper-validate
`[1:],
		},
		"add none with existing validator": {
			initialPipeline: `
        validators:
          - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
            name: gatekeeper-validate`[1:],
			pvPipeline:  "",
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        validators:
        - image: gcr.io/kpt-fn/gatekeeper-validate:v0.1
          name: gatekeeper-validate
`[1:],
		},
		"add validator and mutator with existing": {
			initialPipeline: `
        validators:
          - image: gcr.io/val1
            name: val1
          - image: gcr.io/val2
            name: val2
        mutators:
          - image: gcr.io/mut1
            name: mut1
          - image: gcr.io/mut2
            name: mut2`[1:],
			pvPipeline: `
    validators:
    - image: gcr.io/val3
      name: val3
    - image: gcr.io/val4
      name: val4
    mutators:
    - image: gcr.io/mut3
      name: mut3
    - image: gcr.io/mut4
      name: mut4
`[1:],
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        validators:
        - name: PackageVariant.my-pv.val3.0
          image: gcr.io/val3
        - name: PackageVariant.my-pv.val4.1
          image: gcr.io/val4
        - image: gcr.io/val1
          name: val1
        - image: gcr.io/val2
          name: val2
        mutators:
        - name: PackageVariant.my-pv.mut3.0
          image: gcr.io/mut3
        - name: PackageVariant.my-pv.mut4.1
          image: gcr.io/mut4
        - image: gcr.io/mut1
          name: mut1
        - image: gcr.io/mut2
          name: mut2
`[1:],
		},
		"remove pv mutator": {
			initialPipeline: `
        mutators:
        - image: gcr.io/mut:v1
          name: PackageVariant.my-pv.mut.0`[1:],
			pvPipeline:  "",
			expectedErr: "",
			expectedPrr: prrBase + "\n",
		},
		"remove pv validator": {
			initialPipeline: `
        validators:
        - image: gcr.io/val:v1
          name: PackageVariant.my-pv.val.0`[1:],
			pvPipeline:  "",
			expectedErr: "",
			expectedPrr: prrBase + "\n",
		},
		"remove pv validator, keep prr one": {
			initialPipeline: `
        validators:
        - image: gcr.io/val:v1
          name: PackageVariant.my-pv.val.0
        - image: gcr.io/val:v1
          name: non-pv-val`[1:],
			pvPipeline:  "",
			expectedErr: "",
			expectedPrr: prrBase + `
      pipeline:
        validators:
        - image: gcr.io/val:v1
          name: non-pv-val
`[1:],
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			locPrrBase := prrBase
			if tc.initialPipeline != "" {
				locPrrBase += "      pipeline:\n"
			}
			var prr porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(locPrrBase+tc.initialPipeline), &prr))
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal([]byte(pvBase+tc.pvPipeline), &pv))

			actualErr := ensureKRMFunctions(&pv, &prr)
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}
			var expectedPRR porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(tc.expectedPrr), &expectedPRR))

			require.Equal(t, expectedPRR, prr)

			// test idempotence
			idemErr := ensureKRMFunctions(&pv, &prr)
			if tc.expectedErr == "" {
				require.NoError(t, idemErr)
			} else {
				require.EqualError(t, idemErr, tc.expectedErr)
			}
			require.Equal(t, expectedPRR, prr) // check that prr still matches expected
		})
	}
}

func TestGeneratePVFuncName(t *testing.T) {
	tt := map[string]struct {
		funcName     string
		pvName       string
		pos          int
		expectedName string
	}{
		"regular func": {
			funcName:     "my-func",
			pvName:       "my-pv",
			pos:          3,
			expectedName: "PackageVariant.my-pv.my-func.3",
		},
		"empty func name": {
			funcName:     "",
			pvName:       "my-pv",
			pos:          0,
			expectedName: "PackageVariant.my-pv..0",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			res := generatePVFuncName(tc.funcName, tc.pvName, tc.pos)

			require.Equal(t, tc.expectedName, res)
		})
	}
}

func TestIsPackageVariantFunc(t *testing.T) {
	tt := map[string]struct {
		funcyaml    string
		pvName      string
		expectedRes bool
	}{
		"valid func name": {
			funcyaml:    "name: PackageVariant.my-pv.my-func.0",
			pvName:      "my-pv",
			expectedRes: true,
		},
		"field name is missing": {
			funcyaml:    "otherkey: PackageVariant.my-pv.my-func.0",
			pvName:      "my-pv",
			expectedRes: false,
		},
		"additional dots": {
			funcyaml:    "name: PackageVariant.too.many.dots.0",
			pvName:      "too",
			expectedRes: false,
		},
		"not enough dots": {
			funcyaml:    "name: PackageVariant.not-enough.dots",
			pvName:      "not-enough",
			expectedRes: false,
		},
		"no PackageVariantPrefix": {
			funcyaml:    "name: noprefix.my-pv.my-func.0",
			pvName:      "my-pv",
			expectedRes: false,
		},
		"pv-name mismatch": {
			funcyaml:    "name: PackageVariant.my-pv.my-func.0",
			pvName:      "actually-a-different-pv",
			expectedRes: false,
		},
		"empty func name": {
			funcyaml:    "name: PackageVariant.my-pv..0",
			pvName:      "my-pv",
			expectedRes: true,
		},
		"positional location is not an int": {
			funcyaml:    "name: PackageVariant.my-pv.my-func.str",
			pvName:      "my-pv",
			expectedRes: false,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			o, err := fn.ParseKubeObject([]byte(tc.funcyaml))
			require.NoError(t, err)
			res, _ := isPackageVariantFunc(&o.SubObject, tc.pvName)

			require.Equal(t, tc.expectedRes, res)
		})
	}
}
