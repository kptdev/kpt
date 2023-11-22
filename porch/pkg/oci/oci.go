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

package oci

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/oci"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func OpenRepository(name string, namespace string, content configapi.RepositoryContent, spec *configapi.OciRepository, deployment bool, storage *oci.Storage) (repository.Repository, error) {
	return &ociRepository{
		name:       name,
		namespace:  namespace,
		content:    content,
		spec:       *spec.DeepCopy(),
		deployment: deployment,
		storage:    storage,
	}, nil

}

type ociRepository struct {
	name       string
	namespace  string
	content    configapi.RepositoryContent
	spec       configapi.OciRepository
	deployment bool

	storage *oci.Storage
}

var _ repository.Repository = &ociRepository{}
var _ repository.FunctionRepository = &ociRepository{}

func (r *ociRepository) Close() error {
	return nil
}

// there is probably a more efficient way to do this
func (r *ociRepository) Version(ctx context.Context) (string, error) {
	ctx, span := tracer.Start(ctx, "ociRepository::Version")
	defer span.End()

	if r.content != configapi.RepositoryContentPackage {
		return "", nil
	}

	ociRepo, err := name.NewRepository(r.spec.Registry)
	if err != nil {
		return "", err
	}

	options := r.storage.CreateOptions(ctx)

	tags, err := google.List(ociRepo, options...)
	if err != nil {
		return "", err
	}

	klog.Infof("tags: %#v", tags)

	b := bytes.Buffer{}
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
			b.WriteString(digest)
			mb, err := m.MarshalJSON()
			if err != nil {
				return "", err
			}
			b.Write(mb)
		}
	}
	hash := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(hash[:]), nil
}

func (r *ociRepository) ListPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	if r.content != configapi.RepositoryContentPackage {
		return []repository.PackageRevision{}, nil
	}

	ctx, span := tracer.Start(ctx, "ociRepository::ListPackageRevisions")
	defer span.End()

	ociRepo, err := name.NewRepository(r.spec.Registry)
	if err != nil {
		return nil, err
	}

	options := r.storage.CreateOptions(ctx)

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
					digestName: oci.ImageDigestName{
						Image:  child.Name(),
						Digest: digest,
					},
					packageName:     childName,
					workspaceName:   v1alpha1.WorkspaceName(tag),
					created:         created,
					parent:          r,
					resourceVersion: constructResourceVersion(m.Created),
				}
				p.uid = constructUID(p.packageName + ":" + string(p.workspaceName))

				lifecycle, err := r.getLifecycle(ctx, p.digestName)
				if err != nil {
					return nil, err
				}
				p.lifecycle = lifecycle

				revision, err := r.getRevisionNumber(ctx, p.digestName)
				if err != nil {
					return nil, err
				}
				p.revision = revision

				tasks, err := r.loadTasks(ctx, p.digestName)
				if err != nil {
					return nil, err
				}
				p.tasks = tasks

				if filter.Matches(p) {
					result = append(result, p)
				}
			}
		}
	}

	return result, nil
}

func (r *ociRepository) ListPackages(ctx context.Context, filter repository.ListPackageFilter) ([]repository.Package, error) {
	return nil, fmt.Errorf("ListPackages not supported for OCI packages")
}

func (r *ociRepository) buildPackageRevision(ctx context.Context, name oci.ImageDigestName, packageName string,
	workspace v1alpha1.WorkspaceName, revision string, created time.Time) (repository.PackageRevision, error) {
	if r.content != configapi.RepositoryContentPackage {
		return nil, fmt.Errorf("repository is not a package repo, type is %v", r.content)
	}

	ctx, span := tracer.Start(ctx, "ociRepository::buildPackageRevision")
	defer span.End()

	// for backwards compatibility with packages that existed before porch supported
	// workspaces, we populate the workspaceName as the revision number if it is empty
	if workspace == "" {
		workspace = v1alpha1.WorkspaceName(revision)
	}

	p := &ociPackageRevision{
		digestName:      name,
		packageName:     packageName,
		workspaceName:   workspace,
		revision:        revision,
		created:         created,
		parent:          r,
		resourceVersion: constructResourceVersion(created),
	}
	p.uid = constructUID(p.packageName + ":" + string(p.workspaceName))

	lifecycle, err := r.getLifecycle(ctx, p.digestName)
	if err != nil {
		return nil, err
	}
	p.lifecycle = lifecycle

	tasks, err := r.loadTasks(ctx, p.digestName)
	if err != nil {
		return nil, err
	}
	p.tasks = tasks

	return p, nil
}

