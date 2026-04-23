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

package kptfileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestValidateInventory tests the ValidateInventory function.
func TestValidateInventory(t *testing.T) {
	// nil inventory should not validate
	isValid, err := ValidateInventory(nil)
	if isValid || err == nil {
		t.Errorf("nil inventory should not validate")
	}
	// Empty inventory should not validate
	inv := &kptfilev1.Inventory{}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory should not validate")
	}
	// Empty inventory parameters strings should not validate
	inv = &kptfilev1.Inventory{
		Namespace:   "",
		Name:        "",
		InventoryID: "",
	}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory parameters strings should not validate")
	}
	// Inventory with non-empty namespace, name, and id should validate.
	inv = &kptfilev1.Inventory{
		Namespace:   "test-namespace",
		Name:        "test-name",
		InventoryID: "test-id",
	}
	isValid, err = ValidateInventory(inv)
	if !isValid || err != nil {
		t.Errorf("inventory with non-empty namespace, name, and id should validate")
	}
}

func TestUpdateKptfile(t *testing.T) {
	testCases := map[string]struct {
		origin         string
		updated        string
		local          string
		updateUpstream bool
		expected       string
	}{
		"no pipeline and no upstream info": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: base
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: base
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
		},

		"upstream information is not copied from upstream unless updateUpstream is true": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v1
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v2
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
		},

		"upstream information is copied from upstream when updateUpstream is true": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v1
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v2
upstreamLock:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v2
    commit: abc123
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v2
upstreamLock:
  type: git
  git:
    repo: github.com/kptdev/kpt
    directory: /
    ref: v2
    commit: abc123
`,
		},

		"pipeline in local remains if there are no changes in upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
      configMap:
        foo: bar
    - image: foo:bar
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
      configMap:
        foo: bar
    - image: foo:bar
`,
		},

		"pipeline remains if it is only added locally": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
		},

		"pipeline in local is emptied if it is gone from upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline: {}
`,
		},
		"first readinessGate and condition added in upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
  - conditionType: foo
status:
  conditions:
  - type: foo
    status: "True"
    reason: reason
    message: message
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
`,
		},
		"additional readinessGate and condition added in upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
    - conditionType: bar
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
    - type: bar
      status: "False"
      reason: reason
      message: message
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
    - conditionType: bar
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
    - type: bar
      status: "False"
      reason: reason
      message: message
		`,
		},
		"readinessGate added removed in upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info: {}
status: {}
`,
		},
		"readinessGates removed and added in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
    - conditionType: bar
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
    - type: bar
      status: "False"
      reason: reason
      message: message
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
    - conditionType: zork
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
    - type: zork
      status: "Unknown"
      reason: reason
      message: message
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: xandar
    - conditionType: foo
status:
  conditions:
    - type: xandar
      status: "True"
      reason: reason
      message: message
    - type: foo
      status: "True"
      reason: reason
      message: message  
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
info:
  readinessGates:
    - conditionType: foo
    - conditionType: zork
status:
  conditions:
    - type: foo
      status: "True"
      reason: reason
      message: message
    - type: zork
      status: Unknown
      reason: reason
      message: message
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			files := map[string]string{
				"origin":  tc.origin,
				"updated": tc.updated,
				"local":   tc.local,
			}
			dirs := make(map[string]string)
			for n, content := range files {
				dir := writeKptfileToTemp(t, content)
				dirs[n] = dir
			}

			err := UpdateKptfile(dirs["local"], dirs["updated"], dirs["origin"], tc.updateUpstream)
			require.NoError(t, err)

			c, err := os.ReadFile(filepath.Join(dirs["local"], kptfilev1.KptFileName))
			require.NoError(t, err)

			expectedObj := map[string]any{}
			err = yaml.Unmarshal([]byte(strings.TrimSpace(tc.expected)), &expectedObj)
			require.NoError(t, err)

			actualObj := map[string]any{}
			err = yaml.Unmarshal(c, &actualObj)
			require.NoError(t, err)

			assert.Equal(t, expectedObj, actualObj)
		})
	}
}

