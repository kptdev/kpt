// Copyright 2019,2026 The kpt Authors.
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

package cmdtree

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeCommandDefaultCurDir_files(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}
	revert := testutil.Chdir(t, d)
	defer revert()

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: ghcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, b), "")
	r.Command.SetArgs([]string{})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, filepath.Base(d)), b.String()) {
		return
	}
}

func TestTreeCommand_files(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: ghcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, filepath.Base(d)), b.String()) {
		return
	}
}

func TestTreeCommand_Kustomization(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	err = os.WriteFile(filepath.Join(d, "Kustomization"), []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- f2.yaml
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f2.yaml]  Deployment bar
└── Kustomization
`, filepath.Base(d)), b.String()) {
		return
	}
}

func TestTreeCommand_subpkgs(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.MkdirAll(filepath.Join(d, "subpkg"), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: ghcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "subpkg", "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	err = os.WriteFile(filepath.Join(d, "Kptfile"), []byte(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mainpkg
openAPI:
  definitions:
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "subpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: subpkg
openAPI:
  definitions:
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`Package %q
├── [Kptfile]  Kptfile mainpkg
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── Package "subpkg"
    ├── [Kptfile]  Kptfile subpkg
    └── [f2.yaml]  Deployment bar
`, filepath.Base(d)), b.String()) {
		return
	}
}

func TestTreeCommand_CurDirInput(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.MkdirAll(filepath.Join(d, "Mainpkg", "Subpkg"), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	revert := testutil.Chdir(t, filepath.Join(d, "Mainpkg"))
	defer revert()

	err = os.WriteFile(filepath.Join(d, "Mainpkg", "f1.yaml"), []byte(`
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "Mainpkg", "Subpkg", "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	err = os.WriteFile(filepath.Join(d, "Mainpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: Mainpkg
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "Mainpkg", "Subpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: Subpkg
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `Package "Mainpkg"
├── [Kptfile]  Kptfile Mainpkg
├── [f1.yaml]  Deployment foo
└── Package "Subpkg"
    ├── [Kptfile]  Kptfile Subpkg
    └── [f2.yaml]  Deployment bar
`, b.String()) {
		return
	}
}

func TestTreeCommand_symlink(t *testing.T) {
	d, err := os.MkdirTemp("", "tree-test")
	if !assert.NoError(t, err) {
		return
	}
	revert := testutil.Chdir(t, d)
	defer revert()
	err = os.MkdirAll(filepath.Join(d, "foo"), 0700)
	assert.NoError(t, err)
	err = os.Symlink("foo", "foo-link")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)
	err = os.WriteFile(filepath.Join(d, "foo", "f1.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: ghcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "foo", "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, stderr), "")
	r.Command.SetArgs([]string{filepath.Join(d, "foo-link")})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `foo-link
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, b.String()) {
		return
	}
	assert.Contains(t, stderr.String(), "please note that the symlinks within the package are ignored")
}

// TestTreeCommand_NonKRMInSubpackage verifies non-KRM files in a subpackage
// appear under the subpackage branch, not the parent.
func TestTreeCommand_NonKRMInSubpackage(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(d, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(d, "sub", "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: sub\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "sub", "NOTES.txt"), []byte("hello\n"), 0600))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	require.Contains(t, out, `Package "sub"`)
	require.Contains(t, out, "NOTES.txt")
	subIdx := strings.Index(out, `Package "sub"`)
	notesIdx := strings.Index(out, "NOTES.txt")
	assert.Greater(t, notesIdx, subIdx, "NOTES.txt should be under the subpackage branch")
}

// TestTreeCommand_DotfilesExcluded verifies dotfiles and dot-dirs are excluded.
func TestTreeCommand_DotfilesExcluded(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, ".hidden"), []byte("secret\n"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(d, ".git"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(d, ".git", "config"), []byte("[core]\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "visible.txt"), []byte("hi\n"), 0600))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	assert.Contains(t, out, "visible.txt")
	assert.NotContains(t, out, ".hidden")
	assert.NotContains(t, out, ".git")
	assert.NotContains(t, out, "config")
}

// TestTreeCommand_SymlinkFileSkipped verifies symlinked files inside a package are skipped.
func TestTreeCommand_SymlinkFileSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "real.txt"), []byte("content\n"), 0600))
	require.NoError(t, os.Symlink(filepath.Join(d, "real.txt"), filepath.Join(d, "link.txt")))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	assert.Contains(t, out, "real.txt")
	assert.NotContains(t, out, "link.txt")
}

// TestTreeCommand_MultipleNonKRMSorted verifies multiple non-KRM files are sorted.
func TestTreeCommand_MultipleNonKRMSorted(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "zebra.md"), []byte("z\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "alpha.txt"), []byte("a\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "middle.log"), []byte("m\n"), 0600))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	alphaIdx := strings.Index(out, "alpha.txt")
	middleIdx := strings.Index(out, "middle.log")
	zebraIdx := strings.Index(out, "zebra.md")
	require.NotEqual(t, -1, alphaIdx, "alpha.txt should be present in output")
	require.NotEqual(t, -1, middleIdx, "middle.log should be present in output")
	require.NotEqual(t, -1, zebraIdx, "zebra.md should be present in output")
	assert.Less(t, alphaIdx, middleIdx, "alpha.txt should come before middle.log")
	assert.Less(t, middleIdx, zebraIdx, "middle.log should come before zebra.md")
}

// TestTreeCommand_NonKRMInNonPackageSubdir verifies that non-KRM files inside
// a non-package subdirectory (no Kptfile) are rendered under the parent package
// branch (not as a spurious directory branch), and KRM files in that subdir are
// deduplicated properly.
func TestTreeCommand_NonKRMInNonPackageSubdir(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "deployment.yaml"), []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: root-deploy\n"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(d, "docs"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(d, "docs", "README.md"), []byte("# Hello\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "docs", "svc.yaml"), []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: my-svc\n"), 0600))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	// KRM file in subdir should appear as a resource under the "docs" branch
	assert.Contains(t, out, "[svc.yaml]  Service my-svc")
	// Non-KRM file should appear under the same "docs" branch
	assert.Contains(t, out, "README.md")
	// "docs" should NOT appear as a Package branch (no Kptfile)
	assert.NotContains(t, out, `Package "docs"`)
	// svc.yaml should appear only once (as KRM, not duplicated as non-KRM)
	assert.Equal(t, 1, strings.Count(out, "svc.yaml"), "svc.yaml should appear exactly once")
}

// TestTreeCommand_DedupKRMFile verifies a YAML file rendered as KRM is not
// duplicated in the non-KRM list.
func TestTreeCommand_DedupKRMFile(t *testing.T) {
	d := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(d, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: root\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(d, "cm.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\n"), 0600))

	b := &bytes.Buffer{}
	r := GetTreeRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	require.NoError(t, r.Command.Execute())

	out := b.String()
	assert.Contains(t, out, "[cm.yaml]  ConfigMap cfg")
	assert.Equal(t, 1, strings.Count(out, "cm.yaml"), "cm.yaml should appear exactly once (as KRM, not duplicated)")
}
