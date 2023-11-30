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

package engine

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/builtins"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/cache"
	"github.com/GoogleContainerTools/kpt/porch/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/GoogleContainerTools/kpt/porch/pkg/objects"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/comments"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var tracer = otel.Tracer("engine")

const (
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

type CaDEngine interface {
	// ObjectCache() is a cache of all our objects.
	ObjectCache() WatcherManager

	UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *PackageRevision, old, new *api.PackageRevisionResources) (*PackageRevision, *api.RenderStatus, error)
	ListFunctions(ctx context.Context, repositoryObj *configapi.Repository) ([]*Function, error)

	ListPackageRevisions(ctx context.Context, repositorySpec *configapi.Repository, filter repository.ListPackageRevisionFilter) ([]*PackageRevision, error)
	CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj *api.PackageRevision, parent *PackageRevision) (*PackageRevision, error)
	UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *PackageRevision, old, new *api.PackageRevision, parent *PackageRevision) (*PackageRevision, error)
	DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj *PackageRevision) error

	ListPackages(ctx context.Context, repositorySpec *configapi.Repository, filter repository.ListPackageFilter) ([]*Package, error)
	CreatePackage(ctx context.Context, repositoryObj *configapi.Repository, obj *api.Package) (*Package, error)
	UpdatePackage(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *Package, old, new *api.Package) (*Package, error)
	DeletePackage(ctx context.Context, repositoryObj *configapi.Repository, obj *Package) error
}

type Package struct {
	repoPackage repository.Package
}

func (p *Package) GetPackage() *api.Package {
	return p.repoPackage.GetPackage()
}

func (p *Package) KubeObjectName() string {
	return p.repoPackage.KubeObjectName()
}

// TODO: This is a bit awkward, and we should see if there is a way to avoid
// having to expose this function. Any functionality that requires creating new
// engine.PackageRevision resources should be in the engine package.
func ToPackageRevision(pkgRev repository.PackageRevision, pkgRevMeta meta.PackageRevisionMeta) *PackageRevision {
	return &PackageRevision{
		repoPackageRevision: pkgRev,
		packageRevisionMeta: pkgRevMeta,
	}
}

type PackageRevision struct {
	repoPackageRevision repository.PackageRevision
	packageRevisionMeta meta.PackageRevisionMeta
}

func (p *PackageRevision) GetPackageRevision(ctx context.Context) (*api.PackageRevision, error) {
	repoPkgRev, err := p.repoPackageRevision.GetPackageRevision(ctx)
	if err != nil {
		return nil, err
	}
	var isLatest bool
	if val, found := repoPkgRev.Labels[api.LatestPackageRevisionKey]; found && val == api.LatestPackageRevisionValue {
		isLatest = true
	}
	repoPkgRev.Labels = p.packageRevisionMeta.Labels
	if isLatest {
		// copy the labels in case the cached object is being read by another go routine
		labels := make(map[string]string, len(repoPkgRev.Labels))
		for k, v := range repoPkgRev.Labels {
			labels[k] = v
		}
		labels[api.LatestPackageRevisionKey] = api.LatestPackageRevisionValue
		repoPkgRev.Labels = labels
	}
	repoPkgRev.Annotations = p.packageRevisionMeta.Annotations
	repoPkgRev.Finalizers = p.packageRevisionMeta.Finalizers
	repoPkgRev.OwnerReferences = p.packageRevisionMeta.OwnerReferences
	repoPkgRev.DeletionTimestamp = p.packageRevisionMeta.DeletionTimestamp

	return repoPkgRev, nil
}

func (p *PackageRevision) KubeObjectName() string {
	return p.repoPackageRevision.KubeObjectName()
}

func (p *PackageRevision) GetResources(ctx context.Context) (*api.PackageRevisionResources, error) {
	return p.repoPackageRevision.GetResources(ctx)
}

type Function struct {
	RepoFunction repository.Function
}

func (f *Function) Name() string {
	return f.RepoFunction.Name()
}

func (f *Function) GetFunction() (*api.Function, error) {
	return f.RepoFunction.GetFunction()
}

func NewCaDEngine(opts ...EngineOption) (CaDEngine, error) {
	engine := &cadEngine{}
	for _, opt := range opts {
		if err := opt.apply(engine); err != nil {
			return nil, err
		}
	}
	return engine, nil
}

type cadEngine struct {
	cache *cache.Cache

	// runnerOptionsResolver returns the RunnerOptions for function execution in the specified namespace.
	runnerOptionsResolver func(namespace string) fnruntime.RunnerOptions

	runtime            fn.FunctionRuntime
	credentialResolver repository.CredentialResolver
	referenceResolver  ReferenceResolver
	userInfoProvider   repository.UserInfoProvider
	metadataStore      meta.MetadataStore
	watcherManager     *watcherManager
}

var _ CaDEngine = &cadEngine{}

type mutation interface {
	Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error)
}

// ObjectCache is a cache of all our objects.
func (cad *cadEngine) ObjectCache() WatcherManager {
	return cad.watcherManager
}

func (cad *cadEngine) OpenRepository(ctx context.Context, repositorySpec *configapi.Repository) (repository.Repository, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::OpenRepository", trace.WithAttributes())
	defer span.End()

	return cad.cache.OpenRepository(ctx, repositorySpec)
}

