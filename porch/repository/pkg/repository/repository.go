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

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
)

// TODO: 	"sigs.k8s.io/kustomize/kyaml/filesys" FileSystem?
type PackageResources struct {
	Contents map[string]string
}

// PackageRevision is an abstract package version.
// We have a single object for both Revision and Resources, because conceptually they are one object.
// The best way we've found (so far) to represent them in k8s is as two resources, but they map to the same object.
// Interesting reading: https://github.com/zecke/Kubernetes/blob/master/docs/devel/api-conventions.md#differing-representations
type PackageRevision interface {
	Name() string
	GetPackageRevision() (*v1alpha1.PackageRevision, error)
	// TODO: return PackageResources or filesystem abstraction?
	GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error)
}

type PackageDraft interface {
	UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, task *v1alpha1.Task) error
	// Finish round of updates.
	Close(ctx context.Context) (PackageRevision, error)
}

// Function is an abstract function.
type Function interface {
	Name() string
	GetFunction() (*v1alpha1.Function, error)
}

// Repository is the interface for interacting with packages in repositories
// TODO: we may need interface to manage repositories too. Stay tuned.
type Repository interface {
	ListPackageRevisions(ctx context.Context) ([]PackageRevision, error)

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

type Credential struct {
	// TODO: support different credential types
	Data map[string][]byte
}

type CredentialResolver interface {
	ResolveCredential(ctx context.Context, namespace, name string) (Credential, error)
}
