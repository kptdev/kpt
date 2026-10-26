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

package cmdcat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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

// TestCmd_AnnotateDefaultOff verifies that in default mode (no --annotate),
// neither config.kubernetes.io/path nor internal.config.kubernetes.io/path leaks.
func TestCmd_AnnotateDefaultOff(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, f1Yaml), `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
`)

	got, err := runCat(t, filepath.Join(d, f1Yaml))
	require.NoError(t, err)
	assert.NotContains(t, got, "config.kubernetes.io/path",
		"config.kubernetes.io/path should be cleared by default")
	assert.NotContains(t, got, "internal.config.kubernetes.io/path",
		"internal.config.kubernetes.io/path should be cleared by default")
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
	require.NoError(t, err)
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
// This guards against path-cleaning bugs and double-traversal of subpackages.
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
	require.NoError(t, err)
	// Both packages' resources and Kptfiles should be emitted.
	assert.Equal(t, 2, strings.Count(got, "\nkind: ConfigMap\n"),
		"each pkg should be emitted exactly once")
	assert.Contains(t, got, "name: root-cm")
	assert.Contains(t, got, "name: sub-cm")
	assert.Contains(t, got, "kind: Kptfile", "Kptfile should be included in output")
}

// TestCmd_PathCleaning verifies that ./path and path/ behave identically to path.
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
			require.NoError(t, err)
			assert.Equal(t, 1, strings.Count(got, "\nkind: ConfigMap\n"),
				"arg %q should yield exactly one resource", arg)
		})
	}
}

// TestCmd_TrailingSeparator verifies that output does not end with a stray
// `---\n` separator.
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
	require.NoError(t, err)
	assert.False(t, strings.HasSuffix(got, "---\n") || strings.HasSuffix(got, "---"),
		"output must not end with a stray separator; got %q", got)
}

// TestCmd_BrokenYAMLReturnsError verifies that a broken YAML file
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
	// The returned error must carry the pkg-path context ("kpt pkg cat: "<pkg>":")
	// as well as the underlying file that failed. The root cobra handler
	// (main.handleErr) is responsible for printing; ExecuteCmd must not
	// write to stderr itself, or the user would see the message twice.
	assert.Contains(t, err.Error(), "kpt pkg cat:")
	assert.Contains(t, err.Error(), d)
	assert.Contains(t, err.Error(), "broken.yaml")

	// Sanity-check that ExecuteCmd itself does not write to the printer's
	// ErrStream — otherwise the root handler would duplicate the output.
	var errBuf bytes.Buffer
	ctx := printer.WithContext(context.Background(), printer.New(nil, &errBuf))
	r := GetCatRunner(ctx, "")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(&bytes.Buffer{})
	r.Command.SetErr(&bytes.Buffer{})
	_ = r.Command.Execute()
	assert.Empty(t, errBuf.String(),
		"ExecuteCmd must not print to ErrStream; the root handler prints the returned error")
}