func TestUpdateKptfile_PreservesCommentsAndFormatting(t *testing.T) {
	originDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0
`)

	updatedDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.1.0
upstreamLock:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.1.0
    commit: abcdef
`)

	localDir := writeKptfileToTemp(t, `
# local package level comment
apiVersion: kpt.dev/v1 # api comment
kind: Kptfile
metadata:
  name: sample

# preserve this section comment
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0 # keep inline comment
`)

	err := UpdateKptfile(localDir, updatedDir, originDir, true)
	require.NoError(t, err)

	contentBytes, err := os.ReadFile(filepath.Join(localDir, kptfilev1.KptFileName))
	require.NoError(t, err)
	content := string(contentBytes)

	// Head/section comments and inline comments on UNCHANGED fields survive.
	assert.Contains(t, content, "# local package level comment")
	assert.Contains(t, content, "apiVersion: kpt.dev/v1 # api comment")
	assert.Contains(t, content, "# preserve this section comment")
	// Inline comments on CHANGED scalars are lost during merge3 -
	// the entire YAML node is replaced by the one from the update.
	assert.Contains(t, content, "ref: v1.1.0")
	assert.Contains(t, content, "commit: abcdef")
}

func TestUpdateKptfile_PreservesExactFormattingAndComments(t *testing.T) {
	originDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0
upstreamLock:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0
    commit: abc123
`)

	updatedDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.1.0
upstreamLock:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.1.0
    commit: def456
`)

	localDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1 # keep api inline comment
kind: Kptfile
metadata:
  name: sample
# preserve this comment block
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0 # keep ref inline comment

upstreamLock:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: package
    ref: v1.0.0
    commit: abc123 # keep commit inline comment
`)

	err := UpdateKptfile(localDir, updatedDir, originDir, true)
	require.NoError(t, err)

	contentBytes, err := os.ReadFile(filepath.Join(localDir, kptfilev1.KptFileName))
	require.NoError(t, err)

	// Verify structural preservation:
	// - Head/section comments survive
	// - Inline comments on UNCHANGED fields (apiVersion) survive
	// - Inline comments on CHANGED scalars (ref, commit) are lost
	//   because merge3 replaces the entire YAML node from the update.
	content := string(contentBytes)
	assert.Contains(t, content, "apiVersion: kpt.dev/v1 # keep api inline comment")
	assert.Contains(t, content, "# preserve this comment block")
	assert.Contains(t, content, "ref: v1.1.0")
	assert.Contains(t, content, "commit: def456")
	assert.NotContains(t, content, "ref: v1.0.0")
	assert.NotContains(t, content, "commit: abc123")
}

func TestWriteFile_ReturnsErrorWhenDirectoryMissing(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")

	err := WriteFile(nonExistentDir, DefaultKptfile("sample"))
	assert.Error(t, err)
}

func TestWriteFile_ReturnsErrorWhenPathIsFile(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "not-a-directory")
	err := os.WriteFile(filePath, []byte("content"), 0600)
	require.NoError(t, err)

	err = WriteFile(filePath, DefaultKptfile("sample"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestWriteFile_RecoversFromInvalidExistingKptfile(t *testing.T) {
	dir := t.TempDir()
	kptfilePath := filepath.Join(dir, kptfilev1.KptFileName)
	err := os.WriteFile(kptfilePath, []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata: [bad\n"), 0600)
	require.NoError(t, err)

	err = WriteFile(dir, DefaultKptfile("sample"))
	require.NoError(t, err)

	content, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "apiVersion: kpt.dev/v1")
	assert.Contains(t, string(content), "kind: Kptfile")
	assert.Contains(t, string(content), "name: sample")
}

func TestWriteFile_RoundTripIdempotency(t *testing.T) {
	original := strings.TrimSpace(`
# Package-level comment
apiVersion: kpt.dev/v1 # api version comment
kind: Kptfile
metadata:
  name: my-package
  annotations:
    example.com/team: platform
# upstream comment
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: /
    ref: v1.0.0 # pinned version
upstreamLock:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: /
    ref: v1.0.0
    commit: abc123def456
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: my-ns
info:
  description: A sample package
