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

package pkgbuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	deploymentResourceManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: myspace
  name: mysql-deployment
spec:
  replicas: 3
  foo: bar
  template:
    spec:
      containers:
      - name: mysql
        image: mysql:1.7.9
`

	configMapResourceManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap
data:
  foo: bar
`

	secretResourceManifest = `
apiVersion: v1
kind: Secret
metadata:
  name: secret
type: Opaque
data:
  foo: bar
`
)

var (
	DeploymentResource = "deployment"
	ConfigMapResource  = "configmap"
	SecretResource     = "secret"
	resources          = map[string]resourceInfo{
		DeploymentResource: {
			filename: "deployment.yaml",
			manifest: deploymentResourceManifest,
		},
		ConfigMapResource: {
			filename: "configmap.yaml",
			manifest: configMapResourceManifest,
		},
		SecretResource: {
			filename: "secret.yaml",
			manifest: secretResourceManifest,
		},
	}
)

// Pkg represents a package that can be created on the file system
// by using the Build function
type pkg struct {
	Kptfile *Kptfile

	RGFile *RGFile

	resources []resourceInfoWithMutators

	files map[string]string

	subPkgs []*SubPkg
}

// WithRGFile configures the current package to have a resourcegroup file.
func (rp *RootPkg) WithRGFile(rg *RGFile) *RootPkg {
	rp.pkg.RGFile = rg
	return rp
}

// withKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (p *pkg) withKptfile(kf ...*Kptfile) {
	if len(kf) > 1 {
		panic("only 0 or 1 Kptfiles are allowed")
	}
	if len(kf) == 0 {
		p.Kptfile = NewKptfile()
	} else {
		p.Kptfile = kf[0]
	}
}

// withResource configures the package to include the provided resource
func (p *pkg) withResource(resourceName string, mutators ...yaml.Filter) {
	resourceInfo, ok := resources[resourceName]
	if !ok {
		panic(fmt.Errorf("unknown resource %s", resourceName))
	}
	p.resources = append(p.resources, resourceInfoWithMutators{
		resourceInfo: resourceInfo,
		mutators:     mutators,
	})
}

// withRawResource configures the package to include the provided resource
func (p *pkg) withRawResource(resourceName, manifest string, mutators ...yaml.Filter) {
	p.resources = append(p.resources, resourceInfoWithMutators{
		resourceInfo: resourceInfo{
			filename: resourceName,
			manifest: manifest,
		},
		mutators: mutators,
	})
}

// withFile configures the package to contain a file with the provided name
// and the given content.
func (p *pkg) withFile(name, content string) {
	p.files[name] = content
}

// withSubPackages adds the provided packages as subpackages to the current
// package
func (p *pkg) withSubPackages(ps ...*SubPkg) {
	p.subPkgs = append(p.subPkgs, ps...)
}

// allReferencedRepos traverses the root package and all subpackages to
// capture all references to other repos.
func (p *pkg) allReferencedRepos(collector map[string]bool) {
	for i := range p.subPkgs {
		p.subPkgs[i].pkg.allReferencedRepos(collector)
	}
	if p.Kptfile != nil && p.Kptfile.Upstream != nil {
		collector[p.Kptfile.Upstream.RepoRef] = true
	}
}

// RootPkg is a package without any parent package.
type RootPkg struct {
	pkg *pkg
}

// NewRootPkg creates a new package for testing.
func NewRootPkg() *RootPkg {
	return &RootPkg{
		pkg: &pkg{
			files: make(map[string]string),
		},
	}
}

// WithKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (rp *RootPkg) WithKptfile(kf ...*Kptfile) *RootPkg {
	rp.pkg.withKptfile(kf...)
	return rp
}

// HasKptfile tells whether the package contains a Kptfile.
func (rp *RootPkg) HasKptfile() bool {
	return rp.pkg.Kptfile != nil
}

