// Copyright 2022 The kpt Authors
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

package repository

import (
	"context"
	"fmt"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"k8s.io/apimachinery/pkg/types"
)

// TODO: 	"sigs.k8s.io/kustomize/kyaml/filesys" FileSystem?
type PackageResources struct {
	Contents map[string]string
}

type PackageRevisionKey struct {
	Repository, Package, Revision string
	WorkspaceName                 v1alpha1.WorkspaceName
}

func (n PackageRevisionKey) String() string {
	return fmt.Sprintf("Repository: %q, Package: %q, Revision: %q, WorkspaceName: %q",
		n.Repository, n.Package, n.Revision, string(n.WorkspaceName))
}

type PackageKey struct {
	Repository, Package string
}

func (n PackageKey) String() string {
	return fmt.Sprintf("Repository: %q, Package: %q", n.Repository, n.Package)
}

// CachedIdentier is a used by a cache and underlying storage
// implementation to avoid unnecessary reloads
type CachedIdentifier struct {
	// Key uniquely identifies the resource in the underlying storage
	Key string

	// Version uniquely identifies the version of the resource in the underlying storage
	Version string
}

type PackageRevisionCacheEntry struct {
	Version         string
	PackageRevision PackageRevision
}

type PackageRevisionCache map[string]PackageRevisionCacheEntry

type packageCacheKey struct{}

func ContextWithPackageRevisionCache(ctx context.Context, cache PackageRevisionCache) context.Context {
	return context.WithValue(ctx, packageCacheKey{}, cache)
}

func PackageRevisionCacheFromContext(ctx context.Context) PackageRevisionCache {
	cache, ok := ctx.Value(packageCacheKey{}).(PackageRevisionCache)
	if !ok {
		cache = make(PackageRevisionCache)
	}
	return cache
}

// PackageRevision is an abstract package version.
// We have a single object for both Revision and Resources, because conceptually they are one object.
// The best way we've found (so far) to represent them in k8s is as two resources, but they map to the same object.
// Interesting reading: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#differing-representations
type PackageRevision interface {
	// KubeObjectName returns an encoded name for the object that should be unique.
	// More "readable" values are returned by Key()
	KubeObjectName() string

	// KubeObjectNamespace returns the namespace in which the PackageRevision
	// belongs.
	KubeObjectNamespace() string

	// UID returns a unique identifier for the PackageRevision.
	UID() types.UID

	// Key returns the "primary key" of the package.
	Key() PackageRevisionKey

	// CachedIdentier returns a unique identifer for this package revision and version
	CachedIdentifier() CachedIdentifier

	// Lifecycle returns the current lifecycle state of the package.
	Lifecycle() v1alpha1.PackageRevisionLifecycle

	// UpdateLifecycle updates the desired lifecycle of the package. This can only
	// be used for Published package revisions to go from Published to DeletionProposed
	// or vice versa. Draft revisions should use PackageDraft.UpdateLifecycle.
	UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error

	// GetPackageRevision returns the PackageRevision ("DRY") API representation of this package-revision
	GetPackageRevision(context.Context) (*v1alpha1.PackageRevision, error)

	// GetResources returns the PackageRevisionResources ("WET") API representation of this package-revision
	// TODO: return PackageResources or filesystem abstraction?
	GetResources(context.Context) (*v1alpha1.PackageRevisionResources, error)

	// GetUpstreamLock returns the kpt lock information.
	GetUpstreamLock(context.Context) (kptfile.Upstream, kptfile.UpstreamLock, error)

	// GetKptfile returns the Kptfile for hte package
	GetKptfile(context.Context) (kptfile.KptFile, error)

	// GetLock returns the current revision's lock information.
	// This will be the upstream info for downstream revisions.
	GetLock() (kptfile.Upstream, kptfile.UpstreamLock, error)

	// ResourceVersion returns the Kube resource version of the package
	ResourceVersion() string
}

// Package is an abstract package.
type Package interface {
	// KubeObjectName returns an encoded name for the object that should be unique.
	// More "readable" values are returned by Key()
	KubeObjectName() string

	// Key returns the "primary key" of the package.
	Key() PackageKey

	// GetPackage returns the object representing this package
	GetPackage() *v1alpha1.Package

	// GetLatestRevision returns the name of the package revision that is the "latest" package
	// revision belonging to this package
	GetLatestRevision() string
}

