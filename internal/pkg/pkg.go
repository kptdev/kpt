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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const CurDir = "."
const ParentDir = ".."

const (
	pkgPathAnnotation = "internal.config.kubernetes.io/package-path"
)

var DeprecatedKptfileVersions = []string{
	"v1alpha1",
	"v1alpha2",
}

var SupportedKptfileVersions = []string{
	kptfilev1.KptFileVersion,
}

// KptfileError records errors regarding reading or parsing of a Kptfile.
type KptfileError struct {
	Path types.UniquePath
	Err  error
}

func (k *KptfileError) Error() string {
	return fmt.Sprintf("error reading Kptfile at %q: %s", k.Path.String(), k.Err.Error())
}

func (k *KptfileError) Unwrap() error {
	return k.Err
}

// RemoteKptfileError records errors regarding reading or parsing of a Kptfile
// in a remote repo.
type RemoteKptfileError struct {
	RepoSpec *git.RepoSpec
	Err      error
}

func (e *RemoteKptfileError) Error() string {
	return fmt.Sprintf("error reading Kptfile from %q: %v", e.RepoSpec.RepoRef(), e.Err)
}

func (e *RemoteKptfileError) Unwrap() error {
	return e.Err
}

// DeprecatedKptfileError is an implementation of the error interface that is
// returned whenever kpt encounters a Kptfile using the legacy format.
type DeprecatedKptfileError struct {
	Version string
}

func (e *DeprecatedKptfileError) Error() string {
	return fmt.Sprintf("old resource version %q found in Kptfile", e.Version)
}

type UnknownKptfileResourceError struct {
	GVK schema.GroupVersionKind
}

func (e *UnknownKptfileResourceError) Error() string {
	return fmt.Sprintf("unknown resource type %q found in Kptfile", e.GVK.String())
}

// RGError is an implementation of the error interface that is returned whenever
// kpt encounters errors reading a resourcegroup object file.
type RGError struct {
	Path types.UniquePath
	Err  error
}

func (rg *RGError) Error() string {
	return fmt.Sprintf("error reading ResourceGroup file at %q: %s", rg.Path.String(), rg.Err.Error())
}

func (rg *RGError) Unwrap() error {
	return rg.Err
}

// Pkg represents a kpt package with a one-to-one mapping to a directory on the local filesystem.
type Pkg struct {
	// UniquePath represents absolute unique OS-defined path to the package directory on the filesystem.
	UniquePath types.UniquePath

	// DisplayPath represents Slash-separated path to the package directory on the filesystem relative
	// to parent directory of root package on which the command is invoked.
	// root package is defined as the package on which the command is invoked by user
	// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
	// be used for display purposes and is subject to change.
	DisplayPath types.DisplayPath

	// rootPkgParentDirPath is the absolute path to the parent directory of root package
	// root package is defined as the package on which the command is invoked by user
	// this must be same for all the nested subpackages in root package
	rootPkgParentDirPath string

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	kptfile *kptfilev1.KptFile

	// A package can contain zero or one ResourceGroup object.
	rgFile *rgfilev1alpha1.ResourceGroup
}

// New returns a pkg given an absolute or relative OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(path string) (*Pkg, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		// If the provided path is relative, we find the absolute path by
		// combining the current working directory with the path.
		absPath = filepath.Join(cwd, path)
	}
	pkg := &Pkg{
		UniquePath: types.UniquePath(absPath),
		// by default, rootPkgParentDirPath should be the absolute path to the parent directory of package being instantiated
		rootPkgParentDirPath: filepath.Dir(absPath),
		// by default, DisplayPath should be the package name which is same as directory name
		DisplayPath: types.DisplayPath(filepath.Base(absPath)),
	}
	return pkg, nil
}

// Kptfile returns the Kptfile meta resource by lazy loading it from the filesytem.
// A nil value represents an implicit package.
func (p *Pkg) Kptfile() (*kptfilev1.KptFile, error) {
	if p.kptfile == nil {
		kf, err := ReadKptfile(p.UniquePath.String())
		if err != nil {
			return nil, err
		}
		p.kptfile = kf
	}
	return p.kptfile, nil
}