func (cad *cadEngine) ListPackageRevisions(ctx context.Context, repositorySpec *configapi.Repository, filter repository.ListPackageRevisionFilter) ([]*PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::ListPackageRevisions", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositorySpec)
	if err != nil {
		return nil, err
	}
	pkgRevs, err := repo.ListPackageRevisions(ctx, filter)
	if err != nil {
		return nil, err
	}

	var packageRevisions []*PackageRevision
	for _, pr := range pkgRevs {
		pkgRevMeta, err := cad.metadataStore.Get(ctx, types.NamespacedName{
			Name:      pr.KubeObjectName(),
			Namespace: pr.KubeObjectNamespace(),
		})
		if err != nil {
			// If a PackageRev CR doesn't exist, we treat the
			// Packagerevision as not existing.
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		packageRevisions = append(packageRevisions, &PackageRevision{
			repoPackageRevision: pr,
			packageRevisionMeta: pkgRevMeta,
		})
	}
	return packageRevisions, nil
}

func buildPackageConfig(ctx context.Context, obj *api.PackageRevision, parent *PackageRevision) (*builtins.PackageConfig, error) {
	config := &builtins.PackageConfig{}

	parentPath := ""

	var parentConfig *unstructured.Unstructured
	if parent != nil {
		parentObj, err := parent.GetPackageRevision(ctx)
		if err != nil {
			return nil, err
		}
		parentPath = parentObj.Spec.PackageName

		resources, err := parent.GetResources(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting resources from parent package %q: %w", parentObj.Name, err)
		}
		configMapObj, err := ExtractContextConfigMap(resources.Spec.Resources)
		if err != nil {
			return nil, fmt.Errorf("error getting configuration from parent package %q: %w", parentObj.Name, err)
		}
		parentConfig = configMapObj

		if parentConfig != nil {
			// TODO: Should we support kinds other than configmaps?
			var parentConfigMap corev1.ConfigMap
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(parentConfig.Object, &parentConfigMap); err != nil {
				return nil, fmt.Errorf("error parsing ConfigMap from parent configuration: %w", err)
			}
			if s := parentConfigMap.Data[builtins.ConfigKeyPackagePath]; s != "" {
				parentPath = s + "/" + parentPath
			}
		}
	}

	if parentPath == "" {
		config.PackagePath = obj.Spec.PackageName
	} else {
		config.PackagePath = parentPath + "/" + obj.Spec.PackageName
	}

	return config, nil
}

func (cad *cadEngine) CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj *api.PackageRevision, parent *PackageRevision) (*PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::CreatePackageRevision", trace.WithAttributes())
	defer span.End()

	packageConfig, err := buildPackageConfig(ctx, obj, parent)
	if err != nil {
		return nil, err
	}

	// Validate package lifecycle. Cannot create a final package
	switch obj.Spec.Lifecycle {
	case "":
		// Set draft as default
		obj.Spec.Lifecycle = api.PackageRevisionLifecycleDraft
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed:
		// These values are ok
	case api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// TODO: generate errors that can be translated to correct HTTP responses
		return nil, fmt.Errorf("cannot create a package revision with lifecycle value 'Final'")
	default:
		return nil, fmt.Errorf("unsupported lifecycle value: %s", obj.Spec.Lifecycle)
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}

	if err := repository.ValidateWorkspaceName(obj.Spec.WorkspaceName); err != nil {
		return nil, fmt.Errorf("failed to create packagerevision: %w", err)
	}

	revs, err := repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{
		Package: obj.Spec.PackageName})
	if err != nil {
		return nil, fmt.Errorf("error listing package revisions: %w", err)
	}

	if err := ensureUniqueWorkspaceName(repositoryObj, obj, revs); err != nil {
		return nil, err
	}

	draft, err := repo.CreatePackageRevision(ctx, obj)
	if err != nil {
		return nil, err
	}

	if err := cad.applyTasks(ctx, draft, repositoryObj, obj, packageConfig); err != nil {
		return nil, err
	}

	if err := draft.UpdateLifecycle(ctx, obj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	// Updates are done.
	repoPkgRev, err := draft.Close(ctx)
	if err != nil {
		return nil, err
	}
	pkgRevMeta := meta.PackageRevisionMeta{
		Name:            repoPkgRev.KubeObjectName(),
		Namespace:       repoPkgRev.KubeObjectNamespace(),
		Labels:          obj.Labels,
		Annotations:     obj.Annotations,
		Finalizers:      obj.Finalizers,
		OwnerReferences: obj.OwnerReferences,
	}
	pkgRevMeta, err = cad.metadataStore.Create(ctx, pkgRevMeta, repositoryObj.Name, repoPkgRev.UID())
	if err != nil {
		return nil, err
	}
	sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Added, repoPkgRev, pkgRevMeta)
	klog.Infof("engine: sent %d for new PackageRevision %s/%s", sent, repoPkgRev.KubeObjectNamespace(), repoPkgRev.KubeObjectName())
	return &PackageRevision{
		repoPackageRevision: repoPkgRev,
		packageRevisionMeta: pkgRevMeta,
	}, nil
}

// The workspaceName must be unique, because it used to generate the package revision's metadata.name.
func ensureUniqueWorkspaceName(repositoryObj *configapi.Repository, obj *api.PackageRevision, existingRevs []repository.PackageRevision) error {
	// HACK
	// It's ok for the "main" revision to have the same workspace name
	// So ignore main revisions in this calculation
	mainRev := ""
	if repositoryObj.Spec.Git != nil {
		mainRev = repositoryObj.Spec.Git.Branch
	}

	for _, r := range existingRevs {
		k := r.Key()
		if mainRev != "" && k.Revision == mainRev {
			continue
		}
		if k.WorkspaceName == obj.Spec.WorkspaceName {
			return fmt.Errorf("package revision workspaceNames must be unique; package revision with name %s in repo %s with "+
				"workspaceName %s already exists", obj.Spec.PackageName, obj.Spec.RepositoryName, obj.Spec.WorkspaceName)
		}
	}
	return nil
}

func getPackageRevision(ctx context.Context, repo repository.Repository, name string) (repository.PackageRevision, bool, error) {
	repoPkgRevs, err := repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{
		KubeObjectName: name,
	})
	if err != nil {
		return nil, false, err
	}
	if len(repoPkgRevs) == 0 {
		return nil, false, nil
	}
	return repoPkgRevs[0], true, nil
}