// AllReferencedRepos returns the name of all remote subpackages referenced
// in the package (including any local subpackages).
func (rp *RootPkg) AllReferencedRepos() []string {
	repoNameMap := make(map[string]bool)
	rp.pkg.allReferencedRepos(repoNameMap)

	var repoNames []string
	for n := range repoNameMap {
		repoNames = append(repoNames, n)
	}
	return repoNames
}

// WithResource configures the package to include the provided resource
func (rp *RootPkg) WithResource(resourceName string, mutators ...yaml.Filter) *RootPkg {
	rp.pkg.withResource(resourceName, mutators...)
	return rp
}

// WithRawResource configures the package to include the provided resource
func (rp *RootPkg) WithRawResource(resourceName, manifest string, mutators ...yaml.Filter) *RootPkg {
	rp.pkg.withRawResource(resourceName, manifest, mutators...)
	return rp
}

// WithFile configures the package to contain a file with the provided name
// and the given content.
func (rp *RootPkg) WithFile(name, content string) *RootPkg {
	rp.pkg.withFile(name, content)
	return rp
}

// WithSubPackages adds the provided packages as subpackages to the current
// package
func (rp *RootPkg) WithSubPackages(ps ...*SubPkg) *RootPkg {
	rp.pkg.withSubPackages(ps...)
	return rp
}

