// Copyright 2026 The kpt Authors
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

package kptops

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/api/runneroptions"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// writeKptfile writes a minimal Kptfile at dir, optionally with the
// AutoIgnoreNonKRMAnnotation set to "true" when optedIn is true, and
// optionally with a single mutator step when withMutator is true (using
// mockFnRuntime, so no real container/exec ever runs).
func writeKptfile(t *testing.T, fsys filesys.FileSystem, dir, name string, optedIn, withMutator bool) {
	t.Helper()
	annotations := ""
	if optedIn {
		annotations = "  annotations:\n    " + AutoIgnoreNonKRMAnnotation + ": \"true\"\n"
	}
	pipeline := ""
	if withMutator {
		pipeline = "pipeline:\n  mutators:\n    - image: example.com/mutate:v1\n"
	}
	content := "apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: " + name + "\n" + annotations + pipeline
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "Kptfile"), []byte(content)))
}

func writeValidKRM(t *testing.T, fsys filesys.FileSystem, dir, resourceName string) {
	t.Helper()
	content := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + resourceName + "\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "cm.yaml"), []byte(content)))
}

// writeNonKRM writes a YAML file with no apiVersion/kind/metadata.name.
func writeNonKRM(t *testing.T, fsys filesys.FileSystem, dir, fileName string) {
	t.Helper()
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, fileName), []byte("not: krm\njust: [a, plain, map]\nwithout: apiVersion-or-kind\n")))
}

// newAutoignoreTestRenderer builds a Renderer rooted at "/root", the
// path every test in this file uses.
func newAutoignoreTestRenderer(fsys filesys.FileSystem, outBuf *bytes.Buffer, mock *mockFnRuntime) *Renderer {
	opts := runneroptions.RunnerOptions{}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	return &Renderer{
		PkgPath:       "/root",
		FileSystem:    fsys,
		Output:        outBuf,
		Runtime:       mock,
		RunnerOptions: opts,
	}
}

// TestNonKRM_NoAnnotation_BlocksEntireRender verifies a non-KRM file
// with no opt-in annotation fails the whole render, and that package's
// mutator never runs at all.
func TestNonKRM_NoAnnotation_BlocksEntireRender(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	root := "/root"
	assert.NoError(t, fsys.Mkdir(root))
	writeKptfile(t, fsys, root, "root-pkg", false, true)
	writeValidKRM(t, fsys, root, "root-cm")
	writeNonKRM(t, fsys, root, "notes.yaml")

	var out bytes.Buffer
	mock := &mockFnRuntime{}
	ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
	r := newAutoignoreTestRenderer(fsys, &out, mock)

	_, err := r.Execute(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input resource list must contain only KRM resources")
	assert.Contains(t, err.Error(), "notes.yaml")
	// The mutator must never have been invoked.
	assert.Empty(t, mock.capturedItems, "mutator ran even though the package had a non-KRM file with no opt-in annotation")
}

// TestNonKRM_ConsolidatedError_ListsEveryFile verifies non-KRM files
// across multiple packages are all reported in a single error.
func TestNonKRM_ConsolidatedError_ListsEveryFile(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	root := "/root"
	sub := "/root/subpkg"
	assert.NoError(t, fsys.Mkdir(root))
	assert.NoError(t, fsys.Mkdir(sub))
	writeKptfile(t, fsys, root, "root-pkg", false, false)
	writeValidKRM(t, fsys, root, "root-cm")
	writeNonKRM(t, fsys, root, "root-notes.yaml")
	writeKptfile(t, fsys, sub, "sub-pkg", false, false)
	writeValidKRM(t, fsys, sub, "sub-cm")
	writeNonKRM(t, fsys, sub, "sub-notes.yaml")

	var out bytes.Buffer
	mock := &mockFnRuntime{}
	ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
	r := newAutoignoreTestRenderer(fsys, &out, mock)

	_, err := r.Execute(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root-notes.yaml")
	assert.Contains(t, err.Error(), "sub-notes.yaml")
	assert.Contains(t, err.Error(), "2 file(s)")
}

// TestNonKRM_AnnotatedPackage_SkipsInMemory_NoDiskWrite verifies an
// opted-in package renders successfully with the non-KRM file excluded
// from the pipeline's input and left untouched on disk.
func TestNonKRM_AnnotatedPackage_SkipsInMemory_NoDiskWrite(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	root := "/root"
	assert.NoError(t, fsys.Mkdir(root))
	writeKptfile(t, fsys, root, "root-pkg", true, true)
	writeValidKRM(t, fsys, root, "root-cm")
	writeNonKRM(t, fsys, root, "notes.yaml")

	var out bytes.Buffer
	mock := &mockFnRuntime{}
	ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
	r := newAutoignoreTestRenderer(fsys, &out, mock)

	_, err := r.Execute(ctx)

	assert.NoError(t, err)
	// The mutator ran, and the only items it saw were the Kptfile itself
	// and the valid ConfigMap -- the non-KRM file never reached the
	// pipeline (LocalResources() always includes the package's own
	// Kptfile as a KRM resource; that's expected, unrelated behavior).
	assert.Len(t, mock.capturedItems, 1)
	assert.ElementsMatch(t, []string{"root-pkg", "root-cm"}, mock.capturedItems[0])

	// The non-KRM file is untouched on disk.
	notesContent, readErr := fsys.ReadFile(filepath.Join(root, "notes.yaml"))
	assert.NoError(t, readErr)
	assert.Contains(t, string(notesContent), "not: krm")

	// .krmignore was never created.
	assert.False(t, fsys.Exists(filepath.Join(root, ".krmignore")), ".krmignore should never be written by the annotation path")

	// The render output mentions the skip.
	assert.Contains(t, out.String(), "notes.yaml")
	assert.Contains(t, out.String(), AutoIgnoreNonKRMAnnotation)
}

// TestNonKRM_AnnotationNotInherited verifies the annotation is scoped
// to the package whose own Kptfile carries it, not inherited.
func TestNonKRM_AnnotationNotInherited(t *testing.T) {
	t.Run("parent annotation does not cover child's non-KRM file", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/subpkg"
		assert.NoError(t, fsys.Mkdir(root))
		assert.NoError(t, fsys.Mkdir(sub))
		writeKptfile(t, fsys, root, "root-pkg", true, false) // opted in
		writeValidKRM(t, fsys, root, "root-cm")
		writeKptfile(t, fsys, sub, "sub-pkg", false, false) // NOT opted in
		writeValidKRM(t, fsys, sub, "sub-cm")
		writeNonKRM(t, fsys, sub, "sub-notes.yaml")

		var out bytes.Buffer
		mock := &mockFnRuntime{}
		ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
		r := newAutoignoreTestRenderer(fsys, &out, mock)

		_, err := r.Execute(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sub-notes.yaml")
	})

	t.Run("child annotation does not cover parent's non-KRM file", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/subpkg"
		assert.NoError(t, fsys.Mkdir(root))
		assert.NoError(t, fsys.Mkdir(sub))
		writeKptfile(t, fsys, root, "root-pkg", false, false) // NOT opted in
		writeValidKRM(t, fsys, root, "root-cm")
		writeNonKRM(t, fsys, root, "root-notes.yaml")
		writeKptfile(t, fsys, sub, "sub-pkg", true, false) // opted in
		writeValidKRM(t, fsys, sub, "sub-cm")

		var out bytes.Buffer
		mock := &mockFnRuntime{}
		ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
		r := newAutoignoreTestRenderer(fsys, &out, mock)

		_, err := r.Execute(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "root-notes.yaml")
	})

	t.Run("only the annotated package's non-KRM file is excused, others still block", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/subpkg"
		assert.NoError(t, fsys.Mkdir(root))
		assert.NoError(t, fsys.Mkdir(sub))
		writeKptfile(t, fsys, root, "root-pkg", false, false) // NOT opted in
		writeValidKRM(t, fsys, root, "root-cm")
		writeNonKRM(t, fsys, root, "root-notes.yaml")
		writeKptfile(t, fsys, sub, "sub-pkg", true, false) // opted in
		writeValidKRM(t, fsys, sub, "sub-cm")
		writeNonKRM(t, fsys, sub, "sub-notes.yaml")

		var out bytes.Buffer
		mock := &mockFnRuntime{}
		ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
		r := newAutoignoreTestRenderer(fsys, &out, mock)

		_, err := r.Execute(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "root-notes.yaml")
		// Only one file should be blocking -- sub-notes.yaml's package opted in.
		assert.NotContains(t, err.Error(), "sub-notes.yaml")
	})
}

