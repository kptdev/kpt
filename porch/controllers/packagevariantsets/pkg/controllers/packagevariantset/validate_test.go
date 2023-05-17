// Copyright 2023 The kpt Authors
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

package packagevariantset

import (
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestValidatePackageVariantSet(t *testing.T) {
	packageVariantHeader := `apiVersion: config.porch.kpt.dev
kind: PackageVariantSet
metadata:
  name: my-pv`

	testCases := map[string]struct {
		packageVariant string
		expectedErrs   []string
	}{
		"empty spec": {
			packageVariant: packageVariantHeader,
			expectedErrs: []string{"spec.upstream is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream package": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    repo: foo
    revision: v1`,
			expectedErrs: []string{"spec.upstream.package is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream repo": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foopkg
    revision: v3`,
			expectedErrs: []string{"spec.upstream.repo is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream revision": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    repo: foo
    package: foopkg`,
			expectedErrs: []string{"spec.upstream.revision is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"invalid targets": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - repositories:
    - name: ""
  - repositories:
    - name: bar
    repositorySelector:
      foo: bar
  - repositories:
    - name: bar
      packageNames:
      - ""
      - foo
      `,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0].repositories[0].name cannot be empty",
				"spec.targets[1] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[2].repositories[0].packageNames[0] cannot be empty",
			},
		},
		"invalid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      adoptionPolicy: invalid
      deletionPolicy: invalid
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template.adoptionPolicy can only be \"adoptNone\" or \"adoptExisting\"",
				"spec.targets[0].template.deletionPolicy can only be \"orphan\" or \"delete\"",
			},
		},
		"valid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  adoptionPolicy: adoptExisting
  deletionPolicy: orphan
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"downstream values and expressions do not mix": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      downstream:
        repo: "foo"
        repoExpr: "'bar'"
        package: "p"
        packageExpr: "'p'"
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template may specify only one of `downstream.repo` and `downstream.repoExpr`",
				"spec.targets[0].template may specify only one of `downstream.package` and `downstream.packageExpr`",
			},
		},
		"MapExprs do not allow both expr-and non-expr for same field": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      labelExprs:
      - key: "foo"
        keyExpr: "'bar'"
        value: "bar"
      - key: "foo"
        value: "bar"
        valueExpr: "'bar'"
      annotationExprs:
      - key: "foo"
        keyExpr: "'bar'"
        value: "bar"
      - key: "foo"
        value: "bar"
        valueExpr: "'bar'"
      packageContext:
        dataExprs:
          - key: "foo"
            keyExpr: "'bar'"
            value: "bar"
          - key: "foo"
            value: "bar"
            valueExpr: "'bar'"
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template.labelExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.labelExprs[1] may specify only one of `value` and `valueExpr`",
				"spec.targets[0].template.annotationExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.annotationExprs[1] may specify only one of `value` and `valueExpr`",
				"spec.targets[0].template.packageContext.dataExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.packageContext.dataExprs[1] may specify only one of `value` and `valueExpr`",
			},
		},
		"injectors must specify exactly one of name or nameexpr": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - repositories:
    - name: bar
    template:
      injectors:
      - name: foo
        nameExpr: bar
      - group: foo
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0].template.injectors[0] may specify only one of `name` and `nameExpr`",
				"spec.targets[0].template.injectors[1] must specify either `name` or `nameExpr`",
			},
		},
		"pipeline function must be valid": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - repositories:
    - name: bar
    template:
      pipeline:
        validators:
        - name: foo
        - image: foo
          name: bar
        mutators:
        - name: foo.bar
          image: bar
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0].template.pipeline.validators[0].image must not be empty",
				"spec.targets[0].template.pipeline.mutators[0].name must not contain '.'",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pvs api.PackageVariantSet
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageVariant), &pvs))
			actualErrs := validatePackageVariantSet(&pvs)
			require.Equal(t, len(tc.expectedErrs), len(actualErrs))
			for i := range actualErrs {
				require.EqualError(t, actualErrs[i], tc.expectedErrs[i])
			}

		})
	}
}