func GetFunctionMeta(reference string, ctx context.Context) (*functionMeta, error) {
	ref, err := name.ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("parse image reference %v: %v", reference, err)
	}
	image, err := remote.Image(ref, remote.WithAuthFromKeychain(gcrane.Keychain), remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("pull remote image %v: %v", reference, err)
	}
	manifest, err := image.Manifest()
	if err != nil {
		return nil, fmt.Errorf("get manifest from image %v: %v", reference, err)
	}
	return &functionMeta{
		FunctionTypes:    GetSliceFromAnnotation(FunctionTypesKey, manifest),
		Description:      GetSingleFromAnnotation(DescriptionKey, manifest),
		DocumentationUrl: GetSingleFromAnnotation(DocumentationURLKey, manifest),
		Keywords:         GetSliceFromAnnotation(keywordsKey, manifest),
		FunctionConfigs:  GetDefaultFunctionConfig(manifest),
	}, nil
}

func GetDefaultFunctionConfig(manifest *v1.Manifest) []functionConfig {
	val, ok := manifest.Annotations[ConfigMapFnKey]
	if !ok {
		return nil
	}
	return []functionConfig{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			RequiredFields: AnnotationToSlice(val),
		},
	}
}

func GetSliceFromAnnotation(key string, manifest *v1.Manifest) []string {
	slice, ok := manifest.Annotations[key]
	if !ok {
		return nil
	}
	return AnnotationToSlice(slice)
}

func GetSingleFromAnnotation(key string, manifest *v1.Manifest) string {
	if val, ok := manifest.Annotations[key]; ok {
		return val
	}
	return fmt.Sprintf("annotation %v unset", key)
}

func (r *ociRepository) ListFunctions(ctx context.Context) ([]repository.Function, error) {
	// Repository whose content type is not Function contains no Function resources.
	if r.content != configapi.RepositoryContentFunction {
		klog.Infof("Repository %q doesn't contain functions; contains %s", r.name, r.content)
		return []repository.Function{}, nil
	}

	ctx, span := tracer.Start(ctx, "ociRepository::ListFunctions")
	defer span.End()

	ociRepo, err := name.NewRepository(r.spec.Registry)
	if err != nil {
		return nil, err
	}

	options := r.storage.CreateOptions(ctx)

	result := []repository.Function{}

	err = google.Walk(ociRepo, func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			klog.Warningf(" Walk %s encountered error: %v", repo, err)
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
				meta, err := GetFunctionMeta(repo.Digest(digest).Name(), ctx)
				if err != nil {
					klog.Warningf(" pull function %v error: %w", functionName, err)
					continue
				}
				result = append(result, &ociFunction{
					ref:     repo.Digest(digest),
					tag:     repo.Tag(tag),
					name:    functionName,
					version: tag,
					meta:    meta,
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
	digestName      oci.ImageDigestName
	packageName     string
	revision        string
	workspaceName   v1alpha1.WorkspaceName
	created         time.Time
	resourceVersion string
	uid             types.UID

	parent *ociRepository
	tasks  []v1alpha1.Task

	lifecycle v1alpha1.PackageRevisionLifecycle
}

func (p *ociPackageRevision) CachedIdentifier() repository.CachedIdentifier {
	return repository.CachedIdentifier{Key: p.packageName + ":" + string(p.workspaceName), Version: p.resourceVersion}
}

var _ repository.PackageRevision = &ociPackageRevision{}

func (p *ociPackageRevision) GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error) {
	resources, err := LoadResources(ctx, p.parent.storage, &p.digestName)
	if err != nil {
		return nil, err
	}

	key := p.Key()

	return &v1alpha1.PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResources",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.KubeObjectName(),
			Namespace: p.parent.namespace,
			CreationTimestamp: metav1.Time{
				Time: p.created,
			},
			ResourceVersion: p.resourceVersion,
			UID:             p.uid,
		},
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			PackageName:    key.Package,
			WorkspaceName:  key.WorkspaceName,
			Revision:       key.Revision,
			RepositoryName: key.Repository,

			Resources: resources.Contents,
		},
	}, nil
}