// TestNonKRM_CleanTree_NoOverheadOnOutcome verifies a tree with no
// non-KRM files renders exactly as before.
func TestNonKRM_CleanTree_NoOverheadOnOutcome(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	root := "/root"
	assert.NoError(t, fsys.Mkdir(root))
	writeKptfile(t, fsys, root, "root-pkg", false, true)
	writeValidKRM(t, fsys, root, "root-cm")

	var out bytes.Buffer
	mock := &mockFnRuntime{}
	ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
	r := newAutoignoreTestRenderer(fsys, &out, mock)

	_, err := r.Execute(ctx)

	assert.NoError(t, err)
	assert.Len(t, mock.capturedItems, 1)
	assert.ElementsMatch(t, []string{"root-pkg", "root-cm"}, mock.capturedItems[0])
	assert.NotContains(t, out.String(), AutoIgnoreNonKRMAnnotation)
}

// TestNonKRM_AnnotatedPackage_RecordsSkipInKptfileCondition verifies a
// successful, annotation-excused render leaves a message naming the
// skipped file in the Kptfile's status condition.
func TestNonKRM_AnnotatedPackage_RecordsSkipInKptfileCondition(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	root := "/root"
	assert.NoError(t, fsys.Mkdir(root))
	writeKptfile(t, fsys, root, "root-pkg", true, true)
	writeValidKRM(t, fsys, root, "root-cm")
	writeNonKRM(t, fsys, root, "notes.yaml")

	var out bytes.Buffer
	mock := &mockFnRuntime{}
	ctx := printer.WithContext(context.Background(), printer.New(&out, &out))
	opts := runneroptions.RunnerOptions{}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	// Output deliberately nil: the in-place path, the only one that
	// writes status.conditions.
	r := &Renderer{
		PkgPath:       root,
		FileSystem:    fsys,
		Runtime:       mock,
		RunnerOptions: opts,
	}

	_, err := r.Execute(ctx)
	assert.NoError(t, err)

	rootKf, readErr := kptfileutil.ReadKptfile(fsys, root)
	assert.NoError(t, readErr)
	assert.NotNil(t, rootKf.Status)
	assert.Len(t, rootKf.Status.Conditions, 1)
	cond := rootKf.Status.Conditions[0]
	assert.Equal(t, kptfilev1.ConditionTrue, cond.Status)
	assert.Equal(t, kptfilev1.ReasonRenderSuccess, cond.Reason)
	assert.Contains(t, cond.Message, "notes.yaml")
	assert.Contains(t, cond.Message, AutoIgnoreNonKRMAnnotation)
}

// Low-level KRM-validity/split/decide tests live in pkg/lib/krmvalidate.