// TODO: See if we can use a library here for parsing OCI image urls
func getBaseImage(image string) string {
	if s := strings.Split(image, "@sha256:"); len(s) > 1 {
		return s[0]
	}
	if s := strings.Split(image, ":"); len(s) > 1 {
		return s[0]
	}
	return image
}

func taskTypeOneOf(taskType api.TaskType, oneOf ...api.TaskType) bool {
	for _, tt := range oneOf {
		if taskType == tt {
			return true
		}
	}
	return false
}

func (cad *cadEngine) applyTasks(ctx context.Context, draft repository.PackageDraft, repositoryObj *configapi.Repository, obj *api.PackageRevision, packageConfig *builtins.PackageConfig) error {
	var mutations []mutation

	// Unless first task is Init or Clone, insert Init to create an empty package.
	tasks := obj.Spec.Tasks
	if len(tasks) == 0 || !taskTypeOneOf(tasks[0].Type, api.TaskTypeInit, api.TaskTypeClone, api.TaskTypeEdit) {
		mutations = append(mutations, &initPackageMutation{
			name: obj.Spec.PackageName,
			task: &api.Task{
				Init: &api.PackageInitTaskSpec{
					Subpackage:  "",
					Description: fmt.Sprintf("%s description", obj.Spec.PackageName),
				},
			},
		})
	}

	for i := range tasks {
		task := &tasks[i]
		mutation, err := cad.mapTaskToMutation(ctx, obj, task, repositoryObj.Spec.Deployment, packageConfig)
		if err != nil {
			return err
		}
		mutations = append(mutations, mutation)
	}

	// Render package after creation.
	mutations = cad.conditionalAddRender(obj, mutations)

	baseResources := repository.PackageResources{}
	if _, _, err := applyResourceMutations(ctx, draft, baseResources, mutations); err != nil {
		return err
	}

	return nil
}

type RepositoryOpener interface {
	OpenRepository(ctx context.Context, repositorySpec *configapi.Repository) (repository.Repository, error)
}

func (cad *cadEngine) mapTaskToMutation(ctx context.Context, obj *api.PackageRevision, task *api.Task, isDeployment bool, packageConfig *builtins.PackageConfig) (mutation, error) {
	switch task.Type {
	case api.TaskTypeInit:
		if task.Init == nil {
			return nil, fmt.Errorf("init not set for task of type %q", task.Type)
		}
		return &initPackageMutation{
			name: obj.Spec.PackageName,
			task: task,
		}, nil
	case api.TaskTypeClone:
		if task.Clone == nil {
			return nil, fmt.Errorf("clone not set for task of type %q", task.Type)
		}
		return &clonePackageMutation{
			task:               task,
			namespace:          obj.Namespace,
			name:               obj.Spec.PackageName,
			isDeployment:       isDeployment,
			repoOpener:         cad,
			credentialResolver: cad.credentialResolver,
			referenceResolver:  cad.referenceResolver,
			packageConfig:      packageConfig,
		}, nil

	case api.TaskTypeUpdate:
		if task.Update == nil {
			return nil, fmt.Errorf("update not set for task of type %q", task.Type)
		}
		cloneTask := findCloneTask(obj)
		if cloneTask == nil {
			return nil, fmt.Errorf("upstream source not found for package rev %q; only cloned packages can be updated", obj.Spec.PackageName)
		}
		return &updatePackageMutation{
			cloneTask:         cloneTask,
			updateTask:        task,
			namespace:         obj.Namespace,
			repoOpener:        cad,
			referenceResolver: cad.referenceResolver,
			pkgName:           obj.Spec.PackageName,
		}, nil

	case api.TaskTypePatch:
		return buildPatchMutation(ctx, task)

	case api.TaskTypeEdit:
		if task.Edit == nil {
			return nil, fmt.Errorf("edit not set for task of type %q", task.Type)
		}
		return &editPackageMutation{
			task:              task,
			namespace:         obj.Namespace,
			packageName:       obj.Spec.PackageName,
			repositoryName:    obj.Spec.RepositoryName,
			repoOpener:        cad,
			referenceResolver: cad.referenceResolver,
		}, nil

	case api.TaskTypeEval:
		if task.Eval == nil {
			return nil, fmt.Errorf("eval not set for task of type %q", task.Type)
		}
		// TODO: We should find a different way to do this. Probably a separate
		// task for render.
		if task.Eval.Image == "render" {
			runnerOptions := cad.runnerOptionsResolver(obj.Namespace)
			return &renderPackageMutation{
				runnerOptions: runnerOptions,
				runtime:       cad.runtime,
			}, nil
		} else {
			runnerOptions := cad.runnerOptionsResolver(obj.Namespace)
			return &evalFunctionMutation{
				runnerOptions: runnerOptions,
				runtime:       cad.runtime,
				task:          task,
			}, nil
		}

	default:
		return nil, fmt.Errorf("task of type %q not supported", task.Type)
	}
}