`)

	// Write the original content
	dir := t.TempDir()
	kptfilePath := filepath.Join(dir, kptfilev1.KptFileName)
	err := os.WriteFile(kptfilePath, []byte(original), 0600)
	require.NoError(t, err)

	// Read it back as a typed KptFile
	kf, err := ReadKptfile(filesys.FileSystemOrOnDisk{}, dir)
	require.NoError(t, err)

	// Write it via WriteFile (first write)
	err = WriteFile(dir, kf)
	require.NoError(t, err)

	firstWrite, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)

	// Write it again via WriteFile (second write)
	err = WriteFile(dir, kf)
	require.NoError(t, err)

	secondWrite, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)

	// The two writes must produce byte-identical output (idempotency)
	assert.Equal(t, string(firstWrite), string(secondWrite), "WriteFile must be idempotent")

	// Comments should be preserved
	assert.Contains(t, string(secondWrite), "# Package-level comment")
	assert.Contains(t, string(secondWrite), "# api version comment")
	assert.Contains(t, string(secondWrite), "# upstream comment")
	assert.Contains(t, string(secondWrite), "# pinned version")
}

func TestWriteKptfileToFS_PreservesFormatting(t *testing.T) {
	original := strings.TrimSpace(`
# This comment must survive
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
upstream:
  type: git
  git:
    repo: https://github.com/example/repo.git
    directory: /
    ref: v1.0.0 # inline comment
`)

	fs := filesys.MakeFsInMemory()
	dir := "/test-pkg"
	require.NoError(t, fs.MkdirAll(dir))
	require.NoError(t, fs.WriteFile(filepath.Join(dir, kptfilev1.KptFileName), []byte(original)))

	// Read, modify, and write back via WriteKptfileToFS
	kf, err := ReadKptfile(fs, dir)
	require.NoError(t, err)

	kf.Upstream.Git.Ref = "v2.0.0"
	err = WriteKptfileToFS(fs, dir, kf)
	require.NoError(t, err)

	result, err := fs.ReadFile(filepath.Join(dir, kptfilev1.KptFileName))
	require.NoError(t, err)

	content := string(result)
	// Head comments on unchanged fields survive merge3.
	assert.Contains(t, content, "# This comment must survive")
	// Inline comment on the CHANGED ref scalar is lost during merge3.
	assert.Contains(t, content, "ref: v2.0.0")
	assert.NotContains(t, content, "ref: v1.0.0")
}

func TestUpdateKptfile_ReturnsErrorOnInvalidLocalKptfile(t *testing.T) {
	originDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
`)

	updatedDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
`)

	localDir := writeKptfileToTemp(t, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata: [bad
`)

	err := UpdateKptfile(localDir, updatedDir, originDir, true)
	assert.Error(t, err)
}