// TestCmd_NonExistent verifies a clean error on a missing path.
func TestCmd_NonExistent(t *testing.T) {
	d := t.TempDir()
	_, err := runCat(t, filepath.Join(d, "nope.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestCmd_KptfileArgDisplayed: passing the Kptfile directly should display
// its content since cat now shows all package files.
func TestCmd_KptfileArgDisplayed(t *testing.T) {
	d := t.TempDir()
	kptContent := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`
	writeFile(t, filepath.Join(d, "Kptfile"), kptContent)

	got, err := runCat(t, d)
	require.NoError(t, err)
	assert.Contains(t, got, "kind: Kptfile")
	assert.Contains(t, got, "name: root")
}

// TestCmd_NonYAMLFileDisplayed: non-YAML files in a package directory
// should be displayed as raw content.
func TestCmd_NonYAMLFileDisplayed(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "README.md"), "# Hello\nThis is a readme.\n")

	got, err := runCat(t, d)
	require.NoError(t, err)
	assert.Contains(t, got, "# Hello")
	assert.Contains(t, got, "This is a readme.")
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
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.NotContains(t, got, "top comment")
	assert.NotContains(t, got, "inline comment")
}

// TestCmd_FormatTrue verifies that the default --format=true reorders fields
// to canonical Kubernetes order (apiVersion, kind, metadata, spec).
func TestCmd_FormatTrue(t *testing.T) {
	d := t.TempDir()
	// Deliberately non-canonical order: metadata before apiVersion/kind.
	writeFile(t, filepath.Join(d, "f.yaml"), `metadata:
  name: c
apiVersion: v1
kind: ConfigMap
`)

	got, err := runCat(t, filepath.Join(d, "f.yaml"))
	require.NoError(t, err)
	apiIdx := strings.Index(got, "apiVersion:")
	kindIdx := strings.Index(got, "kind:")
	metaIdx := strings.Index(got, "metadata:")
	assert.True(t, apiIdx >= 0 && kindIdx >= 0 && metaIdx >= 0, "all fields should be present")
	assert.Less(t, apiIdx, kindIdx,
		"with --format=true, apiVersion should come before kind")
	assert.Less(t, kindIdx, metaIdx,
		"with --format=true, kind should come before metadata")
}

// TestCmd_FormatFalse verifies --format=false preserves the original field
// order, even when it differs from the canonical Kubernetes ordering.
func TestCmd_FormatFalse(t *testing.T) {
	d := t.TempDir()
	// Deliberately non-canonical order: metadata before apiVersion/kind.
	writeFile(t, filepath.Join(d, "f.yaml"), `metadata:
  name: c
apiVersion: v1
kind: ConfigMap
`)

	got, err := runCat(t, filepath.Join(d, "f.yaml"), "--format=false")
	require.NoError(t, err)
	metaIdx := strings.Index(got, "metadata:")
	apiIdx := strings.Index(got, "apiVersion:")
	kindIdx := strings.Index(got, "kind:")
	assert.True(t, metaIdx >= 0 && apiIdx >= 0 && kindIdx >= 0, "all fields should be present")
	assert.Less(t, metaIdx, apiIdx,
		"with --format=false, metadata should remain before apiVersion")
	assert.Less(t, apiIdx, kindIdx,
		"with --format=false, apiVersion should remain before kind")
}

// TestCmd_SingleKptfileArg verifies that passing Kptfile as a direct file
// argument displays only the Kptfile content, not the entire package.
func TestCmd_SingleKptfileArg(t *testing.T) {
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

	got, err := runCat(t, filepath.Join(d, "Kptfile"))
	require.NoError(t, err)
	assert.Contains(t, got, "kind: Kptfile")
	assert.NotContains(t, got, "kind: ConfigMap",
		"only Kptfile should be displayed, not other package files")
}

// TestCmd_SingleNonKRMFileArg verifies that passing a non-KRM file directly
// outputs its raw content.
func TestCmd_SingleNonKRMFileArg(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "README.md"), "# Title\nSome content.\n")

	got, err := runCat(t, filepath.Join(d, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Title\nSome content.\n", got)
}

// TestCmd_EmptyDirectory verifies that an empty directory produces no output
// and no error.
func TestCmd_EmptyDirectory(t *testing.T) {
	d := t.TempDir()

	got, err := runCat(t, d)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestCmd_StyleFlag verifies that --style flag is accepted without error.
func TestCmd_StyleFlag(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "f.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`)

	got, err := runCat(t, filepath.Join(d, "f.yaml"), "--style=DoubleQuotedStyle")
	require.NoError(t, err)
	assert.Contains(t, got, "kind: ConfigMap")
	assert.Contains(t, got, "name: test")
}

// TestCmd_DirectoryOrder verifies that files are output in filesystem walk
// order, with KRM and non-KRM files interleaved correctly.
func TestCmd_DirectoryOrder(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "a.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: a-cm
`)
	writeFile(t, filepath.Join(d, "b-readme.txt"), "hello\n")
	writeFile(t, filepath.Join(d, "c.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: c-cm
`)

	got, err := runCat(t, d)
	require.NoError(t, err)
	// Verify all content is present.
	assert.Contains(t, got, "kind: Kptfile")
	assert.Contains(t, got, "name: a-cm")
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "name: c-cm")
	// Verify order: Kptfile < a.yaml < b-readme.txt < c.yaml (alphabetical walk).
	kptIdx := strings.Index(got, "kind: Kptfile")
	aIdx := strings.Index(got, "name: a-cm")
	bIdx := strings.Index(got, "hello")
	cIdx := strings.Index(got, "name: c-cm")
	assert.Less(t, kptIdx, aIdx, "Kptfile should appear before a.yaml")
	assert.Less(t, aIdx, bIdx, "a.yaml should appear before b-readme.txt")
	assert.Less(t, bIdx, cIdx, "b-readme.txt should appear before c.yaml")
}