// ReadKptfile reads the KptFile in the given pkg.
// TODO(droot): This method exists for current version of Kptfile.
// Need to reconcile with the team how we want to handle multiple versions
// of Kptfile in code. One option is to follow Kubernetes approach to
// have an internal version of Kptfile that all the code uses. In that case,
// we will have to implement pieces for IO/Conversion with right interfaces.
func ReadKptfile(p string) (*kptfilev1.KptFile, error) {
	f, err := os.Open(filepath.Join(p, kptfilev1.KptFileName))
	if err != nil {
		return nil, &KptfileError{
			Path: types.UniquePath(p),
			Err:  err,
		}
	}
	defer f.Close()

	kf, err := DecodeKptfile(f)
	if err != nil {
		return nil, &KptfileError{
			Path: types.UniquePath(p),
			Err:  err,
		}
	}
	return kf, nil
}

func DecodeKptfile(in io.Reader) (*kptfilev1.KptFile, error) {
	kf := &kptfilev1.KptFile{}
	c, err := io.ReadAll(in)
	if err != nil {
		return kf, err
	}

	if err := CheckKptfileVersion(c); err != nil {
		return kf, err
	}

	d := yaml.NewDecoder(bytes.NewBuffer(c))
	d.KnownFields(true)
	if err := d.Decode(kf); err != nil {
		return kf, err
	}
	return kf, nil
}

// CheckKptfileVersion verifies the apiVersion and kind of the resource
// within the Kptfile. If the legacy version is found, the DeprecatedKptfileError
// is returned. If the currently supported apiVersion and kind is found, no
// error is returned.
func CheckKptfileVersion(content []byte) error {
	r, err := yaml.Parse(string(content))
	if err != nil {
		return err
	}

	m, err := r.GetMeta()
	if err != nil {
		return err
	}

	kind := m.Kind
	gv, err := schema.ParseGroupVersion(m.APIVersion)
	if err != nil {
		return err
	}

	switch {
	// If the resource type matches what we are looking for, just return nil.
	case gv.Group == kptfilev1.KptFileGroup &&
		kind == kptfilev1.KptFileKind &&
		isSupportedKptfileVersion(gv.Version):
		return nil
	// If the kind and group is correct and the version is a known deprecated
	// schema for the Kptfile, return DeprecatedKptfileError.
	case gv.Group == kptfilev1.KptFileGroup &&
		kind == kptfilev1.KptFileKind &&
		isDeprecatedKptfileVersion(gv.Version):
		return &DeprecatedKptfileError{
			Version: gv.Version,
		}
	// If the combination of group, version and kind are unknown to us, return
	// UnknownKptfileResourceError.
	default:
		return &UnknownKptfileResourceError{
			GVK: gv.WithKind(kind),
		}
	}
}

func isDeprecatedKptfileVersion(version string) bool {
	return slice.ContainsString(DeprecatedKptfileVersions, version, nil)
}

func isSupportedKptfileVersion(version string) bool {
	return slice.ContainsString(SupportedKptfileVersions, version, nil)
}

// Pipeline returns the Pipeline section of the pkg's Kptfile.
// if pipeline is not specified in a Kptfile, it returns Zero value of the pipeline.
func (p *Pkg) Pipeline() (*kptfilev1.Pipeline, error) {
	kf, err := p.Kptfile()
	if err != nil {
		return nil, err
	}
	pl := kf.Pipeline
	if pl == nil {
		return &kptfilev1.Pipeline{}, nil
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
			return subPkgs, fmt.Errorf("failed to read package at path %q: %w", subPkgPath, err)
		}
		if err := p.adjustDisplayPathForSubpkg(subPkg); err != nil {
			return subPkgs, fmt.Errorf("failed to resolve display path for %q: %w", subPkgPath, err)
		}
		subPkgs = append(subPkgs, subPkg)
	}

	sort.Slice(subPkgs, func(i, j int) bool {
		return subPkgs[i].DisplayPath < subPkgs[j].DisplayPath
	})
	return subPkgs, nil
}