// Build outputs the current data structure as a set of (nested) package
// in the provided path.
func (rp *RootPkg) Build(path string, pkgName string, reposInfo ReposInfo) error {
	pkgPath := filepath.Join(path, pkgName)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}
	if rp == nil {
		return nil
	}
	err = buildPkg(pkgPath, rp.pkg, pkgName, reposInfo)
	if err != nil {
		return err
	}
	for i := range rp.pkg.subPkgs {
		subPkg := rp.pkg.subPkgs[i]
		err := buildSubPkg(pkgPath, subPkg, reposInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

// SubPkg is a subpackage, so it is contained inside another package. The
// name sets both the name of the directory in which the package is stored
// and the metadata.name field in the Kptfile (if there is one).
type SubPkg struct {
	pkg *pkg

	Name string
}

// NewSubPkg returns a new subpackage for testing.
func NewSubPkg(name string) *SubPkg {
	return &SubPkg{
		pkg: &pkg{
			files: make(map[string]string),
		},
		Name: name,
	}
}

// WithKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (sp *SubPkg) WithKptfile(kf ...*Kptfile) *SubPkg {
	sp.pkg.withKptfile(kf...)
	return sp
}

// WithResource configures the package to include the provided resource
func (sp *SubPkg) WithResource(resourceName string, mutators ...yaml.Filter) *SubPkg {
	sp.pkg.withResource(resourceName, mutators...)
	return sp
}

// WithRawResource configures the package to include the provided resource
func (sp *SubPkg) WithRawResource(resourceName, manifest string, mutators ...yaml.Filter) *SubPkg {
	sp.pkg.withRawResource(resourceName, manifest, mutators...)
	return sp
}

// WithFile configures the package to contain a file with the provided name
// and the given content.
func (sp *SubPkg) WithFile(name, content string) *SubPkg {
	sp.pkg.withFile(name, content)
	return sp
}

// WithSubPackages adds the provided packages as subpackages to the current
// package
func (sp *SubPkg) WithSubPackages(ps ...*SubPkg) *SubPkg {
	sp.pkg.withSubPackages(ps...)
	return sp
}

// RGFile represents a minimal resourcegroup.
type RGFile struct {
	Name, Namespace, ID string
}

func NewRGFile() *RGFile {
	return &RGFile{}
}

func (rg *RGFile) WithInventory(inv Inventory) *RGFile {
	rg.Name = inv.Name
	rg.Namespace = inv.Namespace
	rg.ID = inv.ID
	return rg
}

// Kptfile represents the Kptfile of a package.
type Kptfile struct {
	Upstream     *Upstream
	UpstreamLock *UpstreamLock
	Pipeline     *Pipeline
	Inventory    *Inventory
}

func NewKptfile() *Kptfile {
	return &Kptfile{}
}

// WithUpstream adds information about the upstream information to the Kptfile.
// The upstream section of the Kptfile is only added if this information is
// provided.
func (k *Kptfile) WithUpstream(repo, dir, ref, strategy string) *Kptfile {
	k.Upstream = &Upstream{
		Repo:     repo,
		Dir:      dir,
		Ref:      ref,
		Strategy: strategy,
	}
	return k
}

// WithUpstreamRef adds information about the upstream information to the
// Kptfile. Unlike WithUpstream, this function allows providing just a
// reference to the repo rather than the actual path. The reference will
// be resolved to an actual path when the package is written to disk.
func (k *Kptfile) WithUpstreamRef(repoRef, dir, ref, strategy string) *Kptfile {
	k.Upstream = &Upstream{
		RepoRef:  repoRef,
		Dir:      dir,
		Ref:      ref,
		Strategy: strategy,
	}
	return k
}

// WithUpstreamLock adds upstreamLock information to the Kptfile. If no
// upstreamLock information is provided,
func (k *Kptfile) WithUpstreamLock(repo, dir, ref, commit string) *Kptfile {
	k.UpstreamLock = &UpstreamLock{
		Repo:   repo,
		Dir:    dir,
		Ref:    ref,
		Commit: commit,
	}
	return k
}

// WithUpstreamLockRef adds upstreamLock information to the Kptfile. But unlike
// WithUpstreamLock, this function takes a the name to a repo and will resolve
// the actual path when expanding the package. The commit SHA is also not provided,
// but rather the index of a commit that will be resolved when expanding the
// package.
func (k *Kptfile) WithUpstreamLockRef(repoRef, dir, ref string, index int) *Kptfile {
	k.UpstreamLock = &UpstreamLock{
		RepoRef: repoRef,
		Dir:     dir,
		Ref:     ref,
		Index:   index,
	}
	return k
}

type Upstream struct {
	Repo     string
	RepoRef  string
	Dir      string
	Ref      string
	Strategy string
}

type UpstreamLock struct {
	Repo    string
	RepoRef string
	Dir     string
	Ref     string
	Index   int
	Commit  string
}

func (k *Kptfile) WithInventory(inv Inventory) *Kptfile {
	k.Inventory = &inv
	return k
}

type Inventory struct {
	Name      string
	Namespace string
	ID        string
}

func (k *Kptfile) WithPipeline(functions ...Function) *Kptfile {
	k.Pipeline = &Pipeline{
		Functions: functions,
	}
	return k
}

type Pipeline struct {
	Functions []Function
}

func NewFunction(image string) Function {
	return Function{
		Image: image,
	}
}

type Function struct {
	Image      string
	ConfigPath string
}

func (f Function) WithConfigPath(configPath string) Function {
	f.ConfigPath = configPath
	return f
}

// RemoteSubpackage contains information about remote subpackages that should
// be listed in the Kptfile.
type RemoteSubpackage struct {
	// Name is the name of the remote subpackage. It will be used as the value
	// for the LocalDir property and also used to resolve the Repo path from
	// other defined repos.
	RepoRef   string
	Repo      string
	Directory string
	Ref       string
	Strategy  string
	LocalDir  string
}

type resourceInfo struct {
	filename string
	manifest string
}

type resourceInfoWithMutators struct {
	resourceInfo resourceInfo
	mutators     []yaml.Filter
}

func buildSubPkg(path string, pkg *SubPkg, reposInfo ReposInfo) error {
	pkgPath := filepath.Join(path, pkg.Name)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}
	err = buildPkg(pkgPath, pkg.pkg, pkg.Name, reposInfo)
	if err != nil {
		return err
	}
	for i := range pkg.pkg.subPkgs {
		subPkg := pkg.pkg.subPkgs[i]
		err := buildSubPkg(pkgPath, subPkg, reposInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildPkg(pkgPath string, pkg *pkg, pkgName string, reposInfo ReposInfo) error {
	if pkg.Kptfile != nil {
		content := buildKptfile(pkg, pkgName, reposInfo)

		err := os.WriteFile(filepath.Join(pkgPath, kptfilev1.KptFileName),
			[]byte(content), 0600)
		if err != nil {
			return err
		}
	}

	if pkg.RGFile != nil {
		content := buildRGFile(pkg)

		err := os.WriteFile(filepath.Join(pkgPath, rgfilev1alpha1.RGFileName),
			[]byte(content), 0600)
		if err != nil {
			return err
		}
	}

	for _, ri := range pkg.resources {
		m := ri.resourceInfo.manifest
		r := yaml.MustParse(m)

		for _, m := range ri.mutators {
			if err := r.PipeE(m); err != nil {
				return err
			}
		}

		filePath := filepath.Join(pkgPath, ri.resourceInfo.filename)
		err := os.WriteFile(filePath, []byte(r.MustString()), 0600)
		if err != nil {
			return err
		}
	}

	for name, content := range pkg.files {
		filePath := filepath.Join(pkgPath, name)
		_, err := os.Stat(filePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("file %s already exists", name)
		}
		err = os.WriteFile(filePath, []byte(content), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

// buildRGFile creates a ResourceGroup inventory file.
func buildRGFile(pkg *pkg) string {
	tmp := rgfilev1alpha1.ResourceGroup{ResourceMeta: rgfilev1alpha1.DefaultMeta}
	tmp.ObjectMeta.Name = pkg.RGFile.Name
	tmp.ObjectMeta.Namespace = pkg.RGFile.Namespace
	if pkg.RGFile.ID != "" {
		tmp.ObjectMeta.Labels = map[string]string{rgfilev1alpha1.RGInventoryIDLabel: pkg.RGFile.ID}
	}

	b, err := yaml.MarshalWithOptions(tmp, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		panic(err)
	}

	return string(b)
}

type ReposInfo interface {
	ResolveRepoRef(repoRef string) (string, bool)
	ResolveCommitIndex(repoRef string, index int) (string, bool)
}

func buildKptfile(pkg *pkg, pkgName string, reposInfo ReposInfo) string {
	if pkg.Kptfile.Upstream != nil && len(pkg.Kptfile.Upstream.RepoRef) > 0 {
		repoRef := pkg.Kptfile.Upstream.RepoRef
		ref := pkg.Kptfile.Upstream.Ref
		pkg.Kptfile.Upstream.Repo = resolveRepoRef(repoRef, reposInfo)

		if newRef, ok := resolveCommitRef(repoRef, ref, reposInfo); ok {
			pkg.Kptfile.Upstream.Ref = newRef
		}
	}
	if pkg.Kptfile.UpstreamLock != nil && len(pkg.Kptfile.UpstreamLock.RepoRef) > 0 {
		repoRef := pkg.Kptfile.UpstreamLock.RepoRef
		ref := pkg.Kptfile.UpstreamLock.Ref
		pkg.Kptfile.UpstreamLock.Repo = resolveRepoRef(repoRef, reposInfo)

		index := pkg.Kptfile.UpstreamLock.Index
		pkg.Kptfile.UpstreamLock.Commit = resolveCommitIndex(repoRef, index, reposInfo)

		if newRef, ok := resolveCommitRef(repoRef, ref, reposInfo); ok {
			pkg.Kptfile.UpstreamLock.Ref = newRef
		}
	}

	kptfile := &kptfilev1.KptFile{}
	kptfile.APIVersion, kptfile.Kind = kptfilev1.KptFileGVK().ToAPIVersionAndKind()
	kptfile.ObjectMeta.Name = pkgName
	if pkg.Kptfile.Upstream != nil {
		kptfile.Upstream = &kptfilev1.Upstream{
			Type: "git",
			Git: &kptfilev1.Git{
				Repo:      pkg.Kptfile.Upstream.Repo,
				Directory: pkg.Kptfile.Upstream.Dir,
				Ref:       pkg.Kptfile.Upstream.Ref,
			},
			UpdateStrategy: kptfilev1.UpdateStrategyType(pkg.Kptfile.Upstream.Strategy),
		}
	}
	if pkg.Kptfile.UpstreamLock != nil {
		kptfile.UpstreamLock = &kptfilev1.UpstreamLock{
			Type: "git",
			Git: &kptfilev1.GitLock{
				Repo:      pkg.Kptfile.UpstreamLock.Repo,
				Directory: pkg.Kptfile.UpstreamLock.Dir,
				Ref:       pkg.Kptfile.UpstreamLock.Ref,
				Commit:    pkg.Kptfile.UpstreamLock.Commit,
			},
		}
	}
	if pkg.Kptfile.Pipeline != nil {
		kptfile.Pipeline = &kptfilev1.Pipeline{}
		for _, fn := range pkg.Kptfile.Pipeline.Functions {
			mutator := kptfilev1.Function{
				Image: fn.Image,
			}
			if fn.ConfigPath != "" {
				mutator.ConfigPath = fn.ConfigPath
			}
			kptfile.Pipeline.Mutators = append(kptfile.Pipeline.Mutators, mutator)
		}
	}

	if inventory := pkg.Kptfile.Inventory; inventory != nil {
		kptfile.Inventory = &kptfilev1.Inventory{}
		if inventory.Name != "" {
			kptfile.Inventory.Name = inventory.Name
		}
		if inventory.Namespace != "" {
			kptfile.Inventory.Namespace = inventory.Namespace
		}
		if inventory.ID != "" {
			kptfile.Inventory.InventoryID = inventory.ID
		}
	}
	b, err := yaml.Marshal(kptfile)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// resolveRepoRef looks up the repo path for a repo from the reposInfo
// object based on the provided reference.
func resolveRepoRef(repoRef string, reposInfo ReposInfo) string {
	repo, found := reposInfo.ResolveRepoRef(repoRef)
	if !found {
		panic(fmt.Errorf("path for package %s not found", repoRef))
	}
	return repo
}

// resolveCommitIndex looks up the commit SHA for a specific commit in a repo.
// It looks up the repo based on the provided repoRef and returns the commit for
// the commit with the provided index.
func resolveCommitIndex(repoRef string, index int, reposInfo ReposInfo) string {
	commit, found := reposInfo.ResolveCommitIndex(repoRef, index)
	if !found {
		panic(fmt.Errorf("can't find commit for index %d in repo %s", index, repoRef))
	}
	return commit
}

// resolveCommitRef looks up the commit SHA for a commit with the index given
// through a special string format as the ref. If the string value follows the
// correct format, the commit will looked up from the repo given by the RepoRef
// and returned with the second value being true. If the ref string does not
// follow the correct format, the second return value will be false.
func resolveCommitRef(repoRef, ref string, reposInfo ReposInfo) (string, bool) {
	re := regexp.MustCompile(`^COMMIT-INDEX:([0-9]+)$`)
	matches := re.FindStringSubmatch(ref)
	if len(matches) != 2 {
		return "", false
	}
	index, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", false
	}
	return resolveCommitIndex(repoRef, index, reposInfo), true
}

// ExpandPkg writes the provided package to disk. The name of the root package
// will just be set to "base".
func (rp *RootPkg) ExpandPkg(t *testing.T, reposInfo ReposInfo) string {
	return rp.ExpandPkgWithName(t, "base", reposInfo)
}

// ExpandPkgWithName writes the provided package to disk and uses the given
// rootName to set the value of the package directory and the metadata.name
// field of the root package.
func (rp *RootPkg) ExpandPkgWithName(t *testing.T, rootName string, reposInfo ReposInfo) string {
	dir, err := os.MkdirTemp("", "test-kpt-builder-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = rp.Build(dir, rootName, reposInfo)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return filepath.Join(dir, rootName)
}