type PackageDraft interface {
	UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, task *v1alpha1.Task) error
	// Updates desired lifecycle of the package. The lifecycle is applied on Close.
	UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error
	// Finish round of updates.
	Close(ctx context.Context) (PackageRevision, error)
}

// Function is an abstract function.
type Function interface {
	Name() string
	GetFunction() (*v1alpha1.Function, error)
	GetCRD() (*configapi.Function, error)
}

// ListPackageRevisionFilter is a predicate for filtering PackageRevision objects;
// only matching PackageRevision objects will be returned.
type ListPackageRevisionFilter struct {
	// KubeObjectName matches the generated kubernetes object name.
	KubeObjectName string

	// Package matches the name of the package (spec.package)
	Package string

	// WorkspaceName matches the description of the package (spec.workspaceName)
	WorkspaceName v1alpha1.WorkspaceName

	// Revision matches the revision of the package (spec.revision)
	Revision string
}

// Matches returns true if the provided PackageRevision satisfies the conditions in the filter.
func (f *ListPackageRevisionFilter) Matches(p PackageRevision) bool {
	if f.Package != "" && f.Package != p.Key().Package {
		return false
	}
	if f.Revision != "" && f.Revision != p.Key().Revision {
		return false
	}
	if f.WorkspaceName != "" && f.WorkspaceName != p.Key().WorkspaceName {
		return false
	}
	if f.KubeObjectName != "" && f.KubeObjectName != p.KubeObjectName() {
		return false
	}
	return true
}

// ListPackageFilter is a predicate for filtering Package objects;
// only matching Package objects will be returned.
type ListPackageFilter struct {
	// KubeObjectName matches the generated kubernetes object name.
	KubeObjectName string

	// Package matches the name of the package (spec.package)
	Package string
}

// Matches returns true if the provided Package satisfies the conditions in the filter.
func (f *ListPackageFilter) Matches(p Package) bool {
	if f.Package != "" && f.Package != p.Key().Package {
		return false
	}
	if f.KubeObjectName != "" && f.KubeObjectName != p.KubeObjectName() {
		return false
	}
	return true
}

// Repository is the interface for interacting with packages in repositories
// TODO: we may need interface to manage repositories too. Stay tuned.
type Repository interface {
	// ListPackageRevisions lists the existing package revisions in the repository
	ListPackageRevisions(ctx context.Context, filter ListPackageRevisionFilter) ([]PackageRevision, error)

	// CreatePackageRevision creates a new package revision
	CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (PackageDraft, error)

	// DeletePackageRevision deletes a package revision
	DeletePackageRevision(ctx context.Context, old PackageRevision) error

	// UpdatePackageRevision updates a package
	UpdatePackageRevision(ctx context.Context, old PackageRevision) (PackageDraft, error)

	// ListPackages lists all packages in the repository
	ListPackages(ctx context.Context, filter ListPackageFilter) ([]Package, error)

	// CreatePackage creates a new package
	CreatePackage(ctx context.Context, obj *v1alpha1.Package) (Package, error)

	// DeletePackage deletes a package
	DeletePackage(ctx context.Context, old Package) error

	// Version returns a string that is guaranteed to be different if any change has been made to the repo contents
	Version(ctx context.Context) (string, error)

	// Close cleans up any resources associated with the repository
	Close() error
}

type FunctionRepository interface {
	// TODO: Should repository understand functions, or just packages (and function is just a package in an OCI repo?)
	ListFunctions(ctx context.Context) ([]Function, error)
}

// The definitions below would be more appropriately located in a package usable by any Porch component.
// They are located in repository package because repository is one such package though thematically
// they rather belong to a package of their own.

type Credential interface {
	Valid() bool
	ToAuthMethod() transport.AuthMethod
}

type CredentialResolver interface {
	ResolveCredential(ctx context.Context, namespace, name string) (Credential, error)
}

type UserInfo struct {
	Name  string
	Email string
}

// UserInfoProvider providers name of the authenticated user on whose behalf the request
// is being processed.
type UserInfoProvider interface {
	// GetUserInfo returns the information about the user on whose behalf the request is being
	// processed, if any. If user cannot be determnined, returns nil.
	GetUserInfo(ctx context.Context) *UserInfo
}
