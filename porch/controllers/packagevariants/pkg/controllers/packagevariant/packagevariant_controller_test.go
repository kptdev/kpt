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
			expectedErr:    "[spec.upstream: Invalid value: \"{}\": missing required field, spec.downstream: Invalid value: \"{}\": missing required field]",
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
			expectedErr: "[spec.upstream.package: Invalid value: \"\": missing required field, spec.downstream.package: Invalid value: \"\": missing required field]",
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
			expectedErr: "[spec.adoptionPolicy: Invalid value: \"invalid\": field can only be \"adoptNone\" or \"adoptExisting\", spec.deletionPolicy: Invalid value: \"invalid\": field can only be \"orphan\" or \"delete\"]",
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
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageVariant), &pv))
			actualErr := validatePackageVariant(&pv).ToAggregate()
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}
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