func (cad *cadEngine) UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *PackageRevision, oldObj, newObj *api.PackageRevision, parent *PackageRevision) (*PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::UpdatePackageRevision", trace.WithAttributes())
	defer span.End()

	newRV := newObj.GetResourceVersion()
	if len(newRV) == 0 {
		return nil, fmt.Errorf("resourceVersion must be specified for an update")
	}

	if newRV != oldObj.GetResourceVersion() {
		return nil, apierrors.NewConflict(api.Resource("packagerevisions"), oldObj.GetName(), fmt.Errorf(OptimisticLockErrorMsg))
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}

	// Check if the PackageRevision is in the terminating state and
	// and this request removes the last finalizer.
	repoPkgRev := oldPackage.repoPackageRevision
	pkgRevMetaNN := types.NamespacedName{
		Name:      repoPkgRev.KubeObjectName(),
		Namespace: repoPkgRev.KubeObjectNamespace(),
	}
	pkgRevMeta, err := cad.metadataStore.Get(ctx, pkgRevMetaNN)
	if err != nil {
		return nil, err
	}
	// If this is in the terminating state and we are removing the last finalizer,
	// we delete the resource instead of updating it.
	if pkgRevMeta.DeletionTimestamp != nil && len(newObj.Finalizers) == 0 {
		if err := cad.deletePackageRevision(ctx, repo, repoPkgRev, pkgRevMeta); err != nil {
			return nil, err
		}
		return ToPackageRevision(repoPkgRev, pkgRevMeta), nil
	}

	// Validate package lifecycle. Can only update a draft.
	switch lifecycle := oldObj.Spec.Lifecycle; lifecycle {
	default:
		return nil, fmt.Errorf("invalid original lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed:
		// Draft or proposed can be updated.
	case api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// Only metadata (currently labels and annotations) and lifecycle can be updated for published packages.
		if oldObj.Spec.Lifecycle != newObj.Spec.Lifecycle {
			if err := oldPackage.repoPackageRevision.UpdateLifecycle(ctx, newObj.Spec.Lifecycle); err != nil {
				return nil, err
			}
		}

		pkgRevMeta, err = cad.updatePkgRevMeta(ctx, repoPkgRev, newObj)
		if err != nil {
			return nil, err
		}

		sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Modified, repoPkgRev, pkgRevMeta)
		klog.Infof("engine: sent %d for updated PackageRevision metadata %s/%s", sent, repoPkgRev.KubeObjectNamespace(), repoPkgRev.KubeObjectName())
		return ToPackageRevision(repoPkgRev, pkgRevMeta), nil
	}
	switch lifecycle := newObj.Spec.Lifecycle; lifecycle {
	default:
		return nil, fmt.Errorf("invalid desired lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// These values are ok
	}

	if isRecloneAndReplay(oldObj, newObj) {
		packageConfig, err := buildPackageConfig(ctx, newObj, parent)
		if err != nil {
			return nil, err
		}
		repoPkgRev, err := cad.recloneAndReplay(ctx, repo, repositoryObj, newObj, packageConfig)
		if err != nil {
			return nil, err
		}

		pkgRevMeta, err = cad.updatePkgRevMeta(ctx, repoPkgRev, newObj)
		if err != nil {
			return nil, err
		}

		sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Modified, repoPkgRev, pkgRevMeta)
		klog.Infof("engine: sent %d for reclone and replay PackageRevision %s/%s", sent, repoPkgRev.KubeObjectNamespace(), repoPkgRev.KubeObjectName())
		return ToPackageRevision(repoPkgRev, pkgRevMeta), nil
	}

	var mutations []mutation
	if len(oldObj.Spec.Tasks) > len(newObj.Spec.Tasks) {
		return nil, fmt.Errorf("removing tasks is not yet supported")
	}
	for i := range oldObj.Spec.Tasks {
		oldTask := &oldObj.Spec.Tasks[i]
		newTask := &newObj.Spec.Tasks[i]
		if oldTask.Type != newTask.Type {
			return nil, fmt.Errorf("changing task types is not yet supported")
		}
	}
	if len(newObj.Spec.Tasks) > len(oldObj.Spec.Tasks) {
		if len(newObj.Spec.Tasks) > len(oldObj.Spec.Tasks)+1 {
			return nil, fmt.Errorf("can only append one task at a time")
		}

		newTask := newObj.Spec.Tasks[len(newObj.Spec.Tasks)-1]
		if newTask.Type != api.TaskTypeUpdate {
			return nil, fmt.Errorf("appended task is type %q, must be type %q", newTask.Type, api.TaskTypeUpdate)
		}
		if newTask.Update == nil {
			return nil, fmt.Errorf("update not set for updateTask of type %q", newTask.Type)
		}

		cloneTask := findCloneTask(oldObj)
		if cloneTask == nil {
			return nil, fmt.Errorf("upstream source not found for package rev %q; only cloned packages can be updated", oldObj.Spec.PackageName)
		}

		mutation := &updatePackageMutation{
			cloneTask:         cloneTask,
			updateTask:        &newTask,
			repoOpener:        cad,
			referenceResolver: cad.referenceResolver,
			namespace:         repositoryObj.Namespace,
			pkgName:           oldObj.GetName(),
		}
		mutations = append(mutations, mutation)
	}

	// Re-render if we are making changes.
	mutations = cad.conditionalAddRender(newObj, mutations)

	draft, err := repo.UpdatePackageRevision(ctx, oldPackage.repoPackageRevision)
	if err != nil {
		return nil, err
	}

	// If any of the fields in the API that are projections from the Kptfile
	// must be updated in the Kptfile as well.
	kfPatchTask, created, err := createKptfilePatchTask(ctx, oldPackage.repoPackageRevision, newObj)
	if err != nil {
		return nil, err
	}
	if created {
		kfPatchMutation, err := buildPatchMutation(ctx, kfPatchTask)
		if err != nil {
			return nil, err
		}
		mutations = append(mutations, kfPatchMutation)
	}

	// Re-render if we are making changes.
	mutations = cad.conditionalAddRender(newObj, mutations)

	// TODO: Handle the case if alongside lifecycle change, tasks are changed too.
	// Update package contents only if the package is in draft state
	if oldObj.Spec.Lifecycle == api.PackageRevisionLifecycleDraft {
		apiResources, err := oldPackage.GetResources(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot get package resources: %w", err)
		}
		resources := repository.PackageResources{
			Contents: apiResources.Spec.Resources,
		}

		if _, _, err := applyResourceMutations(ctx, draft, resources, mutations); err != nil {
			return nil, err
		}
	}

	if err := draft.UpdateLifecycle(ctx, newObj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	// Updates are done.
	repoPkgRev, err = draft.Close(ctx)
	if err != nil {
		return nil, err
	}

	pkgRevMeta, err = cad.updatePkgRevMeta(ctx, repoPkgRev, newObj)
	if err != nil {
		return nil, err
	}

	sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Modified, repoPkgRev, pkgRevMeta)
	klog.Infof("engine: sent %d for updated PackageRevision %s/%s", sent, repoPkgRev.KubeObjectNamespace(), repoPkgRev.KubeObjectName())
	return ToPackageRevision(repoPkgRev, pkgRevMeta), nil
}

