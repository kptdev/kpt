// Copyright 2022 Google LLC
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
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// TODO: 	"sigs.k8s.io/kustomize/kyaml/filesys" FileSystem?
type PackageResources struct {
	Contents map[string]string
}

type PackageRevisionKey struct {
	Repository, Package, Revision string
}

func (n PackageRevisionKey) String() string {
	return fmt.Sprintf("Repository: %q, Package: %q, Revision: %q", n.Repository, n.Package, n.Revision)
}

// PackageRevision is an abstract package version.
// We have a single object for both Revision and Resources, because conceptually they are one object.
// The best way we've found (so far) to represent them in k8s is as two resources, but they map to the same object.
// Interesting reading: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#differing-representations
type PackageRevision interface {
	// KubeObjectName returns an encoded name for the object that should be unique.
	// More "readable" values are returned by Key()
	KubeObjectName() string

	// Key returns the "primary key" of the package.
	Key() PackageRevisionKey

	// Lifecycle returns the current lifecycle state of the package.
	Lifecycle() v1alpha1.PackageRevisionLifecycle

	// GetPackageRevision returns the PackageRevision ("DRY") API representation of this package-revision
	GetPackageRevision() *v1alpha1.PackageRevision

	// GetResources returns the PackageRevisionResources ("WET") API representation of this package-revision
	// TODO: return PackageResources or filesystem abstraction?
	GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error)

	// GetUpstreamLock returns the kpt lock information.
	GetUpstreamLock() (kptfile.Upstream, kptfile.UpstreamLock, error)

	// GetLock returns the current revision's lock information.
	// This will be the upstream info for downstream revisions.
	GetLock() (kptfile.Upstream, kptfile.UpstreamLock, error)
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
}

// ListPackageRevisionFilter is a predicate for filtering PackageRevision objects;
// only matching PackageRevision objects will be returned.
type ListPackageRevisionFilter struct {
	// KubeObjectName matches the generated kubernetes object name.
	KubeObjectName string

	// Package matches the name of the package (spec.package)
	Package string

	// Revision matches the revision of the package (spec.revision)
	Revision string
}

// Matches returns true if the provided PackageRevision satisifies the conditions in the filter.
func (f *ListPackageRevisionFilter) Matches(p PackageRevision) bool {
	if f.Package != "" && f.Package != p.Key().Package {
		return false
	}
	if f.Revision != "" && f.Revision != p.Key().Revision {
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
	ListPackageRevisions(ctx context.Context, filter ListPackageRevisionFilter) ([]PackageRevision, error)

	// CreatePackageRevision creates a new package revision
	CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (PackageDraft, error)

	// DeletePackageRevision deletes a package revision
	DeletePackageRevision(ctx context.Context, old PackageRevision) error

	// UpdatePackage updates a package
	UpdatePackage(ctx context.Context, old PackageRevision) (PackageDraft, error)
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