// adjustDisplayPathForSubpkg adjusts the display path of subPkg relative to the RootPkgUniquePath
// subPkg also inherits the RootPkgUniquePath value from parent package p
func (p *Pkg) adjustDisplayPathForSubpkg(subPkg *Pkg) error {
	// inherit the rootPkgParentDirPath from the parent package
	subPkg.rootPkgParentDirPath = p.rootPkgParentDirPath
	// display path of subPkg should be relative to parent dir of rootPkg
	// e.g. if mysql(subPkg) is direct subpackage of wordpress(p), DisplayPath of "mysql" should be "wordpress/mysql"
	dp, err := filepath.Rel(subPkg.rootPkgParentDirPath, string(subPkg.UniquePath))
	if err != nil {
		return err
	}
	// make sure that the DisplayPath is always Slash-separated os-agnostic
	subPkg.DisplayPath = types.DisplayPath(filepath.ToSlash(dp))
	return nil
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
	// None means that no subpackages will be returned.
	None SubpackageMatcher = "NONE"
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
	const op errors.Op = "pkg.Subpackages"

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
				kf, err := ReadKptfile(path)
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
	_, err := os.Stat(filepath.Join(path, kptfilev1.KptFileName))

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
	kf, err := ReadKptfile(path)
	if err != nil {
		return false, err
	}
	return kf.Upstream != nil && kf.UpstreamLock == nil, nil
}

// LocalResources returns resources that belong to this package excluding the subpackage resources.
func (p *Pkg) LocalResources(includeMetaResources bool) (resources []*yaml.RNode, err error) {
	const op errors.Op = "pkg.readResources"

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
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: false,
		MatchFilesGlob:     kio.MatchAll,
		PreserveSeqIndent:  true,
		SetAnnotations: map[string]string{
			pkgPathAnnotation: string(p.UniquePath),
		},
		WrapBareSeqNode: true,
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

// Validates the package pipeline.
func (p *Pkg) ValidatePipeline() error {
	pl, err := p.Pipeline()
	if err != nil {
		return err
	}

	if pl.IsEmpty() {
		return nil
	}

	// read all resources including function pipeline.
	resources, err := p.LocalResources(true)
	if err != nil {
		return err
	}

	resourcesByPath := sets.String{}

	for _, r := range resources {
		rPath, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return fmt.Errorf("resource missing path annotation err: %w", err)
		}
		resourcesByPath.Insert(filepath.Clean(rPath))
	}

	for i, fn := range pl.Mutators {
		if fn.ConfigPath != "" && !resourcesByPath.Has(filepath.Clean(fn.ConfigPath)) {
			return &kptfilev1.ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].configPath", "mutators", i),
				Value:  fn.ConfigPath,
				Reason: "functionConfig must exist in the current package",
			}
		}
	}
	for i, fn := range pl.Validators {
		if fn.ConfigPath != "" && !resourcesByPath.Has(filepath.Clean(fn.ConfigPath)) {
			return &kptfilev1.ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].configPath", "validators", i),
				Value:  fn.ConfigPath,
				Reason: "functionConfig must exist in the current package",
			}
		}
	}
	return nil
}

