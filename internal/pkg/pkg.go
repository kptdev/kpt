// Copyright 2020 Google LLC
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

// Package pkg defines the concept of a kpt package.
package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/types"
)

const CurDir = "."
const ParentDir = ".."

// Pkg represents a kpt package with a one-to-one mapping to a directory on the local filesystem.
type Pkg struct {
	UniquePath  types.UniquePath
	DisplayPath types.DisplayPath

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	kptfile *kptfilev1alpha2.KptFile
}

// New returns a pkg given an absolute or relative OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(path string) (*Pkg, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var relPath string
	var absPath string
	if filepath.IsAbs(path) {
		// If the provided path is absolute, we find the relative path by
		// comparing it to the current working directory.
		relPath, err = filepath.Rel(cwd, path)
		if err != nil {
			return nil, err
		}
		absPath = filepath.Clean(path)
	} else {
		// If the provided path is relative, we find the absolute path by
		// combining the current working directory with the relative path.
		relPath = filepath.Clean(path)
		absPath = filepath.Join(cwd, path)
	}
	return &Pkg{
		UniquePath:  types.UniquePath(absPath),
		DisplayPath: types.DisplayPath(relPath),
	}, nil
}

// Kptfile returns the Kptfile meta resource by lazy loading it from the filesytem.
// A nil value represents an implicit package.
func (p *Pkg) Kptfile() (*kptfilev1alpha2.KptFile, error) {
	if p.kptfile == nil {
		kf, err := readKptfile(p.UniquePath.String())
		if err != nil {
			return nil, err
		}
		p.kptfile = kf
	}
	return p.kptfile, nil
}

// readKptfile reads the KptFile in the given pkg.
// TODO(droot): This method exists for current version of Kptfile.
// Need to reconcile with the team how we want to handle multiple versions
// of Kptfile in code. One option is to follow Kubernetes approach to
// have an internal version of Kptfile that all the code uses. In that case,
// we will have to implement pieces for IO/Conversion with right interfaces.
func readKptfile(p string) (*kptfilev1alpha2.KptFile, error) {
	op := errors.Op("pkg.readKptfile")
	kf := &kptfilev1alpha2.KptFile{}

	f, err := os.Open(filepath.Join(p, kptfilev1alpha2.KptFileName))
	if err != nil {
		return kf, errors.E(op, fmt.Errorf("package must have a %q: %w", kptfilev1alpha2.KptFileName, err))
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(kf); err != nil {
		return kf, errors.E(op, fmt.Errorf("unable to parse %q: %w", kptfilev1alpha2.KptFileName, err))
	}
	return kf, nil
}

// Pipeline returns the Pipeline section of the pkg's Kptfile.
// if pipeline is not specified in a Kptfile, it returns Zero value of the pipeline.
func (p *Pkg) Pipeline() (*kptfilev1alpha2.Pipeline, error) {
	kf, err := p.Kptfile()
	if err != nil {
		return nil, err
	}
	pl := kf.Pipeline
	if pl == nil {
		return &kptfilev1alpha2.Pipeline{}, nil
	}
	return pl, nil
}

// String returns the slash-separated relative path to the package.
func (p *Pkg) String() string {
	return string(p.DisplayPath)
}

// RelativePathTo returns current package's path relative to a given package.
// It returns an error if relative path doesn't exist.
// In a nested package chain, one can use this method to get the relative
// path of a subpackage relative to an ancestor package up the chain.
// Example: rel, _ := subpkg.RelativePathTo(rootPkg)
// The returned relative path is compatible with the target operating
// system-defined file paths.
func (p *Pkg) RelativePathTo(ancestorPkg *Pkg) (string, error) {
	return filepath.Rel(string(ancestorPkg.UniquePath), string(p.UniquePath))
}

// DirectSubpackages returns subpackages of a pkg. It will return all direct
// subpackages, i.e. subpackages that aren't nested inside other subpackages
// under the current package. It will return packages that are nested inside
// directories of the current package.
// TODO: This does not support symlinks, so we need to figure out how
// we should support that with kpt.
func (p *Pkg) DirectSubpackages() ([]*Pkg, error) {
	var subPkgs []*Pkg

	packagePaths, err := Subpackages(p.UniquePath.String(), All, false)
	if err != nil {
		return subPkgs, err
	}

	for _, subPkgPath := range packagePaths {
		subPkg, err := New(filepath.Join(p.UniquePath.String(), subPkgPath))
		if err != nil {
			return subPkgs, fmt.Errorf("failed to read subpkg at path %s %w", subPkgPath, err)
		}
		subPkgs = append(subPkgs, subPkg)
	}

	sort.Slice(subPkgs, func(i, j int) bool {
		return subPkgs[i].DisplayPath < subPkgs[j].DisplayPath
	})
	return subPkgs, nil
}

// SubpackageMatcher is type for specifying the types of subpackages which
// should be included when listing them.
type SubpackageMatcher string

const (
	// All means all types of subpackages will be returned.
	All SubpackageMatcher = "ALL"
	// Local means only local subpackages will be returned.
	Local SubpackageMatcher = "LOCAL"
	// remote means only remote subpackages will be returned.
	Remote SubpackageMatcher = "REMOTE"
)

// Subpackages returns a slice of paths to any subpackages of the provided path.
// The matcher parameter decides if all types of subpackages should be considered,
// and the recursive parameter determines if only direct subpackages are
// considered. All returned paths will be relative to the provided rootPath.
// The top level package is not considered a subpackage. If the provided path
// doesn't exist, an empty slice will be returned.
// Symlinks are ignored.
// TODO: For now this accepts the path as a string type. See if we can leverage
// the package type here.
func Subpackages(rootPath string, matcher SubpackageMatcher, recursive bool) ([]string, error) {
	op := errors.Op("pkg.Subpackages")

	_, err := os.Stat(rootPath)
	if err != nil && !os.IsNotExist(err) {
		return []string{}, err
	}
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	packagePaths := make(map[string]bool)
	if err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to read package %s: %w", rootPath, err)
		}

		// Ignore the root folder
		if path == rootPath {
			return nil
		}

		// For every folder, we check if it is a kpt package
		if info.IsDir() {
			// Ignore anything inside the .git folder
			// TODO: We eventually want to support user-defined ignore lists.
			if info.Name() == ".git" {
				return filepath.SkipDir
			}

			// Check if the directory is the root of a kpt package
			isPkg, err := IsPackageDir(path)
			if err != nil {
				return err
			}

			// If the path is the root of a subpackage, add the
			// path to the slice and return SkipDir since we don't need to
			// walk any deeper into the directory.
			if isPkg {
				kf, err := readKptfile(path)
				if err != nil {
					return errors.E(op, types.UniquePath(path), err)
				}
				switch matcher {
				case Local:
					if kf.Upstream == nil {
						packagePaths[path] = true
					}
				case Remote:
					if kf.Upstream != nil {
						packagePaths[path] = true
					}
				case All:
					packagePaths[path] = true
				default:

				}
				if !recursive {
					return filepath.SkipDir
				}
				return nil
			}
		}
		return nil
	}); err != nil {
		return []string{}, fmt.Errorf("failed to read package at %s: %w", rootPath, err)
	}

	paths := []string{}
	for subPkgPath := range packagePaths {
		relPath, err := filepath.Rel(rootPath, subPkgPath)
		if err != nil {
			return paths, fmt.Errorf("failed to find relative path for %s: %w", subPkgPath, err)
		}
		paths = append(paths, relPath)
	}
	return paths, nil
}