func TestUpdateKptfileContent_UsesDecodeValidation(t *testing.T) {
	testCases := map[string]struct {
		content             string
		expectedErr         any
		expectedDecodeError string
	}{
		"deprecated version": {
			content: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: sample
`,
			expectedErr:         &DeprecatedKptfileError{},
			expectedDecodeError: "old resource version \"v1alpha2\" found in Kptfile",
		},
		"unknown kind": {
			content: `
apiVersion: kpt.dev/v1
kind: ConfigMap
metadata:
  name: sample
`,
			expectedErr:         &UnknownKptfileResourceError{},
			expectedDecodeError: "unknown resource type \"kpt.dev/v1, Kind=ConfigMap\" found in Kptfile",
		},
		"unknown field": {
			content: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
unexpectedField: true
`,
			expectedDecodeError: "yaml: unmarshal errors:\n  line 6: field unexpectedField not found in type v1.KptFile",
		},
		"multiple yaml documents": {
			content: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: extra
`,
			expectedDecodeError: "expected exactly one YAML document in Kptfile, found 2",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			_, decodeErr := DecodeKptfile(strings.NewReader(tc.content))
			_, updateErr := UpdateKptfileContent(tc.content, func(*kptfilev1.KptFile) {})

			if !assert.EqualError(t, decodeErr, tc.expectedDecodeError) {
				t.FailNow()
			}
			if !assert.EqualError(t, updateErr, decodeErr.Error()) {
				t.FailNow()
			}
			if tc.expectedErr != nil {
				assert.IsType(t, tc.expectedErr, decodeErr)
				assert.IsType(t, tc.expectedErr, updateErr)
			}
		})
	}
}

func TestUpdateKptfileContent_StripsSDKInternalAnnotations(t *testing.T) {
	t.Run("preserves user annotations", func(t *testing.T) {
		content := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
  annotations:
    config.kubernetes.io/index: "0"
    internal.config.kubernetes.io/path: Kptfile
    user.example.com/keep: value
`

		updatedContent, err := UpdateKptfileContent(content, func(kf *kptfilev1.KptFile) {
			kf.Name = "updated-sample"
		})
		require.NoError(t, err)

		updatedKf, err := DecodeKptfile(strings.NewReader(updatedContent))
		require.NoError(t, err)

		assert.Equal(t, "updated-sample", updatedKf.Name)
		if assert.NotNil(t, updatedKf.Annotations) {
			assert.Equal(t, "value", updatedKf.Annotations["user.example.com/keep"])
			for _, key := range sdkGeneratedKptfileAnnotations {
				assert.NotContains(t, updatedKf.Annotations, key)
			}
		}
		assert.NotContains(t, updatedContent, "config.kubernetes.io/index")
		assert.NotContains(t, updatedContent, "internal.config.kubernetes.io/path")
	})

	t.Run("removes empty annotation map", func(t *testing.T) {
		content := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
  annotations:
    config.kubernetes.io/index: "0"
    internal.config.kubernetes.io/index: "0"
`

		updatedContent, err := UpdateKptfileContent(content, func(*kptfilev1.KptFile) {})
		require.NoError(t, err)

		updatedKf, err := DecodeKptfile(strings.NewReader(updatedContent))
		require.NoError(t, err)

		// After stripping SDK annotations, the annotation map is empty.
		// merge3 may leave an empty map marker (annotations: {}) which
		// is semantically equivalent to no annotations.
		if updatedKf.Annotations != nil {
			assert.Empty(t, updatedKf.Annotations)
		}
		assert.NotContains(t, updatedContent, "config.kubernetes.io/index")
		assert.NotContains(t, updatedContent, "internal.config.kubernetes.io/index")
	})

	t.Run("handles missing annotations safely", func(t *testing.T) {
		content := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
`

		updatedContent, err := UpdateKptfileContent(content, func(kf *kptfilev1.KptFile) {
			kf.Name = "updated-sample"
		})
		require.NoError(t, err)

		updatedKf, err := DecodeKptfile(strings.NewReader(updatedContent))
		require.NoError(t, err)

		assert.Equal(t, "updated-sample", updatedKf.Name)
		assert.Nil(t, updatedKf.Annotations)
	})
}

func TestUpdateKptfileContent_ReturnsErrorOnNilMutator(t *testing.T) {
	content := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sample
`

	_, err := UpdateKptfileContent(content, nil)
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "mutator cannot be nil")
}

func TestMerge(t *testing.T) {
	testCases := map[string]struct {
		origin   string
		update   string
		local    string
		expected string
		err      error
	}{
		// With no associative key, there is no merge, just a replacement
		// of the pipeline with upstream. This is aligned with the general behavior
		// of kyaml merge where in conflicts the upstream version win.s
		"no associative key, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/folder-ref
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/folder-ref
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
		},

		"exec: no associative key, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: gen-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: folder-ref
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: folder-ref
  - exec: gen-folders
`,
		},

		"add new setter in upstream, update local setter value": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configMap:
        image: nginx
        tag: 1.0.1
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configMap:
        image: nginx
        tag: 1.0.1
        new-setter: new-setter-value // new setter is added
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configMap:
        image: nginx
        tag: 1.2.0 // value of tag is updated
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
    configMap:
      image: nginx
      new-setter: new-setter-value // new setter is added
      tag: 1.2.0 // value of tag is updated
`,
		},

		"both upstream and local configPath is updated, take upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters-updated.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters-local.yaml
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
    configPath: setters-updated.yaml
`,
		},

		"both upstream and local version is updated, take upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
    configPath: setters.yaml
