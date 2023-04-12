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

// Package pkg defines the concept of a kpt package.
package pkg

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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

var DeprecatedKptfileVersions = []schema.GroupVersionKind{
	kptfilev1.KptFileGVK().GroupKind().WithVersion("v1alpha1"),
	kptfilev1.KptFileGVK().GroupKind().WithVersion("v1alpha2"),
}

// MatchAllKRM represents set of glob pattern to match all KRM
// resources including Kptfile.
var MatchAllKRM = append([]string{kptfilev1.KptFileName}, kio.MatchAll...)

var SupportedKptfileVersions = []schema.GroupVersionKind{
	kptfilev1.KptFileGVK(),
}

// KptfileError records errors regarding reading or parsing of a Kptfile.
type KptfileError struct {
	Path types.UniquePath
	Err  error
}

func (k *KptfileError) Error() string {
	return fmt.Sprintf("error reading Kptfile at %q: %v", k.Path.String(), k.Err)
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

// MultipleResourceGroupsError is the error returned if there are multiple
// inventories provided in a stream or package as ResourceGroup objects.
type MultipleResourceGroupsError struct{}

func (e *MultipleResourceGroupsError) Error() string {
	return "multiple ResourceGroup objects found in package"
}

// MultipleKfInv is the error returned if there are multiple
// inventories provided in a stream or package as ResourceGroup objects.
type MultipleKfInv struct{}

func (e *MultipleKfInv) Error() string {
	return "multiple Kptfile inventories found in package"
}

// MultipleInventoryInfoError is the error returned if there are multiple
// inventories provided in a stream or package contained with both Kptfile and
// ResourceGroup objects.
type MultipleInventoryInfoError struct{}

func (e *MultipleInventoryInfoError) Error() string {
	return "inventory was found in both Kptfile and ResourceGroup object"
}

// NoInvInfoError is the error returned if there are no inventory information
// provided in either a stream or locally.
type NoInvInfoError struct{}

func (e *NoInvInfoError) Error() string {
	return "no ResourceGroup object was provided within the stream or package"
}

type InvInfoInvalid struct{}

func (e *InvInfoInvalid) Error() string {
	return "the provided ResourceGroup is not valid"
}

// warnInvInKptfile is the warning message when the inventory information is present within the Kptfile.
//
//nolint:lll
const warnInvInKptfile = "[WARN] The resourcegroup file was not found... Using Kptfile to gather inventory information. We recommend migrating to a resourcegroup file for inventories. Please migrate with `kpt live migrate`."

// Pkg represents a kpt package with a one-to-one mapping to a directory on the local filesystem.
type Pkg struct {
	// fsys represents the FileSystem of the package, it may or may not be FileSystem on disk
	fsys filesys.FileSystem

	// UniquePath represents absolute unique OS-defined path to the package directory on the filesystem.
	UniquePath types.UniquePath

	// DisplayPath represents Slash-separated path to the package directory on the filesystem relative
	// to parent directory of root package on which the command is invoked.
	// root package is defined as the package on which the command is invoked by user
	// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
	// be used for display purposes and is subject to change.
	DisplayPath types.DisplayPath

	// rootPkgParentDirPath is the absolute path to the parent directory of root package,
	// root package is defined as the package on which the command is invoked by user
	// this must be same for all the nested subpackages in root package
	rootPkgParentDirPath string

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	kptfile *kptfilev1.KptFile

	// A package can contain zero or one ResourceGroup object.
	rgFile *rgfilev1alpha1.ResourceGroup
}

// New returns a pkg given an absolute OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(fs filesys.FileSystem, path string) (*Pkg, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("provided path %s must be absolute", path)
	}
	absPath := filepath.Clean(path)
	pkg := &Pkg{
		fsys:       fs,
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
		kf, err := ReadKptfile(p.fsys, p.UniquePath.String())
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
func ReadKptfile(fs filesys.FileSystem, p string) (*kptfilev1.KptFile, error) {
	f, err := fs.Open(filepath.Join(p, kptfilev1.KptFileName))
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
	gvk := gv.WithKind(kind)

	switch {
	// If the resource type matches what we are looking for, just return nil.
	case isSupportedKptfileVersion(gvk):
		return nil
	// If the kind and group is correct and the version is a known deprecated
	// schema for the Kptfile, return DeprecatedKptfileError.
	case isDeprecatedKptfileVersion(gvk):
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

func isDeprecatedKptfileVersion(gvk schema.GroupVersionKind) bool {
	for _, v := range DeprecatedKptfileVersions {
		if v == gvk {
			return true
		}
	}
	return false
}

func isSupportedKptfileVersion(gvk schema.GroupVersionKind) bool {
	for _, v := range SupportedKptfileVersions {
		if v == gvk {
			return true
		}
	}
	return false
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

	packagePaths, err := Subpackages(p.fsys, p.UniquePath.String(), All, false)
	if err != nil {
		return subPkgs, err
	}

	for _, subPkgPath := range packagePaths {
		subPkg, err := New(p.fsys, filepath.Join(p.UniquePath.String(), subPkgPath))
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
// The matcher parameter decides the types of subpackages should be considered(ALL/LOCAL/REMOTE/NONE),
// and the recursive parameter determines if only direct subpackages are
// considered. All returned paths will be relative to the provided rootPath.
// The top level package is not considered a subpackage. If the provided path
// doesn't exist, an empty slice will be returned.
// Symlinks are ignored.
// TODO: For now this accepts the path as a string type. See if we can leverage
// the package type here.
func Subpackages(fsys filesys.FileSystem, rootPath string, matcher SubpackageMatcher, recursive bool) ([]string, error) {
	const op errors.Op = "pkg.Subpackages"

	if !fsys.Exists(rootPath) {
		return []string{}, nil
	}
	packagePaths := make(map[string]bool)
	if err := fsys.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
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
			isPkg, err := IsPackageDir(fsys, path)
			if err != nil {
				return err
			}

			// If the path is the root of a subpackage, add the
			// path to the slice and return SkipDir since we don't need to
			// walk any deeper into the directory.
			if isPkg {
				kf, err := ReadKptfile(fsys, path)
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
func IsPackageDir(fsys filesys.FileSystem, path string) (bool, error) {
	if !fsys.Exists(filepath.Join(path, kptfilev1.KptFileName)) {
		return false, nil
	}
	return true, nil
}

// IsPackageUnfetched returns true if a package has Upstream information,
// but no UpstreamLock. For local packages that doesn't have Upstream
// information, it will always return false.
// If a Kptfile is not found on the provided path, an error will be returned.
func IsPackageUnfetched(path string) (bool, error) {
	kf, err := ReadKptfile(filesys.FileSystemOrOnDisk{}, path)
	if err != nil {
		return false, err
	}
	return kf.Upstream != nil && kf.UpstreamLock == nil, nil
}

// LocalResources returns resources that belong to this package excluding the subpackage resources.
func (p *Pkg) LocalResources() (resources []*yaml.RNode, err error) {
	const op errors.Op = "pkg.readResources"

	var hasKptfile bool
	hasKptfile, err = IsPackageDir(p.fsys, p.UniquePath.String())
	if err != nil {
		return nil, errors.E(op, p.UniquePath, err)
	}
	if !hasKptfile {
		return nil, nil
	}

	pkgReader := &kio.LocalPackageReader{
		PackagePath:        string(p.UniquePath),
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: false,
		MatchFilesGlob:     MatchAllKRM,
		PreserveSeqIndent:  true,
		SetAnnotations: map[string]string{
			pkgPathAnnotation: string(p.UniquePath),
		},
		WrapBareSeqNode: true,
		FileSystem: filesys.FileSystemOrOnDisk{
			FileSystem: p.fsys,
		},
	}
	resources, err = pkgReader.Read()
	if err != nil {
		return resources, errors.E(op, p.UniquePath, err)
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
	resources, err := p.LocalResources()
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
func (p *Pkg) ReadRGFile(rgfile string) (*rgfilev1alpha1.ResourceGroup, error) {
	if p.rgFile == nil {
		rg, err := ReadRGFile(p.UniquePath.String(), rgfile)
		if err != nil {
			return nil, err
		}
		p.rgFile = rg
	}
	return p.rgFile, nil
}

// TODO(rquitales): Consolidate both Kptfile and ResourceGroup file reading functions to use
// shared logic/function.

// ReadRGFile reads the resourcegroup inventory in the given pkg.
func ReadRGFile(pkgPath, rgfile string) (*rgfilev1alpha1.ResourceGroup, error) {
	// Check to see if filename for ResourceGroup is a filepath, rather than being relative to the pkg path.
	// If only a filename is provided, we assume that the resourcegroup file is relative to the pkg path.
	var absPath string
	if filepath.Base(rgfile) == rgfile {
		absPath = filepath.Join(pkgPath, rgfile)
	} else {
		rgFilePath, _, err := pathutil.ResolveAbsAndRelPaths(rgfile)
		if err != nil {
			return nil, &RGError{
				Path: types.UniquePath(rgfile),
				Err:  err,
			}
		}

		absPath = rgFilePath
	}

	f, err := os.Open(absPath)
	if err != nil {
		return nil, &RGError{
			Path: types.UniquePath(absPath),
			Err:  err,
		}
	}
	defer f.Close()

	rg, err := DecodeRGFile(f)
	if err != nil {
		return nil, &RGError{
			Path: types.UniquePath(absPath),
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

// LocalInventory returns the package inventory stored within a package. If more than one, or no inventories are
// found, an error is returned instead.
func (p *Pkg) LocalInventory() (kptfilev1.Inventory, error) {
	const op errors.Op = "pkg.LocalInventory"

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
		FileSystem: filesys.FileSystemOrOnDisk{
			FileSystem: p.fsys,
		},
	}
	resources, err := pkgReader.Read()
	if err != nil {
		return kptfilev1.Inventory{}, errors.E(op, p.UniquePath, err)
	}

	resources, err = filterResourceGroups(resources)
	if err != nil {
		return kptfilev1.Inventory{}, errors.E(op, p.UniquePath, err)
	}

	// Multiple ResourceGroups found.
	if len(resources) > 1 {
		return kptfilev1.Inventory{}, &MultipleResourceGroupsError{}
	}

	// Load Kptfile and check if we have any inventory information there.
	var hasKptfile bool
	hasKptfile, err = IsPackageDir(p.fsys, p.UniquePath.String())
	if err != nil {
		return kptfilev1.Inventory{}, errors.E(op, p.UniquePath, err)
	}

	if !hasKptfile {
		// Return the ResourceGroup object as inventory.
		if len(resources) == 1 {
			return kptfilev1.Inventory{
				Name:        resources[0].GetName(),
				Namespace:   resources[0].GetNamespace(),
				InventoryID: resources[0].GetLabels()[rgfilev1alpha1.RGInventoryIDLabel],
			}, nil
		}

		// No inventory information found as ResourceGroup objects, and Kptfile does not exist.
		return kptfilev1.Inventory{}, &NoInvInfoError{}
	}

	kf, err := p.Kptfile()
	if err != nil {
		return kptfilev1.Inventory{}, errors.E(op, p.UniquePath, err)
	}

	// No inventory found in either Kptfile or as ResourceGroup objects.
	if kf.Inventory == nil && len(resources) == 0 {
		return kptfilev1.Inventory{}, &NoInvInfoError{}
	}

	// Multiple inventories found, in both Kptfile and resourcegroup objects.
	if kf.Inventory != nil && len(resources) > 0 {
		return kptfilev1.Inventory{}, &MultipleInventoryInfoError{}
	}

	// ResourceGroup stores the inventory and Kptfile does not contain inventory.
	if len(resources) == 1 {
		return kptfilev1.Inventory{
			Name:        resources[0].GetName(),
			Namespace:   resources[0].GetNamespace(),
			InventoryID: resources[0].GetLabels()[rgfilev1alpha1.RGInventoryIDLabel],
		}, nil
	}

	// Kptfile stores the inventory.
	fmt.Println(warnInvInKptfile)
	return *kf.Inventory, nil
}

// filterResourceGroups only retains ResourceGroup objects.
func filterResourceGroups(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	for _, r := range input {
		meta, err := r.GetMeta()
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata for resource %w", err)
		}
		// Filter out any non-ResourceGroup files.
		if !(meta.APIVersion == rgfilev1alpha1.ResourceGroupGVK().GroupVersion().String() && meta.Kind == rgfilev1alpha1.ResourceGroupGVK().Kind) {
			continue
		}

		output = append(output, r)
	}

	return output, nil
}
