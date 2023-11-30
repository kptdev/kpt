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

package fake

import (
	"context"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"k8s.io/apimachinery/pkg/types"
)

// Implementation of the repository.PackageRevision interface for testing.
type PackageRevision struct {
	Name               string
	Namespace          string
	Uid                types.UID
	PackageRevisionKey repository.PackageRevisionKey
	PackageLifecycle   v1alpha1.PackageRevisionLifecycle
	PackageRevision    *v1alpha1.PackageRevision
	Resources          *v1alpha1.PackageRevisionResources
	Kptfile            kptfile.KptFile
}

func (pr *PackageRevision) CachedIdentifier() repository.CachedIdentifier {
	return repository.CachedIdentifier{Key: pr.Key().String(), Version: pr.Key().Revision}
}

func (pr *PackageRevision) KubeObjectName() string {
	return pr.Name
}

func (pr *PackageRevision) KubeObjectNamespace() string {
	return pr.Namespace
}

func (pr *PackageRevision) UID() types.UID {
	return pr.Uid
}

func (pr *PackageRevision) ResourceVersion() string {
	return pr.PackageRevision.ResourceVersion
}

func (pr *PackageRevision) Key() repository.PackageRevisionKey {
	return pr.PackageRevisionKey
}

func (pr *PackageRevision) Lifecycle() v1alpha1.PackageRevisionLifecycle {
	return pr.PackageLifecycle
}

func (pr *PackageRevision) GetPackageRevision(context.Context) (*v1alpha1.PackageRevision, error) {
	return pr.PackageRevision, nil
}

func (f *PackageRevision) GetResources(context.Context) (*v1alpha1.PackageRevisionResources, error) {
	return f.Resources, nil
}

func (f *PackageRevision) GetKptfile(ctx context.Context) (kptfile.KptFile, error) {
	return f.Kptfile, nil
}

func (f *PackageRevision) GetUpstreamLock(context.Context) (kptfile.Upstream, kptfile.UpstreamLock, error) {
	return *f.Kptfile.Upstream, *f.Kptfile.UpstreamLock, nil
}

func (f *PackageRevision) GetLock() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	return *f.Kptfile.Upstream, *f.Kptfile.UpstreamLock, nil
}

func (f *PackageRevision) UpdateLifecycle(context.Context, v1alpha1.PackageRevisionLifecycle) error {
	return nil
}
