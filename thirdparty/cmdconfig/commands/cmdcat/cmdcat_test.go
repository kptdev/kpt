// Copyright 2019-2026 The kpt Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdcat

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
)

const f1Yaml = "f1.yaml"

// writeFile is a small helper to keep the test bodies terse.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// runCat invokes `kpt pkg cat` with the given args and captures stdout.
func runCat(t *testing.T, args ...string) (string, error) {
	t.Helper()
	b := &bytes.Buffer{}
	ctx := printer.WithContext(context.Background(), printer.New(nil, io.Discard))
	r := GetCatRunner(ctx, "")
	r.Command.SetArgs(args)
	r.Command.SetOut(b)
	r.Command.SetErr(&bytes.Buffer{})
	err := r.Command.Execute()
	return b.String(), err
}

// TestCmd_DIR covers the basic directory case with two multi-doc YAML files.
func TestCmd_DIR(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, f1Yaml), `
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
`)
	writeFile(t, filepath.Join(d, "f2.yaml"), `
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: gcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`)

	got, err := runCat(t, d)
	assert.NoError(t, err)
	assert.Equal(t, `kind: Deployment
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
---
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/local-config: "true"
  configFn:
    container:
      image: gcr.io/example/reconciler:v1
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bar
  labels:
    app: nginx
  annotations:
    app: nginx
spec:
  replicas: 3
`, got)
}

// TestCmd_File covers the single-file-arg code path.
func TestCmd_File(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, f1Yaml), `
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`)

	got, err := runCat(t, filepath.Join(d, f1Yaml))
	assert.NoError(t, err)
	assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`, got)
}

// TestCmd_Annotate verifies --annotate emits all four annotations
// (config.*/path, config.*/index, internal.*/path, internal.*/index).
func TestCmd_Annotate(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, f1Yaml), `
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`)

	got, err := runCat(t, filepath.Join(d, f1Yaml), "--annotate")
	assert.NoError(t, err)
	assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
    config.kubernetes.io/index: '0'
    config.kubernetes.io/path: 'f1.yaml'
    internal.config.kubernetes.io/index: '0'
    internal.config.kubernetes.io/path: 'f1.yaml'
spec:
  replicas: 1
`, got)
}

// TestCmd_AnnotateDefaultOff verifies Issue 8: in default mode, neither
// config.kubernetes.io/path nor internal.config.kubernetes.io/path leaks.
func TestCmd_AnnotateDefaultOff(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, f1Yaml), `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
`)

	got, err := runCat(t, filepath.Join(d, f1Yaml))
	assert.NoError(t, err)
	assert.NotContains(t, got, "config.kubernetes.io/path",
		"config.kubernetes.io/path should be cleared by default")
	assert.NotContains(t, got, "internal.config.kubernetes.io/path",
		"internal.config.kubernetes.io/path should be cleared by default (Issue 8)")
}

// TestCmd_Subpkgs covers a directory with a regular sub-directory (no Kptfile).
// Resources in the sub-dir are emitted as part of the root package.
func TestCmd_Subpkgs(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "subpkg"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d, f1Yaml), `
kind: Deployment
metadata:
  labels:
    app: nginx1
  name: foo
  annotations:
    app: nginx1
spec:
  replicas: 1
`)
	writeFile(t, filepath.Join(d, "subpkg", "f2.yaml"), `
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`)

	got, err := runCat(t, d)
	assert.NoError(t, err)
	assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx1
  name: foo
  annotations:
    app: nginx1
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
`, got)
}

// TestCmd_NestedPackages exercises a true multi-pkg tree (Kptfile in root and
// in subdir) and asserts each pkg's resource is emitted exactly once.
// This guards Issue 3 (path-clean) and Issue 9 (no double-traverse).
func TestCmd_NestedPackages(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "root.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: root-cm
`)
	if err := os.MkdirAll(filepath.Join(d, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d, "sub", "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub
`)
	writeFile(t, filepath.Join(d, "sub", "sub.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: sub-cm
`)

	got, err := runCat(t, d)
	assert.NoError(t, err)
	// exactly one of each resource, separated by a single ---.
	assert.Equal(t, 2, strings.Count(got, "\nkind: ConfigMap\n"),
		"each pkg should be emitted exactly once")
	assert.Equal(t, 1, strings.Count(got, "\n---\n"),
		"a single separator between the two packages")
	assert.Contains(t, got, "name: root-cm")
	assert.Contains(t, got, "name: sub-cm")
}

// TestCmd_PathCleaning (Issue 3): ./path and path/ must behave like path.
func TestCmd_PathCleaning(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "cm.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`)

	for _, arg := range []string{d, d + "/", "./" + filepath.Base(d)} {
		t.Run(arg, func(t *testing.T) {
			// Run from d's parent so "./<name>" resolves.
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(filepath.Dir(d)); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Chdir(oldWd); err != nil {
					t.Fatal(err)
				}
			}()

			got, err := runCat(t, arg)
			assert.NoError(t, err)
			assert.Equal(t, 1, strings.Count(got, "\nkind: ConfigMap\n"),
				"arg %q should yield exactly one resource", arg)
		})
	}
}