func (cad *cadEngine) updatePkgRevMeta(ctx context.Context, repoPkgRev repository.PackageRevision, apiPkgRev *api.PackageRevision) (meta.PackageRevisionMeta, error) {
	pkgRevMeta := meta.PackageRevisionMeta{
		Name:            repoPkgRev.KubeObjectName(),
		Namespace:       repoPkgRev.KubeObjectNamespace(),
		Labels:          apiPkgRev.Labels,
		Annotations:     apiPkgRev.Annotations,
		Finalizers:      apiPkgRev.Finalizers,
		OwnerReferences: apiPkgRev.OwnerReferences,
	}
	return cad.metadataStore.Update(ctx, pkgRevMeta)
}

func createKptfilePatchTask(ctx context.Context, oldPackage repository.PackageRevision, newObj *api.PackageRevision) (*api.Task, bool, error) {
	kf, err := oldPackage.GetKptfile(ctx)
	if err != nil {
		return nil, false, err
	}

	var orgKfString string
	{
		var buf bytes.Buffer
		d := yaml.NewEncoder(&buf)
		if err := d.Encode(kf); err != nil {
			return nil, false, err
		}
		orgKfString = buf.String()
	}

	var readinessGates []kptfile.ReadinessGate
	for _, rg := range newObj.Spec.ReadinessGates {
		readinessGates = append(readinessGates, kptfile.ReadinessGate{
			ConditionType: rg.ConditionType,
		})
	}

	var conditions []kptfile.Condition
	for _, c := range newObj.Status.Conditions {
		conditions = append(conditions, kptfile.Condition{
			Type:    c.Type,
			Status:  convertStatusToKptfile(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
	}

	if kf.Info == nil && len(readinessGates) > 0 {
		kf.Info = &kptfile.PackageInfo{}
	}
	if len(readinessGates) > 0 {
		kf.Info.ReadinessGates = readinessGates
	}

	if kf.Status == nil && len(conditions) > 0 {
		kf.Status = &kptfile.Status{}
	}
	if len(conditions) > 0 {
		kf.Status.Conditions = conditions
	}

	var newKfString string
	{
		var buf bytes.Buffer
		d := yaml.NewEncoder(&buf)
		if err := d.Encode(kf); err != nil {
			return nil, false, err
		}
		newKfString = buf.String()
	}
	patchSpec, err := GeneratePatch(kptfile.KptFileName, orgKfString, newKfString)
	if err != nil {
		return nil, false, err
	}
	// If patch is empty, don't create a Task.
	if patchSpec.Contents == "" {
		return nil, false, nil
	}

	return &api.Task{
		Type: api.TaskTypePatch,
		Patch: &api.PackagePatchTaskSpec{
			Patches: []api.PatchSpec{
				patchSpec,
			},
		},
	}, true, nil
}

func convertStatusToKptfile(s api.ConditionStatus) kptfile.ConditionStatus {
	switch s {
	case api.ConditionTrue:
		return kptfile.ConditionTrue
	case api.ConditionFalse:
		return kptfile.ConditionFalse
	case api.ConditionUnknown:
		return kptfile.ConditionUnknown
	default:
		panic(fmt.Errorf("unknown condition status: %v", s))
	}
}

// conditionalAddRender adds a render mutation to the end of the mutations slice if the last
// entry is not already a render mutation.
func (cad *cadEngine) conditionalAddRender(subject client.Object, mutations []mutation) []mutation {
	if len(mutations) == 0 || isRenderMutation(mutations[len(mutations)-1]) {
		return mutations
	}

	runnerOptions := cad.runnerOptionsResolver(subject.GetNamespace())

	return append(mutations, &renderPackageMutation{
		runnerOptions: runnerOptions,
		runtime:       cad.runtime,
	})
}

func isRenderMutation(m mutation) bool {
	_, isRender := m.(*renderPackageMutation)
	return isRender
}

func (cad *cadEngine) DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *PackageRevision) error {
	ctx, span := tracer.Start(ctx, "cadEngine::DeletePackageRevision", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return err
	}

	// We delete the PackageRev regardless of any finalizers, since it
	// will always have the same finalizers as the PackageRevision. This
	// will put the PackageRev, and therefore the PackageRevision in the
	// terminating state.
	// But we only delete the PackageRevision from the repo once all finalizers
	// have been removed.
	namespacedName := types.NamespacedName{
		Name:      oldPackage.repoPackageRevision.KubeObjectName(),
		Namespace: oldPackage.repoPackageRevision.KubeObjectNamespace(),
	}
	pkgRevMeta, err := cad.metadataStore.Delete(ctx, namespacedName, false)
	if err != nil {
		return err
	}

	if len(pkgRevMeta.Finalizers) > 0 {
		klog.Infof("PackageRevision %s deleted, but still have finalizers: %s", oldPackage.KubeObjectName(), strings.Join(pkgRevMeta.Finalizers, ","))
		sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Modified, oldPackage.repoPackageRevision, oldPackage.packageRevisionMeta)
		klog.Infof("engine: sent %d modified for deleted PackageRevision %s/%s with finalizers", sent, oldPackage.repoPackageRevision.KubeObjectNamespace(), oldPackage.KubeObjectName())
		return nil
	}
	klog.Infof("PackageRevision %s deleted for real since no finalizers", oldPackage.KubeObjectName())

	return cad.deletePackageRevision(ctx, repo, oldPackage.repoPackageRevision, oldPackage.packageRevisionMeta)
}

func (cad *cadEngine) deletePackageRevision(ctx context.Context, repo repository.Repository, repoPkgRev repository.PackageRevision, pkgRevMeta meta.PackageRevisionMeta) error {
	ctx, span := tracer.Start(ctx, "cadEngine::deletePackageRevision", trace.WithAttributes())
	defer span.End()

	if err := repo.DeletePackageRevision(ctx, repoPkgRev); err != nil {
		return err
	}

	nn := types.NamespacedName{
		Name:      pkgRevMeta.Name,
		Namespace: pkgRevMeta.Namespace,
	}
	if _, err := cad.metadataStore.Delete(ctx, nn, true); err != nil {
		// If this fails, the CR will be cleaned up by the background job.
		if !apierrors.IsNotFound(err) {
			klog.Warningf("Error deleting PkgRevMeta %s: %v", nn.String(), err)
		}
	}

	sent := cad.watcherManager.NotifyPackageRevisionChange(watch.Deleted, repoPkgRev, pkgRevMeta)
	klog.Infof("engine: sent %d for deleted PackageRevision %s/%s", sent, repoPkgRev.KubeObjectNamespace(), repoPkgRev.KubeObjectName())
	return nil
}

func (cad *cadEngine) ListPackages(ctx context.Context, repositorySpec *configapi.Repository, filter repository.ListPackageFilter) ([]*Package, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::ListPackages", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositorySpec)
	if err != nil {
		return nil, err
	}

	pkgs, err := repo.ListPackages(ctx, filter)
	if err != nil {
		return nil, err
	}
	var packages []*Package
	for _, p := range pkgs {
		packages = append(packages, &Package{
			repoPackage: p,
		})
	}

	return packages, nil
}

func (cad *cadEngine) CreatePackage(ctx context.Context, repositoryObj *configapi.Repository, obj *api.Package) (*Package, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::CreatePackage", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}
	pkg, err := repo.CreatePackage(ctx, obj)
	if err != nil {
		return nil, err
	}

	return &Package{
		repoPackage: pkg,
	}, nil
}

func (cad *cadEngine) UpdatePackage(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *Package, oldObj, newObj *api.Package) (*Package, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::UpdatePackage", trace.WithAttributes())
	defer span.End()

	// TODO
	var pkg *Package
	return pkg, fmt.Errorf("Updating packages is not yet supported")
}

func (cad *cadEngine) DeletePackage(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *Package) error {
	ctx, span := tracer.Start(ctx, "cadEngine::DeletePackage", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return err
	}

	if err := repo.DeletePackage(ctx, oldPackage.repoPackage); err != nil {
		return err
	}

	return nil
}

func (cad *cadEngine) UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, oldPackage *PackageRevision, old, new *api.PackageRevisionResources) (*PackageRevision, *api.RenderStatus, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::UpdatePackageResources", trace.WithAttributes())
	defer span.End()

	rev, err := oldPackage.repoPackageRevision.GetPackageRevision(ctx)
	if err != nil {
		return nil, nil, err
	}

	newRV := new.GetResourceVersion()
	if len(newRV) == 0 {
		return nil, nil, fmt.Errorf("resourceVersion must be specified for an update")
	}

	if newRV != old.GetResourceVersion() {
		return nil, nil, apierrors.NewConflict(api.Resource("packagerevisionresources"), old.GetName(), fmt.Errorf(OptimisticLockErrorMsg))
	}

	// Validate package lifecycle. Can only update a draft.
	switch lifecycle := rev.Spec.Lifecycle; lifecycle {
	default:
		return nil, nil, fmt.Errorf("invalid original lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft:
		// Only drafts can be updated.
	case api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// TODO: generate errors that can be translated to correct HTTP responses
		return nil, nil, fmt.Errorf("cannot update a package revision with lifecycle value %q; package must be Draft", lifecycle)
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, nil, err
	}
	draft, err := repo.UpdatePackageRevision(ctx, oldPackage.repoPackageRevision)
	if err != nil {
		return nil, nil, err
	}

	runnerOptions := cad.runnerOptionsResolver(old.GetNamespace())

	mutations := []mutation{
		&mutationReplaceResources{
			newResources: new,
			oldResources: old,
		},
	}
	prevResources, err := oldPackage.repoPackageRevision.GetResources(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get package resources: %w", err)
	}
	resources := repository.PackageResources{
		Contents: prevResources.Spec.Resources,
	}
	appliedResources, _, err := applyResourceMutations(ctx, draft, resources, mutations)
	if err != nil {
		return nil, nil, err
	}

	// render the package
	// Render failure will not fail the overall API operation.
	// The render error and result is captured as part of renderStatus above
	// and is returned in packageresourceresources API's status field. We continue with
	// saving the non-rendered resources to avoid losing user's changes.
	// and supress this err.
	_, renderStatus, _ := applyResourceMutations(ctx,
		draft,
		appliedResources,
		[]mutation{&renderPackageMutation{
			runnerOptions: runnerOptions,
			runtime:       cad.runtime,
		}})

	// No lifecycle change when updating package resources; updates are done.
	repoPkgRev, err := draft.Close(ctx)
	if err != nil {
		return nil, renderStatus, err
	}
	return &PackageRevision{
		repoPackageRevision: repoPkgRev,
	}, renderStatus, nil
}

// applyResourceMutations mutates the resources and returns the most recent renderResult.
func applyResourceMutations(ctx context.Context, draft repository.PackageDraft, baseResources repository.PackageResources, mutations []mutation) (applied repository.PackageResources, renderStatus *api.RenderStatus, err error) {
	var lastApplied mutation
	for _, m := range mutations {
		updatedResources, taskResult, err := m.Apply(ctx, baseResources)
		if taskResult == nil && err == nil {
			// a nil taskResult means nothing changed
			continue
		}

		var task *api.Task
		if taskResult != nil {
			task = taskResult.Task
		}
		if taskResult != nil && task.Type == api.TaskTypeEval {
			renderStatus = taskResult.RenderStatus
		}
		if err != nil {
			return updatedResources, renderStatus, err
		}

		// if the last applied mutation was a render mutation, and so is this one, skip it
		if lastApplied != nil && isRenderMutation(m) && isRenderMutation(lastApplied) {
			continue
		}
		lastApplied = m

		if err := draft.UpdateResources(ctx, &api.PackageRevisionResources{
			Spec: api.PackageRevisionResourcesSpec{
				Resources: updatedResources.Contents,
			},
		}, task); err != nil {
			return updatedResources, renderStatus, err
		}
		baseResources = updatedResources
		applied = updatedResources
	}

	return applied, renderStatus, nil
}

func (cad *cadEngine) ListFunctions(ctx context.Context, repositoryObj *configapi.Repository) ([]*Function, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::ListFunctions", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}

	fns, err := repo.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}

	var functions []*Function
	for _, f := range fns {
		functions = append(functions, &Function{
			RepoFunction: f,
		})
	}

	return functions, nil
}

