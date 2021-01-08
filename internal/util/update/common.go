package update

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
)

// findAllSubpackages traverses the provided package paths
// and finds all subpackages. A subpackage is a subdirectory underneath the
// root that has a Kptfile in it.
// The list is sorted in increasing order based on the depth of the subpackage
// relative to the root package.
func findAllSubpackages(pkgPaths ...string) ([]string, error) {
	uniquePaths := make(map[string]bool)
	for _, path := range pkgPaths {
		paths, err := pathutil.DirsWithFile(path, kptfile.KptFileName, true)
		if err != nil {
			return []string{}, err
		}
		for _, p := range paths {
			relPath, err := filepath.Rel(path, p)
			if err != nil {
				return []string{}, err
			}
			uniquePaths[relPath] = true
		}
	}
	var paths []string
	for p := range uniquePaths {
		paths = append(paths, p)
	}
	sort.Slice(paths, rootPkgFirstSorter(paths))
	return paths, nil
}

// rootPkgFirstSorter returns a "less" function that can be used with the
// sort.Slice function to correctly sort package paths so parent packages
// are always before subpackages.
func rootPkgFirstSorter(paths []string) func(i, j int) bool {
	return func(i, j int) bool {
		iPath := paths[i]
		jPath := paths[j]
		if iPath == "." {
			return true
		}
		if jPath == "." {
			return false
		}
		iSegmentCount := len(strings.Split(iPath, "/"))
		jSegmentCount := len(strings.Split(jPath, "/"))
		return iSegmentCount < jSegmentCount
	}
}

// subPkgFirstSorter returns a "less" function that can be used with the
// sort.Slice function to correctly sort package paths so subpackages are
// always before parent packages.
func subPkgFirstSorter(paths []string) func(i, j int) bool {
	sorter := rootPkgFirstSorter(paths)
	return func(i, j int) bool {
		return !sorter(i, j)
	}
}

// exists returns true if a file or directory exists on the provided path,
// and false otherwise.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return !os.IsNotExist(err), nil
}