// filterMetaResources filters kpt metadata files such as Kptfile, function configs.
func filterMetaResources(input []*yaml.RNode, pl *kptfilev1.Pipeline) (output []*yaml.RNode, err error) {
	pathsToExclude := fnConfigFilePaths(pl)
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

// fnConfigFilePaths returns paths to function config files referred in the
// given pipeline.
func fnConfigFilePaths(pl *kptfilev1.Pipeline) (fnConfigPaths sets.String) {
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

// FunctionConfigFilePaths returns a set of config file paths that used by
// package pipeline. rootPath is the path to the package. recursive decides
// will config file paths in subpackages will be returned. Returned paths
// are all relative to rootPath.
func FunctionConfigFilePaths(rootPath types.UniquePath, recursive bool) (sets.String, error) {
	const op errors.Op = "pkg.FunctionConfigFilePaths"
	ok, err := IsPackageDir(string(rootPath))
	if err != nil {
		return nil, errors.E(op, rootPath, err)
	}
	var pkgPaths []types.UniquePath
	if ok {
		pkgPaths = []types.UniquePath{rootPath}
	}
	if recursive {
		subPkgPaths, err := Subpackages(string(rootPath), All, true)
		if err != nil {
			return nil, errors.E(op, rootPath, fmt.Errorf("failed to get subpackage paths: %w", err))
		}
		for _, spp := range subPkgPaths {
			// sub package paths are all relative to rootPath
			pkgPaths = append(pkgPaths, types.UniquePath(filepath.Join(string(rootPath), spp)))
		}
	}
	fnConfigPaths := sets.String{}
	for _, uniquePath := range pkgPaths {
		path := string(uniquePath)
		p, err := New(path)
		if err != nil {
			return nil, errors.E(op, rootPath, err)
		}
		pl, err := p.Pipeline()
		if err != nil {
			return nil, errors.E(op, rootPath, fmt.Errorf("failed to get pipeline in package %s: %w", path, err))
		}
		// function file path are relative to the package which it's in
		for _, ffp := range fnConfigFilePaths(pl).List() {
			fnRelPath, err := filepath.Rel(string(rootPath), filepath.Join(path, ffp))
			if err != nil {
				return nil, errors.E(op, rootPath, fmt.Errorf("failed to get path relative to %s from %s: %w",
					rootPath, filepath.Join(path, ffp), err))
			}
			fnConfigPaths.Insert(fnRelPath)
		}
	}
	return fnConfigPaths, nil
}

// FunctionConfigFilterFunc returns a kio.LocalPackageSkipFileFunc filter which will be
// invoked by kio.LocalPackageReader when it reads the package. The filter will return
// true if the file should be skipped during reading. Skipped files will not be included
// in all steps following.
func FunctionConfigFilterFunc(pkgPath types.UniquePath, includeMetaResources bool) (kio.LocalPackageSkipFileFunc, error) {
	if includeMetaResources {
		return func(relPath string) bool {
			return false
		}, nil
	}

	fnConfigPaths, err := FunctionConfigFilePaths(pkgPath, true)
	if err != nil {
		return nil, err
	}

	return func(relPath string) bool {
		if len(fnConfigPaths) == 0 {
			return false
		}
		// relPath is cleaned so we can directly use it here
		return fnConfigPaths.Has(relPath)
	}, nil
}

// GetPkgPathAnnotation returns the package path annotation on
// a given resource.
func GetPkgPathAnnotation(rn *yaml.RNode) (string, error) {
	meta, err := rn.GetMeta()
	if err != nil {
		return "", err
	}
	pkgPath := meta.Annotations[pkgPathAnnotation]
	return pkgPath, nil
}

// SetPkgPathAnnotation sets package path on a given resource.
func SetPkgPathAnnotation(rn *yaml.RNode, pkgPath types.UniquePath) error {
	return rn.PipeE(yaml.SetAnnotation(pkgPathAnnotation, string(pkgPath)))
}

// RemovePkgPathAnnotation removes the package path on a given resource.
func RemovePkgPathAnnotation(rn *yaml.RNode) error {
	return rn.PipeE(yaml.ClearAnnotation(pkgPathAnnotation))
}

// ReadRGFile returns the resourcegroup object by lazy loading it from the filesytem.
func (p *Pkg) ReadRGFile(filename string) (*rgfilev1alpha1.ResourceGroup, error) {
	if p.rgFile == nil {
		rg, err := ReadRGFile(p.UniquePath.String(), filename)
		if err != nil {
			return nil, err
		}
		p.rgFile = rg
	}
	return p.rgFile, nil
}

// TODO(rquitales): Consolidate both Kptfile and ResourceGroup file reading functions to use
// shared logic/function.

// ReadRGFile reads the KptFile in the given pkg.
func ReadRGFile(path, filename string) (*rgfilev1alpha1.ResourceGroup, error) {
	f, err := os.Open(filepath.Join(path, filename))
	if err != nil {
		return nil, &RGError{
			Path: types.UniquePath(path),
			Err:  err,
		}
	}
	defer f.Close()

	rg, err := DecodeRGFile(f)
	if err != nil {
		return nil, &RGError{
			Path: types.UniquePath(path),
			Err:  err,
		}
	}
	return rg, nil
}

// DecodeRGFile converts a string reader into structured a ResourceGroup object.
func DecodeRGFile(in io.Reader) (*rgfilev1alpha1.ResourceGroup, error) {
	rg := &rgfilev1alpha1.ResourceGroup{}
	c, err := io.ReadAll(in)
	if err != nil {
		return rg, err
	}

	d := yaml.NewDecoder(bytes.NewBuffer(c))
	d.KnownFields(true)
	if err := d.Decode(rg); err != nil {
		return rg, err
	}
	return rg, nil
}