type updatePackageMutation struct {
	cloneTask         *api.Task
	updateTask        *api.Task
	repoOpener        RepositoryOpener
	referenceResolver ReferenceResolver
	namespace         string
	pkgName           string
}

func (m *updatePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error) {
	ctx, span := tracer.Start(ctx, "updatePackageMutation::Apply", trace.WithAttributes())
	defer span.End()

	currUpstreamPkgRef, err := m.currUpstream()
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	targetUpstream := m.updateTask.Update.Upstream
	if targetUpstream.Type == api.RepositoryTypeGit || targetUpstream.Type == api.RepositoryTypeOCI {
		return repository.PackageResources{}, nil, fmt.Errorf("update is not supported for non-porch upstream packages")
	}

	originalResources, err := (&PackageFetcher{
		repoOpener:        m.repoOpener,
		referenceResolver: m.referenceResolver,
	}).FetchResources(ctx, currUpstreamPkgRef, m.namespace)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error fetching the resources for package %s with ref %+v",
			m.pkgName, *currUpstreamPkgRef)
	}

	upstreamRevision, err := (&PackageFetcher{
		repoOpener:        m.repoOpener,
		referenceResolver: m.referenceResolver,
	}).FetchRevision(ctx, targetUpstream.UpstreamRef, m.namespace)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error fetching revision for target upstream %s", targetUpstream.UpstreamRef.Name)
	}
	upstreamResources, err := upstreamRevision.GetResources(ctx)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error fetching resources for target upstream %s", targetUpstream.UpstreamRef.Name)
	}

	klog.Infof("performing pkg upgrade operation for pkg %s resource counts local[%d] original[%d] upstream[%d]",
		m.pkgName, len(resources.Contents), len(originalResources.Spec.Resources), len(upstreamResources.Spec.Resources))

	// May be have packageUpdater part of engine to make it easy for testing ?
	updatedResources, err := (&defaultPackageUpdater{}).Update(ctx,
		resources,
		repository.PackageResources{
			Contents: originalResources.Spec.Resources,
		},
		repository.PackageResources{
			Contents: upstreamResources.Spec.Resources,
		})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error updating the package to revision %s", targetUpstream.UpstreamRef.Name)
	}

	newUpstream, newUpstreamLock, err := upstreamRevision.GetLock()
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error fetching the resources for package revisions %s", targetUpstream.UpstreamRef.Name)
	}
	if err := kpt.UpdateKptfileUpstream("", updatedResources.Contents, newUpstream, newUpstreamLock); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to apply upstream lock to package %q: %w", m.pkgName, err)
	}

	// ensure merge-key comment is added to newly added resources.
	result, err := ensureMergeKey(ctx, updatedResources)
	if err != nil {
		klog.Infof("failed to add merge key comments: %v", err)
	}
	return result, &api.TaskResult{Task: m.updateTask}, nil
}