// TestCmd_SingleYAMLFileArg exercises the code path where LocalPackageReader
// is given a single file path (not a directory) with an empty PackageFileName.
// This guards against kyaml tightening validation on that usage.
func TestCmd_SingleYAMLFileArg(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "deploy.yaml"), `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 1
`)

	got, err := runCat(t, filepath.Join(d, "deploy.yaml"))
	require.NoError(t, err)
	assert.Contains(t, got, "kind: Deployment")
	assert.Contains(t, got, "name: nginx")
	assert.Contains(t, got, "replicas:")
}

// TestCmd_ContextCancellation verifies that a cancelled context aborts the walk.
func TestCmd_ContextCancellation(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(d, "a.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`)

	ctx, cancel := context.WithCancel(
		printer.WithContext(context.Background(), printer.New(nil, io.Discard)),
	)
	cancel()
	defer cancel()

	r := GetCatRunner(ctx, "")
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{d})
	out := &bytes.Buffer{}
	r.Command.SetOut(out)
	r.Command.SetErr(&bytes.Buffer{})

	err := r.Command.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String(), "no output should be produced when ctx is cancelled before walk")
}

// TestCmd_ContextCancellation_Deadline verifies DeadlineExceeded behaves the same.
func TestCmd_ContextCancellation_Deadline(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)

	ctx, cancel := context.WithTimeout(
		printer.WithContext(context.Background(), printer.New(nil, io.Discard)), 0,
	)
	defer cancel()

	r := GetCatRunner(ctx, "")
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(&bytes.Buffer{})
	r.Command.SetErr(&bytes.Buffer{})
	err := r.Command.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestCmd_ContextCancellation_SingleFile verifies the single-file path honors cancellation.
func TestCmd_ContextCancellation_SingleFile(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "a.yaml")
	writeFile(t, f, `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`)

	ctx, cancel := context.WithCancel(
		printer.WithContext(context.Background(), printer.New(nil, io.Discard)),
	)
	cancel()
	defer cancel()

	r := GetCatRunner(ctx, "")
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{f})
	r.Command.SetOut(&bytes.Buffer{})
	r.Command.SetErr(&bytes.Buffer{})
	err := r.Command.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestCmd_ContextCancellation_MidWalk verifies that cancellation during a walk
// stops work before all files are processed.
func TestCmd_ContextCancellation_MidWalk(t *testing.T) {
	d := t.TempDir()
	writeFile(t, filepath.Join(d, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	for i := range 500 {
		writeFile(t, filepath.Join(d, fmt.Sprintf("f%03d.yaml", i)),
			fmt.Sprintf("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm-%d\n", i))
	}

	ctx, cancel := context.WithCancel(
		printer.WithContext(context.Background(), printer.New(nil, io.Discard)),
	)
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	r := GetCatRunner(ctx, "")
	r.Command.SilenceUsage = true
	out := &bytes.Buffer{}
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(out)
	r.Command.SetErr(&bytes.Buffer{})
	err := r.Command.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String(), "no output should leak on cancellation")
}

// TestCmd_SymlinkArg verifies that a symlink argument is resolved and its
// target content is displayed, while symlinks inside the package are skipped.
func TestCmd_SymlinkArg(t *testing.T) {
	d := t.TempDir()
	real := filepath.Join(d, "real")
	require.NoError(t, os.MkdirAll(real, 0o755))
	writeFile(t, filepath.Join(real, "Kptfile"), `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root
`)
	writeFile(t, filepath.Join(real, "cm.yaml"), `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
`)
	// Create a file that a symlink inside the package points to.
	writeFile(t, filepath.Join(d, "external.yaml"), `apiVersion: v1
kind: Secret
metadata:
  name: secret
`)
	// Symlink inside the package — should be skipped.
	require.NoError(t, os.Symlink(filepath.Join(d, "external.yaml"), filepath.Join(real, "link.yaml")))

	// Symlink as the argument — should be resolved.
	link := filepath.Join(d, "pkg-link")
	require.NoError(t, os.Symlink(real, link))

	got, err := runCat(t, link)
	require.NoError(t, err)
	assert.Contains(t, got, "name: cm", "target content should be displayed")
	assert.Contains(t, got, "kind: Kptfile", "Kptfile should be displayed")
	assert.NotContains(t, got, "kind: Secret", "symlinked file inside package should be skipped")
}
