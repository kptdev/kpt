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

// The non-KRM pre-scan: walk a package tree read-only, decide whether
// any non-KRM file found is fatal or excused by annotation, and report
// everything found in one pass, before any KRM-assuming command starts
// doing real work.
package krmvalidate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// AutoIgnoreNonKRMAnnotation is the per-package Kptfile annotation
// that excuses non-KRM files instead of failing on them.
const AutoIgnoreNonKRMAnnotation = "kpt.dev/auto-ignore-non-krm"

// NonKRMFile records one non-KRM file found during a scan.
type NonKRMFile struct {
	PkgPath string // absolute path of the owning package
	RelPath string // path relative to PkgPath
	Reason  string // the IsKRM() validation error
}

// Split partitions files into valid KRM resources and non-KRM files.
// A file containing any non-KRM document is excluded in its entirety.
func Split(pkgDir string, nodes []*yaml.RNode) (valid []*yaml.RNode, nonKRM []NonKRMFile, err error) {
	nonKRMPaths := make(map[string]bool)
	for _, n := range nodes {
		if krmErr := IsKRM(n); krmErr != nil {
			relPath, _, pathErr := kioutil.GetFileAnnotations(n)
			if pathErr != nil {
				return nil, nil, fmt.Errorf("non-krm-detection: resource in %s missing path annotation: %w", pkgDir, pathErr)
			}
			if !nonKRMPaths[relPath] {
				nonKRM = append(nonKRM, NonKRMFile{PkgPath: pkgDir, RelPath: relPath, Reason: krmErr.Error()})
				nonKRMPaths[relPath] = true
			}
		}
	}
	for _, n := range nodes {
		relPath, _, _ := kioutil.GetFileAnnotations(n)
		if !nonKRMPaths[relPath] {
			valid = append(valid, n)
		}
	}
	return valid, nonKRM, nil
}

// PackageOptedIn reports whether kf's package carries
// AutoIgnoreNonKRMAnnotation="true". False if kf is nil.
func PackageOptedIn(kf *kptfilev1.KptFile) bool {
	if kf == nil {
		return false
	}
	return kptfilev1.ToCondition(kf.Annotations[AutoIgnoreNonKRMAnnotation]) == kptfilev1.ConditionTrue
}

// ScanResult is the outcome of a successful Scan: the caller is clear
// to proceed.
type ScanResult struct {
	// AllResources is every (sub)package's resource list, keyed by
	// absolute package path, reusable via CachedLocalResources to
	// avoid a second disk read.
	AllResources map[string][]*yaml.RNode

	skipped []NonKRMFile
}

// Scan runs ScanTree, DecideOutcome, and PrintSkippedSummary for
// pkgPath: the single entry point most callers should use before
// starting any KRM-assuming work.
func Scan(fsys filesys.FileSystem, pkgPath string, printf func(format string, args ...any)) (*ScanResult, error) {
	found, optedIn, allResources, err := ScanTree(fsys, pkgPath, pkg.MatchAllKRM)
	if err != nil {
		return nil, fmt.Errorf("non-krm pre-scan: %w", err)
	}
	if scanErr := DecideOutcome(found, optedIn); scanErr != nil {
		return nil, scanErr
	}
	PrintSkippedSummary(printf, found, optedIn)

	var skipped []NonKRMFile
	for _, f := range found {
		if optedIn[f.PkgPath] {
			skipped = append(skipped, f)
		}
	}
	return &ScanResult{AllResources: allResources, skipped: skipped}, nil
}

// SkippedFiles returns every non-KRM file Scan excused via the
// annotation. Safe to call on a nil receiver.
func (res *ScanResult) SkippedFiles() []NonKRMFile {
	if res == nil {
		return nil
	}
	return res.skipped
}

// SkipFunc returns a FileSkipFunc for the files Scan excused, for
// callers wiring a reader's own skip hook (so those files are never
// read at all, rather than filtered out after the fact).
func (res *ScanResult) SkipFunc(pkgRootPath string) func(relPath string) bool {
	if res == nil {
		return nil
	}
	return skipFuncFor(res.skipped, pkgRootPath)
}