// Currently assumption is that downstream packages will be forked from a porch package.
// As per current implementation, upstream package ref is stored in a new update task but this may
// change so the logic of figuring out current upstream will live in this function.
func (m *updatePackageMutation) currUpstream() (*api.PackageRevisionRef, error) {
	if m.cloneTask == nil || m.cloneTask.Clone == nil {
		return nil, fmt.Errorf("package %s does not have original upstream info", m.pkgName)
	}
	upstream := m.cloneTask.Clone.Upstream
	if upstream.Type == api.RepositoryTypeGit || upstream.Type == api.RepositoryTypeOCI {
		return nil, fmt.Errorf("upstream package must be porch native package. Found it to be %s", upstream.Type)
	}
	return upstream.UpstreamRef, nil
}

func findCloneTask(pr *api.PackageRevision) *api.Task {
	if len(pr.Spec.Tasks) == 0 {
		return nil
	}
	firstTask := pr.Spec.Tasks[0]
	if firstTask.Type == api.TaskTypeClone {
		return &firstTask
	}
	return nil
}

func writeResourcesToDirectory(dir string, resources repository.PackageResources) error {
	for k, v := range resources.Contents {
		p := filepath.Join(dir, k)
		dir := filepath.Dir(p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
		if err := os.WriteFile(p, []byte(v), 0644); err != nil {
			return fmt.Errorf("failed to write file %q: %w", dir, err)
		}
	}
	return nil
}

func loadResourcesFromDirectory(dir string) (repository.PackageResources, error) {
	// TODO: return abstraction instead of loading everything
	result := repository.PackageResources{
		Contents: map[string]string{},
	}
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("cannot compute relative path %q, %q, %w", dir, path, err)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", dir, err)
		}
		result.Contents[rel] = string(contents)
		return nil
	}); err != nil {
		return repository.PackageResources{}, err
	}

	return result, nil
}