func (p *ociPackageRevision) KubeObjectName() string {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%s", p.parent.name, p.packageName, p.workspaceName)))
	return p.parent.name + "-" + hex.EncodeToString(hash[:])
}

func (p *ociPackageRevision) KubeObjectNamespace() string {
	return p.parent.namespace
}

func (p *ociPackageRevision) UID() types.UID {
	return p.uid
}

func (p *ociPackageRevision) ResourceVersion() string {
	return p.resourceVersion
}

func (p *ociPackageRevision) Key() repository.PackageRevisionKey {
	return repository.PackageRevisionKey{
		Repository:    p.parent.name,
		Package:       p.packageName,
		Revision:      p.revision,
		WorkspaceName: p.workspaceName,
	}
}

func (p *ociPackageRevision) GetPackageRevision(ctx context.Context) (*v1alpha1.PackageRevision, error) {
	key := p.Key()

	kf, err := p.GetKptfile(ctx)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.KubeObjectName(),
			Namespace: p.parent.namespace,
			CreationTimestamp: metav1.Time{
				Time: p.created,
			},
			ResourceVersion: p.resourceVersion,
			UID:             p.uid,
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    key.Package,
			RepositoryName: key.Repository,
			Revision:       key.Revision,
			WorkspaceName:  key.WorkspaceName,

			Lifecycle:      p.Lifecycle(),
			Tasks:          p.tasks,
			ReadinessGates: repository.ToApiReadinessGates(kf),
		},
		Status: v1alpha1.PackageRevisionStatus{
			// TODO:        UpstreamLock,
			Deployment: p.parent.deployment,
			Conditions: repository.ToApiConditions(kf),
		},
	}, nil
}

func (p *ociPackageRevision) GetKptfile(ctx context.Context) (kptfile.KptFile, error) {
	resources, err := LoadResources(ctx, p.parent.storage, &p.digestName)
	if err != nil {
		return kptfile.KptFile{}, fmt.Errorf("error loading package resources for %v: %w", p.digestName, err)
	}
	kfString, found := resources.Contents[kptfile.KptFileName]
	if !found {
		return kptfile.KptFile{}, fmt.Errorf("packagerevision does not have a Kptfile")
	}
	kf, err := pkg.DecodeKptfile(strings.NewReader(kfString))
	if err != nil {
		return kptfile.KptFile{}, fmt.Errorf("error decoding Kptfile: %w", err)
	}
	return *kf, nil
}

func (p *ociPackageRevision) GetUpstreamLock(context.Context) (kptfile.Upstream, kptfile.UpstreamLock, error) {
	return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("UpstreamLock is not supported for OCI packages (%s)", p.KubeObjectName())
}

func (p *ociPackageRevision) GetLock() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("Lock is not supported for OCI packages (%s)", p.KubeObjectName())
}

func (p *ociPackageRevision) Lifecycle() v1alpha1.PackageRevisionLifecycle {
	return p.lifecycle
}

// UpdateLifecycle should update the package revision lifecycle from DeletionProposed to Published or vice versa.
//
//	This function is currently only partially implemented; it still needs to store whether the package has been
//	proposed for deletion somewhere in OCI, probably as another OCI image with a "deletionProposed" tag.
func (p *ociPackageRevision) UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error {
	old := p.Lifecycle()

	if old == v1alpha1.PackageRevisionLifecyclePublished {
		if new != v1alpha1.PackageRevisionLifecycleDeletionProposed {
			return fmt.Errorf("invalid new lifecycle value: %q", new)
		}

		// TODO: Create a "deletionProposed" OCI image tag.
		p.lifecycle = v1alpha1.PackageRevisionLifecycleDeletionProposed
	}
	if old == v1alpha1.PackageRevisionLifecycleDeletionProposed {
		if new != v1alpha1.PackageRevisionLifecyclePublished {
			return fmt.Errorf("invalid new lifecycle value: %q", new)
		}

		// TODO: Delete the "deletionProposed" OCI image tag.
		p.lifecycle = v1alpha1.PackageRevisionLifecyclePublished
	}
	return nil
}