`,
		},

		"newly added upstream function": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: foo
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
    configPath: setters.yaml
  - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
    configMap:
      namespace: foo
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
`,
		},

		"deleted function in the upstream, deleted on local if not changed": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configPath: setters.yaml
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: foo
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
  - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
    configPath: setters.yaml
  - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
    configMap:
      namespace: foo
`,
		},

		"multiple declarations of same function": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar-new
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        app: db
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: foo
      put-value: bar-new
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
`,
		},

		"add function at random location with name specified": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar-new
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        app: db
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: foo
      put-value: bar-new
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
`,
		},

		"Ideal deterministic behavior: add function at random location with name specified in all sources": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
      name: gf1
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      name: sl1
      configMap:
        app: db
    - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: YOUR_TEAM
      put-value: my-team
    name: my-new-function
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders:latest
    name: gf1
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: foo
      put-value: bar-new
    name: sr1
  - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
    configMap:
      app: db
    name: sl1
  - image: ghcr.io/kptdev/krm-functions-catalog/search-replace:latest
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
    name: sr2
`,
		},

		// When adding an associative key, we get a real merge of the pipeline.
		// In this case, we have an initial empty list in origin and different
		// functions are added in upstream and local. In this case the element
		// added in local are placed first in the resulting list.
		"associative key name, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: gen-folders
    image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: folder-ref
    image: ghcr.io/kptdev/krm-functions-catalog/folder-ref
`,
			// The reordering of elements in the results is a bug in the
			// merge logic I think.
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/folder-ref
    name: folder-ref
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: gen-folders
`,
		},

		// Even with multiple elements added in both upstream and local, all
		// elements from local comes before upstream, and the order of elements
		// from each source is preserved. There is no lexicographical
		// ordering.
		"associative key name, multiple additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: z-upstream
    image: z-ghcr.io/kptdev/krm-functions-catalog/generate-folders
  - name: a-upstream
    image: a-ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: x-local
    image: x-ghcr.io/kptdev/krm-functions-catalog/generate-folders
  - name: b-local
    image: b-ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: x-ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: x-local
  - image: b-ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: b-local
  - image: z-ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: z-upstream
  - image: a-ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: a-upstream
`,
		},

		// If elements with the same associative key are added in both upstream
		// and local, it will be merged. It will keep the location in the list
		// from local.
		"same element in both local and upstream does not create duplicate": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: gen-folder-upstream
    image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
  - name: ref-folders
    image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
      foo: bar
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
      bar: foo
  - name: gen-folder-local
    image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
      bar: foo
      foo: bar
    name: ref-folders
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: gen-folder-local
  - image: ghcr.io/kptdev/krm-functions-catalog/generate-folders
    name: gen-folder-upstream
`,
		},

		// If a field are set in both upstream and local, the value from
		// upstream will be chosen.
		"If there is a field-level conflict, upstream will win": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
      band: sleater-kinney
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
        band: "H\u00fcsker D\u00fc"
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: ghcr.io/kptdev/krm-functions-catalog/ref-folders
    configMap:
      band: sleater-kinney
    name: ref-folders
`,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			localKf, err := DecodeKptfile(strings.NewReader(tc.local))
			require.NoError(t, err)
			updatedKf, err := DecodeKptfile(strings.NewReader(tc.update))
			require.NoError(t, err)
			originKf, err := DecodeKptfile(strings.NewReader(tc.origin))
			require.NoError(t, err)
			err = merge(localKf, updatedKf, originKf)
			if tc.err == nil {
				require.NoError(t, err)
				actual, err := yaml.Marshal(localKf)
				require.NoError(t, err)
				if !assert.Equal(t,
					strings.TrimSpace(tc.expected), strings.TrimSpace(string(actual))) {
					t.FailNow()
				}
			} else {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, tc.err.Error(), err.Error()) {
					t.FailNow()
				}
			}
		})
	}
}

func writeKptfileToTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, kptfilev1.KptFileName), []byte(content), 0600)
	require.NoError(t, err)
	return dir
}
