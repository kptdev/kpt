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

package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func OpenRepository(name string, namespace string, content configapi.RepositoryContent, spec *configapi.OciRepository, cacheDir string) (repository.Repository, error) {
	storage, err := NewStorage(cacheDir)
	if err != nil {
		return nil, err
	}

	return &ociRepository{
		name:      name,
		namespace: namespace,
		content:   content,
		spec:      *spec.DeepCopy(),
		storage:   storage,
	}, nil

}

type ociRepository struct {
	name      string
	namespace string
	content   configapi.RepositoryContent
	spec      configapi.OciRepository

	storage *Storage
}

var _ repository.Repository = &ociRepository{}
var _ repository.FunctionRepository = &ociRepository{}

func (r *ociRepository) ListPackageRevisions(ctx context.Context) ([]repository.PackageRevision, error) {
	if r.content != configapi.RepositoryContentPackage {
		return []repository.PackageRevision{}, nil
	}

	ctx, span := tracer.Start(ctx, "ListPackageRevisions")
	defer span.End()

	ociRepo, err := name.NewRepository(r.spec.Registry)
	if err != nil {
		return nil, err
	}

	options := r.storage.createOptions(ctx)

	tags, err := google.List(ociRepo, options...)
	if err != nil {
		return nil, err
	}

	klog.Infof("tags: %#v", tags)

	var result []repository.PackageRevision
	for _, childName := range tags.Children {
		path := fmt.Sprintf("%s/%s", r.spec.Registry, childName)
		child, err := name.NewRepository(path, name.StrictValidation)
		if err != nil {
			klog.Warningf("Cannot create nested repository %q: %v", path, err)
			continue
		}

		childTags, err := google.List(child, options...)
		if err != nil {
			klog.Warningf("Cannot list nested repository %q: %v", path, err)
			continue
		}

		// klog.Infof("childTags: %#v", childTags)

		for digest, m := range childTags.Manifests {
			for _, tag := range m.Tags {
				created := m.Created
				if created.IsZero() {
					created = m.Uploaded
				}

				// ref := child.Tag(tag)
				// ref := child.Digest(digest)

				p := &ociPackageRevision{
					// tagName: ImageTagName{
					// 	Image: child.Name(),
					// 	Tag:   tag,
					// },
					digestName: ImageDigestName{
						Image:  child.Name(),
						Digest: digest,
					},
					packageName:     childName,
					revision:        tag,
					created:         created,
					parent:          r,
					resourceVersion: constructResourceVersion(m.Uploaded),
				}
				p.uid = constructUID(p.packageName + ":" + p.revision)

				tasks, err := r.loadTasks(ctx, p.digestName)
				if err != nil {
					return nil, err
				}
				p.tasks = tasks

				result = append(result, p)
			}
		}
	}

	return result, nil
}

func (r *ociRepository) ListFunctions(ctx context.Context) ([]repository.Function, error) {
	// Repository whose content type is not Function contains no Function resources.
	if r.content != configapi.RepositoryContentFunction {
		klog.Infof("Repository %q doesn't contain functions; contains %s", r.name, r.content)
		return []repository.Function{}, nil
	}

	ctx, span := tracer.Start(ctx, "ListFunctions")
	defer span.End()

	ociRepo, err := name.NewRepository(r.spec.Registry)
	if err != nil {
		return nil, err
	}

	options := r.storage.createOptions(ctx)

	result := []repository.Function{}

	err = google.Walk(ociRepo, func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			klog.Warningf(" Walk %s encountered error: %w", repo, err)
			return err
		}

		if tags == nil {
			return nil
		}

		if cl := len(tags.Children); cl > 0 {
			// Expect no manifests or tags
			if ml, tl := len(tags.Manifests), len(tags.Tags); ml != 0 || tl != 0 {
				return fmt.Errorf("OCI repository with children (%d) as well as Manifests (%d) or Tags (%d)", cl, ml, tl)
			}
			return nil
		}

		functionName := parseFunctionName(repo.RepositoryStr())

		for digest, manifest := range tags.Manifests {
			// Only consider tagged images.
			for _, tag := range manifest.Tags {

				created := manifest.Created
				if created.IsZero() {
					created = manifest.Uploaded
				}

				result = append(result, &ociFunction{
					ref:     repo.Digest(digest),
					tag:     repo.Tag(tag),
					name:    functionName,
					version: tag,
					created: created,
					parent:  r,
				})
			}
		}

		return nil
	}, options...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type ociPackageRevision struct {
	digestName      ImageDigestName
	packageName     string
	revision        string
	created         time.Time
	resourceVersion string
	uid             types.UID

	parent *ociRepository

	tasks []v1alpha1.Task
}

var _ repository.PackageRevision = &ociPackageRevision{}

func (p *ociPackageRevision) GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error) {
	resources, err := p.parent.storage.LoadResources(ctx, &p.digestName)
	if err != nil {
		return nil, err
	}

	resourceList, err := resources.AsResourceList()
	if err != nil {
		return nil, err
	}

	return &v1alpha1.PackageRevisionResources{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name(),
			Namespace: p.parent.namespace,
			CreationTimestamp: metav1.Time{
				Time: p.created,
			},
			ResourceVersion: p.resourceVersion,
			UID:             p.uid,
		},
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			Resources: resourceList,
		},
	}, nil
}

func (p *ociPackageRevision) Name() string {
	return p.parent.name + ":" + p.packageName + ":" + p.revision
}

func (p *ociPackageRevision) GetPackageRevision() (*v1alpha1.PackageRevision, error) {
	return &v1alpha1.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name(),
			Namespace: p.parent.namespace,
			CreationTimestamp: metav1.Time{
				Time: p.created,
			},
			ResourceVersion: p.resourceVersion,
			UID:             p.uid,
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    p.packageName,
			Revision:       p.revision,
			RepositoryName: p.parent.name,
			Tasks:          p.tasks,
		},
	}, nil
}