// IsPackageDir checks if there exists a Kptfile on the provided path, i.e.
// whether the provided path is the root of a package.
func IsPackageDir(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, kptfilev1alpha2.KptFileName))

	// If we got an error that wasn't IsNotExist, something went wrong and
	// we don't really know if the file exists or not.
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	// If the error is IsNotExist, we know the file doesn't exist.
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

// IsPackageUnfetched returns true if a package has Upstream information,
// but no UpstreamLock. For local packages that doesn't have Upstream
// information, it will always return false.
// If a Kptfile is not found on the provided path, an error will be returned.
func IsPackageUnfetched(path string) (bool, error) {
	kf, err := readKptfile(path)
	if err != nil {
		return false, err
	}
	return kf.Upstream != nil && kf.UpstreamLock == nil, nil
}

// LocalResources returns resources that belong to this package excluding the subpackage resources.
func (p *Pkg) LocalResources(includeMetaResources bool) (resources []*yaml.RNode, err error) {
	op := errors.Op("pkg.readResources")

	hasKptfile, err := IsPackageDir(p.UniquePath.String())
	if err != nil {
		return nil, errors.E(op, p.UniquePath, err)
	}
	if !hasKptfile {
		return nil, nil
	}
	pl, err := p.Pipeline()
	if err != nil {
		return nil, errors.E(op, p.UniquePath, err)
	}

	pkgReader := &kio.LocalPackageReader{
		PackagePath:        string(p.UniquePath),
		PackageFileName:    kptfilev1alpha2.KptFileName,
		IncludeSubpackages: false,
		MatchFilesGlob:     kio.MatchAll,
	}
	resources, err = pkgReader.Read()
	if err != nil {
		return resources, errors.E(op, p.UniquePath, err)
	}
	if !includeMetaResources {
		resources, err = filterMetaResources(resources, pl)
		if err != nil {
			return resources, errors.E(op, p.UniquePath, err)
		}
	}
	return resources, err
}

// filterMetaResources filters kpt metadata files such as Kptfile, function configs.
func filterMetaResources(input []*yaml.RNode, pl *kptfilev1alpha2.Pipeline) (output []*yaml.RNode, err error) {
	pathsToExclude := functionConfigFilePaths(pl)
	for _, r := range input {
		meta, err := r.GetMeta()
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata for resource %w", err)
		}
		path, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read path while filtering meta resources %w", err)
		}
		// filter out pkg metadata such as Kptfile
		if strings.Contains(meta.APIVersion, "kpt.dev") {
			continue
		}
		// filter out function config files
		if pathsToExclude.Has(path) {
			continue
		}
		output = append(output, r)
	}
	return output, nil
}

// functionConfigFilePaths returns paths to function config files referred in the
// given pipeline.
func functionConfigFilePaths(pl *kptfilev1alpha2.Pipeline) (fnConfigPaths sets.String) {
	if pl == nil {
		return nil
	}
	fnConfigPaths = sets.String{}

	for _, fn := range pl.Mutators {
		if fn.ConfigPath != "" {
			// TODO(droot): check if cleaning this path has some unnecessary side effects
			fnConfigPaths.Insert(filepath.Clean(fn.ConfigPath))
		}
	}
	for _, fn := range pl.Validators {
		if fn.ConfigPath != "" {
			// TODO(droot): check if cleaning this path has some unnecessary side effects
			fnConfigPaths.Insert(filepath.Clean(fn.ConfigPath))
		}
	}
	return fnConfigPaths
}