type mutationReplaceResources struct {
	newResources *api.PackageRevisionResources
	oldResources *api.PackageRevisionResources
}

func (m *mutationReplaceResources) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error) {
	ctx, span := tracer.Start(ctx, "mutationReplaceResources::Apply", trace.WithAttributes())
	defer span.End()

	patch := &api.PackagePatchTaskSpec{}

	old := resources.Contents
	new, err := healConfig(old, m.newResources.Spec.Resources)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to heal resources: %w", err)
	}

	for k, newV := range new {
		oldV, ok := old[k]
		// New config or changed config
		if !ok {
			patchSpec := api.PatchSpec{
				File:      k,
				PatchType: api.PatchTypeCreateFile,
				Contents:  newV,
			}
			patch.Patches = append(patch.Patches, patchSpec)
		} else if newV != oldV {
			patchSpec, err := GeneratePatch(k, oldV, newV)
			if err != nil {
				return repository.PackageResources{}, nil, fmt.Errorf("error generating patch: %w", err)
			}
			if patchSpec.Contents == "" {
				continue
			}
			patch.Patches = append(patch.Patches, patchSpec)
		}
	}
	for k := range old {
		// Deleted config
		if _, ok := new[k]; !ok {
			patchSpec := api.PatchSpec{
				File:      k,
				PatchType: api.PatchTypeDeleteFile,
			}
			patch.Patches = append(patch.Patches, patchSpec)
		}
	}
	// If patch is empty, don't create a Task.
	var taskResult *api.TaskResult
	if len(patch.Patches) > 0 {
		taskResult = &api.TaskResult{
			Task: &api.Task{
				Type:  api.TaskTypePatch,
				Patch: patch,
			},
		}
	}
	return repository.PackageResources{Contents: new}, taskResult, nil
}

func healConfig(old, new map[string]string) (map[string]string, error) {
	// Copy comments from old config to new
	oldResources, err := (&packageReader{
		input: repository.PackageResources{Contents: old},
		extra: map[string]string{},
	}).Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read old packge resources: %w", err)
	}

	var filter kio.FilterFunc = func(r []*yaml.RNode) ([]*yaml.RNode, error) {
		for _, n := range r {
			for _, original := range oldResources {
				if n.GetNamespace() == original.GetNamespace() &&
					n.GetName() == original.GetName() &&
					n.GetApiVersion() == original.GetApiVersion() &&
					n.GetKind() == original.GetKind() {
					comments.CopyComments(original, n)
				}
			}
		}
		return r, nil
	}

	out := &packageWriter{
		output: repository.PackageResources{
			Contents: map[string]string{},
		},
	}

	extra := map[string]string{}

	if err := (kio.Pipeline{
		Inputs: []kio.Reader{&packageReader{
			input: repository.PackageResources{Contents: new},
			extra: extra,
		}},
		Filters:               []kio.Filter{filter},
		Outputs:               []kio.Writer{out},
		ContinueOnEmptyResult: true,
	}).Execute(); err != nil {
		return nil, err
	}

	healed := out.output.Contents

	for k, v := range extra {
		healed[k] = v
	}

	return healed, nil
}

// isRecloneAndReplay determines if an update should be handled using reclone-and-replay semantics.
// We detect this by checking if both old and new versions start by cloning a package, but the version has changed.
// We may expand this scope in future.
func isRecloneAndReplay(oldObj, newObj *api.PackageRevision) bool {
	oldTasks := oldObj.Spec.Tasks
	newTasks := newObj.Spec.Tasks
	if len(oldTasks) == 0 || len(newTasks) == 0 {
		return false
	}

	if oldTasks[0].Type != api.TaskTypeClone || newTasks[0].Type != api.TaskTypeClone {
		return false
	}

	if reflect.DeepEqual(oldTasks[0], newTasks[0]) {
		return false
	}
	return true
}

// recloneAndReplay performs an update by recloning the upstream package and replaying all tasks.
// This is more like a git rebase operation than the "classic" kpt update algorithm, which is more like a git merge.
func (cad *cadEngine) recloneAndReplay(ctx context.Context, repo repository.Repository, repositoryObj *configapi.Repository, newObj *api.PackageRevision, packageConfig *builtins.PackageConfig) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::recloneAndReplay", trace.WithAttributes())
	defer span.End()

	// For reclone and replay, we create a new package every time
	// the version should be in newObj so we will overwrite.
	draft, err := repo.CreatePackageRevision(ctx, newObj)
	if err != nil {
		return nil, err
	}

	if err := cad.applyTasks(ctx, draft, repositoryObj, newObj, packageConfig); err != nil {
		return nil, err
	}

	if err := draft.UpdateLifecycle(ctx, newObj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	return draft.Close(ctx)
}

// ExtractContextConfigMap returns the package-context configmap, if found
func ExtractContextConfigMap(resources map[string]string) (*unstructured.Unstructured, error) {
	unstructureds, err := objects.Parser{}.AsUnstructureds(resources)
	if err != nil {
		return nil, err
	}

	var matches []*unstructured.Unstructured
	for _, o := range unstructureds {
		configMapGK := schema.GroupKind{Kind: "ConfigMap"}
		if o.GroupVersionKind().GroupKind() == configMapGK {
			if o.GetName() == builtins.PkgContextName {
				matches = append(matches, o)
			}
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("found multiple configmaps matching name %q", builtins.PkgContextFile)
	}

	return matches[0], nil
}