// ScanTree walks pkgPath and its subpackages read-only, returning
// every non-KRM file found, which packages opted in via
// AutoIgnoreNonKRMAnnotation, and every package's resource list for
// reuse. A missing Kptfile at the root is tolerated (bare-directory
// targets like `fn eval`'s); any other read error propagates.
//
// matchGlob scopes which files are considered; it should match the
// caller's real reader (pkg.MatchAllKRM for fn render/eval,
// kio.DefaultMatch for the live family), or a file the real operation
// never touches can still block it.
func ScanTree(fsys filesys.FileSystem, pkgPath string, matchGlob []string) (
	found []NonKRMFile, optedIn map[string]bool, allResources map[string][]*yaml.RNode, err error,
) {
	optedIn = map[string]bool{}
	allResources = map[string][]*yaml.RNode{}

	root, err := pkg.New(fsys, pkgPath)
	if err != nil {
		return nil, nil, nil, err
	}

	var walk func(p *pkg.Pkg, isRoot bool) error
	walk = func(p *pkg.Pkg, isRoot bool) error {
		dir := p.UniquePath.String()

		kf, kfErr := p.Kptfile()
		if kfErr != nil && !errors.Is(kfErr, os.ErrNotExist) {
			return kfErr
		}
		// kf is nil here either because Kptfile() returned
		// os.ErrNotExist (tolerated, see doc comment) or because kf
		// itself is a genuinely nil *KptFile; PackageOptedIn handles
		// nil already.
		optedIn[dir] = PackageOptedIn(kf)

		var resources []*yaml.RNode
		var resErr error
		if isRoot && kf == nil {
			// LocalResources() returns empty for a Kptfile-less dir,
			// but a bare-directory target (e.g. fn eval) still has
			// its real reader read every file regardless. Read the
			// same way here so nothing is silently skipped.
			resources, resErr = (&kio.LocalPackageReader{
				PackagePath:        dir,
				PackageFileName:    kptfilev1.KptFileName,
				IncludeSubpackages: false,
				MatchFilesGlob:     matchGlob,
				WrapBareSeqNode:    true,
				FileSystem:         filesys.FileSystemOrOnDisk{FileSystem: fsys},
			}).Read()
		} else {
			// LocalResources() always reads with pkg.MatchAllKRM;
			// narrow to matchGlob here for callers whose real
			// reader is stricter (e.g. live's yaml/yml-only).
			resources, resErr = p.LocalResources()
			resources = filterByMatchGlob(resources, matchGlob)
		}
		if resErr != nil {
			return resErr
		}
		allResources[dir] = resources

		_, nonKRM, splitErr := Split(dir, resources)
		if splitErr != nil {
			return splitErr
		}
		found = append(found, nonKRM...)

		subpkgs, subErr := p.DirectSubpackages()
		if subErr != nil {
			return subErr
		}
		for _, sub := range subpkgs {
			if err := walk(sub, false); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(root, true); err != nil {
		return nil, nil, nil, err
	}
	return found, optedIn, allResources, nil
}

// liveReaderMatchGlob mirrors pkg/live's real reader (no MatchFilesGlob
// override, so it defaults to kio.DefaultMatch): *.yaml/*.yml only, no
// Kptfile, no .json. Used instead of pkg.MatchAllKRM so a file the live
// reader never touches (e.g. a stray package.json) is never flagged.
var liveReaderMatchGlob = kio.DefaultMatch

// filterByMatchGlob drops nodes whose source file doesn't match
// matchGlob, narrowing an already-MatchAllKRM-globbed read without a
// second disk read.
func filterByMatchGlob(nodes []*yaml.RNode, matchGlob []string) []*yaml.RNode {
	var out []*yaml.RNode
	for _, n := range nodes {
		relPath, _, err := kioutil.GetFileAnnotations(n)
		if err != nil {
			continue
		}
		base := filepath.Base(relPath)
		for _, g := range matchGlob {
			if match, _ := filepath.Match(g, base); match {
				out = append(out, n)
				break
			}
		}
	}
	return out
}

// ScanFlat wraps ScanTree for callers whose real reader has no
// subpackage concept (e.g. the `live` family's flat reader), flattening
// the per-package result while keeping per-file ownership for opt-in.
func ScanFlat(fsys filesys.FileSystem, pkgPath string) (found []NonKRMFile, optedIn bool, allResources []*yaml.RNode, err error) {
	treeFound, treeOptedIn, treeResources, scanErr := ScanTree(fsys, pkgPath, liveReaderMatchGlob)
	if scanErr != nil {
		return nil, false, nil, scanErr
	}

	// optedIn is pkgPath's own root only; migrate (the one caller
	// that still reads it) ignores it anyway.
	optedIn = treeOptedIn[pkgPath]

	for _, pkgResources := range treeResources {
		allResources = append(allResources, pkgResources...)
	}

	found = treeFound
	return found, optedIn, allResources, nil
}

// FlatScanResult mirrors ScanResult for ScanFlat's flat-reader model.
type FlatScanResult struct {
	AllResources []*yaml.RNode
	skipped      []NonKRMFile
}

// ScanFlatAndDecide is Scan for flat-reader callers: same per-package
// opt-in via ScanTree, flattened into FlatScanResult's shape.
func ScanFlatAndDecide(fsys filesys.FileSystem, pkgPath string, printf func(format string, args ...any)) (*FlatScanResult, error) {
	found, treeOptedIn, treeResources, err := ScanTree(fsys, pkgPath, liveReaderMatchGlob)
	if err != nil {
		return nil, fmt.Errorf("non-krm pre-scan: %w", err)
	}
	if scanErr := DecideOutcome(found, treeOptedIn); scanErr != nil {
		return nil, scanErr
	}
	PrintSkippedSummary(printf, found, treeOptedIn)

	var allResources []*yaml.RNode
	for _, pkgResources := range treeResources {
		allResources = append(allResources, pkgResources...)
	}

	var skipped []NonKRMFile
	for _, f := range found {
		if treeOptedIn[f.PkgPath] {
			skipped = append(skipped, f)
		}
	}
	return &FlatScanResult{AllResources: allResources, skipped: skipped}, nil
}

// SkippedFiles mirrors ScanResult.SkippedFiles for FlatScanResult.
func (res *FlatScanResult) SkippedFiles() []NonKRMFile {
	if res == nil {
		return nil
	}
	return res.skipped
}

// SkipFunc mirrors ScanResult.SkipFunc, re-rooting each file's path
// (relative to its own, possibly-sub, package) against pkgRootPath.
func (res *FlatScanResult) SkipFunc(pkgRootPath string) func(relPath string) bool {
	if res == nil {
		return nil
	}
	return skipFuncFor(res.skipped, pkgRootPath)
}

// skipFuncFor builds a FileSkipFunc from a list of excused files,
// re-rooting each one's path against pkgRootPath. Shared by
// ScanResult.SkipFunc and FlatScanResult.SkipFunc.
func skipFuncFor(skipped []NonKRMFile, pkgRootPath string) func(relPath string) bool {
	if len(skipped) == 0 {
		return nil
	}
	skip := make(map[string]bool, len(skipped))
	for _, f := range skipped {
		abs := filepath.Join(f.PkgPath, f.RelPath)
		rel, err := filepath.Rel(pkgRootPath, abs)
		if err != nil {
			continue
		}
		skip[filepath.ToSlash(rel)] = true
	}
	return func(relPath string) bool {
		return skip[filepath.ToSlash(relPath)]
	}
}

// CachedLocalResources returns p's resources from cache if present,
// falling back to a real LocalResources() read otherwise.
func CachedLocalResources(p *pkg.Pkg, cache map[string][]*yaml.RNode) ([]*yaml.RNode, error) {
	if cache != nil {
		if resources, ok := cache[p.UniquePath.String()]; ok {
			return resources, nil
		}
	}
	return p.LocalResources()
}

// DecideOutcome returns an error listing every file in found whose
// owning package didn't opt in via optedIn, or nil if all were excused.
func DecideOutcome(found []NonKRMFile, optedIn map[string]bool) error {
	var blocking []NonKRMFile
	for _, f := range found {
		if !optedIn[f.PkgPath] {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) == 0 {
		return nil
	}

	sort.Slice(blocking, func(i, j int) bool {
		if blocking[i].PkgPath != blocking[j].PkgPath {
			return blocking[i].PkgPath < blocking[j].PkgPath
		}
		return blocking[i].RelPath < blocking[j].RelPath
	})

	var b strings.Builder
	fmt.Fprintf(&b, "input resource list must contain only KRM resources (%d file(s)):", len(blocking))
	for _, f := range blocking {
		fmt.Fprintf(&b, "\n  %s: %s", f.RelPath, f.Reason)
	}
	fmt.Fprintf(&b, "\nTo resolve: either exclude these files via .krmignore (manually, or with a pattern rule), "+
		"or add `%s: \"true\"` to this package's Kptfile to skip any non-KRM files automatically from this package.", AutoIgnoreNonKRMAnnotation)
	return errors.New(b.String())
}

// PrintSkippedSummary prints which non-KRM files were excused per
// opted-in package. No-op if nothing was skipped.
func PrintSkippedSummary(printf func(format string, args ...any), found []NonKRMFile, optedIn map[string]bool) {
	byPkg := map[string][]NonKRMFile{}
	var pkgOrder []string
	for _, f := range found {
		if !optedIn[f.PkgPath] {
			continue
		}
		if _, seen := byPkg[f.PkgPath]; !seen {
			pkgOrder = append(pkgOrder, f.PkgPath)
		}
		byPkg[f.PkgPath] = append(byPkg[f.PkgPath], f)
	}
	sort.Strings(pkgOrder)

	for _, pkgPath := range pkgOrder {
		files := byPkg[pkgPath]
		names := make([]string, 0, len(files))
		for _, f := range files {
			names = append(names, f.RelPath)
		}
		printf("[info] ignoring %d non-KRM file(s) in %q due to %s annotation: %s\n",
			len(files), pkgPath, AutoIgnoreNonKRMAnnotation, strings.Join(names, ", "))
	}
}