// TestCmd_TrailingSeparator (Issue 4): a pkg followed by an empty sub-pkg
// must not leave a stray `---\n` at the end of output.
func TestCmd_TrailingSeparator(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "dep.yaml"), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: d
`)
	if err := os.MkdirAll(filepath.Join(d, "sub-empty"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d, "sub-empty", "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub-empty
`)

	got, err := runCat(t, d)
	assert.NoError(t, err)
	assert.False(t, strings.HasSuffix(got, "---\n") || strings.HasSuffix(got, "---"),
		"output must not end with a stray separator; got %q", got)
	assert.Equal(t, 0, strings.Count(got, "\n---\n"),
		"no inter-pkg separator expected when only one pkg has resources")
}

// TestCmd_BrokenYAMLReturnsError (Issue 5, narrowed): a broken YAML file
// must cause a non-nil error; the command must not silently succeed.
func TestCmd_BrokenYAMLReturnsError(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "broken.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: x
data:
  a: [unterminated
`)

	_, err := runCat(t, d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broken.yaml")

	// Stderr must carry the contextual diagnostic. Inject a printer via
	// context (mirroring how run.go wires the root cobra err stream) and
	// assert on its err stream.
	var errBuf bytes.Buffer
	ctx := printer.WithContext(context.Background(), printer.New(nil, &errBuf))
	r := GetCatRunner(ctx, "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(&bytes.Buffer{})
	r.Command.SetErr(&bytes.Buffer{})
	_ = r.Command.Execute()
	assert.Contains(t, errBuf.String(), "kpt pkg cat:")
	assert.Contains(t, errBuf.String(), "broken.yaml")
}

// TestCmd_NonExistent verifies a clean error on a missing path.
func TestCmd_NonExistent(t *testing.T) {
	d := t.TempDir()
	_, err := runCat(t, filepath.Join(d, "nope.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestCmd_KptfileArgRejected: passing the Kptfile directly must error.
// The Kptfile is package metadata, not a resource; the directory form of
// `kpt pkg cat` excludes it from output, so the file form must too.
func TestCmd_KptfileArgRejected(t *testing.T) {
	d := t.TempDir()
	kpt := filepath.Join(d, "Kptfile")
	writeFile(t, kpt, `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	_, err := runCat(t, kpt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Kptfile")
	assert.Contains(t, err.Error(), "metadata")
}

// TestCmd_NonYAMLFileErrors: a regular non-YAML/JSON/Kptfile file must
// fail loudly rather than silently succeed with no output.
func TestCmd_NonYAMLFileErrors(t *testing.T) {
	d := t.TempDir()
	md := filepath.Join(d, "README.md")
	writeFile(t, md, "# hi\n")
	_, err := runCat(t, md)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a YAML/JSON file")
}

// TestCmd_RecurseFalse ensures -R=false limits traversal to the root pkg.
func TestCmd_RecurseFalse(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "root.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: root-cm
`)
	if err := os.MkdirAll(filepath.Join(d, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d, "sub", "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub
`)
	writeFile(t, filepath.Join(d, "sub", "sub.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: sub-cm
`)

	got, err := runCat(t, d, "-R=false")
	assert.NoError(t, err)
	assert.Contains(t, got, "name: root-cm")
	assert.NotContains(t, got, "name: sub-cm",
		"sub-pkg should not be traversed when -R=false")
}

// TestCmd_StripComments verifies the --strip-comments flag removes comments.
func TestCmd_StripComments(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "f.yaml"), `# top comment
apiVersion: v1
kind: ConfigMap
metadata:
  name: c  # inline comment
`)

	got, err := runCat(t, filepath.Join(d, "f.yaml"), "--strip-comments")
	assert.NoError(t, err)
	assert.NotContains(t, got, "top comment")
	assert.NotContains(t, got, "inline comment")
}

// TestCmd_FormatFalse verifies --format=false preserves original map key
// order (Issue 6 — documented as won't-fix, but the flag must work).
func TestCmd_FormatFalse(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "f.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: c
data:
  z-last: "1"
  a-first: "2"
`)

	got, err := runCat(t, filepath.Join(d, "f.yaml"), "--format=false")
	assert.NoError(t, err)
	zIdx := strings.Index(got, "z-last")
	aIdx := strings.Index(got, "a-first")
	assert.True(t, zIdx >= 0 && aIdx >= 0, "both keys should be present")
	assert.Less(t, zIdx, aIdx,
		"with --format=false, original order (z-last before a-first) must be preserved")
}
