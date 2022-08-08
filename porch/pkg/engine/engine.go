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

package engine

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/cache"
	"github.com/GoogleContainerTools/kpt/porch/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/comments"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var tracer = otel.Tracer("engine")

type CaDEngine interface {
	OpenRepository(ctx context.Context, repositorySpec *configapi.Repository) (repository.Repository, error)
	CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj *api.PackageRevision) (repository.PackageRevision, error)
	UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage repository.PackageRevision, old, new *api.PackageRevision) (repository.PackageRevision, error)
	UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, oldPackage repository.PackageRevision, old, new *api.PackageRevisionResources) (repository.PackageRevision, error)
	DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj repository.PackageRevision) error
	ListFunctions(ctx context.Context, repositoryObj *configapi.Repository) ([]repository.Function, error)
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
	cache              *cache.Cache
	renderer           fn.Renderer
	runtime            fn.FunctionRuntime
	credentialResolver repository.CredentialResolver
	referenceResolver  ReferenceResolver
	userInfoProvider   repository.UserInfoProvider
}

var _ CaDEngine = &cadEngine{}

type mutation interface {
	Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error)
}

func (cad *cadEngine) OpenRepository(ctx context.Context, repositorySpec *configapi.Repository) (repository.Repository, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::OpenRepository", trace.WithAttributes())
	defer span.End()

	return cad.cache.OpenRepository(ctx, repositorySpec)
}

func (cad *cadEngine) CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, obj *api.PackageRevision) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::CreatePackageRevision", trace.WithAttributes())
	defer span.End()

	// Validate package lifecycle. Cannot create a final package
	switch obj.Spec.Lifecycle {
	case "":
		// Set draft as default
		obj.Spec.Lifecycle = api.PackageRevisionLifecycleDraft
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed:
		// These values are ok
	case api.PackageRevisionLifecyclePublished:
		// TODO: generate errors that can be translated to correct HTTP responses
		return nil, fmt.Errorf("cannot create a package revision with lifecycle value 'Final'")
	default:
		return nil, fmt.Errorf("unsupported lifecycle value: %s", obj.Spec.Lifecycle)
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}
	draft, err := repo.CreatePackageRevision(ctx, obj)
	if err != nil {
		return nil, err
	}

	if err := cad.applyTasks(ctx, draft, repositoryObj, obj); err != nil {
		return nil, err
	}

	if err := draft.UpdateLifecycle(ctx, obj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	// Updates are done.
	return draft.Close(ctx)
}

func (cad *cadEngine) applyTasks(ctx context.Context, draft repository.PackageDraft, repositoryObj *configapi.Repository, obj *api.PackageRevision) error {
	var mutations []mutation

	// Unless first task is Init or Clone, insert Init to create an empty package.
	tasks := obj.Spec.Tasks
	if len(tasks) == 0 || (tasks[0].Type != api.TaskTypeInit && tasks[0].Type != api.TaskTypeClone) {
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
		mutation, err := cad.mapTaskToMutation(ctx, obj, task, repositoryObj.Spec.Deployment)
		if err != nil {
			return err
		}
		mutations = append(mutations, mutation)
	}

	// Render package after creation.
	mutations = cad.conditionalAddRender(mutations)

	baseResources := repository.PackageResources{}
	if err := applyResourceMutations(ctx, draft, baseResources, mutations); err != nil {
		return err
	}

	return nil
}

func (cad *cadEngine) mapTaskToMutation(ctx context.Context, obj *api.PackageRevision, task *api.Task, isDeployment bool) (mutation, error) {
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
			cad:                cad,
			credentialResolver: cad.credentialResolver,
			referenceResolver:  cad.referenceResolver,
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
			cad:               cad,
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
			cad:               cad,
			referenceResolver: cad.referenceResolver,
		}, nil

	case api.TaskTypeEval:
		if task.Eval == nil {
			return nil, fmt.Errorf("eval not set for task of type %q", task.Type)
		}
		// TODO: We should find a different way to do this. Probably a separate
		// task for render.
		if task.Eval.Image == "render" {
			return &renderPackageMutation{
				renderer: cad.renderer,
				runtime:  cad.runtime,
			}, nil
		} else {
			return &evalFunctionMutation{
				runtime: cad.runtime,
				task:    task,
			}, nil
		}

	default:
		return nil, fmt.Errorf("task of type %q not supported", task.Type)
	}
}

func (cad *cadEngine) UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage repository.PackageRevision, oldObj, newObj *api.PackageRevision) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::UpdatePackageRevision", trace.WithAttributes())
	defer span.End()

	// Validate package lifecycle. Can only update a draft.
	switch lifecycle := oldObj.Spec.Lifecycle; lifecycle {
	default:
		return nil, fmt.Errorf("invalid original lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed:
		// Draft or proposed can be updated.
	case api.PackageRevisionLifecyclePublished:
		// TODO: generate errors that can be translated to correct HTTP responses
		return nil, fmt.Errorf("cannot update a package revision with lifecycle value %q", lifecycle)
	}
	switch lifecycle := newObj.Spec.Lifecycle; lifecycle {
	default:
		return nil, fmt.Errorf("invalid desired lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecyclePublished:
		// These values are ok
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}

	if isRecloneAndReplay(oldObj, newObj) {
		return cad.recloneAndReplay(ctx, repo, repositoryObj, newObj)
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
			cad:               cad,
			referenceResolver: cad.referenceResolver,
			namespace:         repositoryObj.Namespace,
			pkgName:           oldObj.GetName(),
		}
		mutations = append(mutations, mutation)
	}

	// Re-render if we are making changes.
	mutations = cad.conditionalAddRender(mutations)

	draft, err := repo.UpdatePackage(ctx, oldPackage)
	if err != nil {
		return nil, err
	}

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

		if err := applyResourceMutations(ctx, draft, resources, mutations); err != nil {
			return nil, err
		}
	}

	if err := draft.UpdateLifecycle(ctx, newObj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	// Updates are done.
	return draft.Close(ctx)
}

// conditionalAddRender adds a render mutation to the end of the mutations slice if the last
// entry is not already a render mutation.
func (cad *cadEngine) conditionalAddRender(mutations []mutation) []mutation {
	if len(mutations) == 0 {
		return mutations
	}

	lastMutation := mutations[len(mutations)-1]
	_, isRender := lastMutation.(*renderPackageMutation)
	if isRender {
		return mutations
	}

	return append(mutations, &renderPackageMutation{
		renderer: cad.renderer,
		runtime:  cad.runtime,
	})
}

func (cad *cadEngine) DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, oldPackage repository.PackageRevision) error {
	ctx, span := tracer.Start(ctx, "cadEngine::DeletePackageRevision", trace.WithAttributes())
	defer span.End()

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return err
	}

	if err := repo.DeletePackageRevision(ctx, oldPackage); err != nil {
		return err
	}

	return nil
}

func (cad *cadEngine) UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, oldPackage repository.PackageRevision, old, new *api.PackageRevisionResources) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::UpdatePackageResources", trace.WithAttributes())
	defer span.End()

	rev := oldPackage.GetPackageRevision()

	// Validate package lifecycle. Can only update a draft.
	switch lifecycle := rev.Spec.Lifecycle; lifecycle {
	default:
		return nil, fmt.Errorf("invalid original lifecycle value: %q", lifecycle)
	case api.PackageRevisionLifecycleDraft:
		// Only draf can be updated.
	case api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecyclePublished:
		// TODO: generate errors that can be translated to correct HTTP responses
		return nil, fmt.Errorf("cannot update a package revision with lifecycle value %q; package must be Draft", lifecycle)
	}

	repo, err := cad.cache.OpenRepository(ctx, repositoryObj)
	if err != nil {
		return nil, err
	}

	draft, err := repo.UpdatePackage(ctx, oldPackage)
	if err != nil {
		return nil, err
	}

	mutations := []mutation{
		&mutationReplaceResources{
			newResources: new,
			oldResources: old,
		},
		&renderPackageMutation{
			renderer: cad.renderer,
			runtime:  cad.runtime,
		},
	}

	apiResources, err := oldPackage.GetResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get package resources: %w", err)
	}
	resources := repository.PackageResources{
		Contents: apiResources.Spec.Resources,
	}

	if err := applyResourceMutations(ctx, draft, resources, mutations); err != nil {
		return nil, err
	}

	// No lifecycle change when updating package resources; updates are done.
	return draft.Close(ctx)
}

func applyResourceMutations(ctx context.Context, draft repository.PackageDraft, baseResources repository.PackageResources, mutations []mutation) error {
	for _, m := range mutations {
		applied, task, err := m.Apply(ctx, baseResources)
		if err != nil {
			return err
		}
		if err := draft.UpdateResources(ctx, &api.PackageRevisionResources{
			Spec: api.PackageRevisionResourcesSpec{
				Resources: applied.Contents,
			},
		}, task); err != nil {
			return err
		}
		baseResources = applied
	}

	return nil
}

func (cad *cadEngine) ListFunctions(ctx context.Context, repositoryObj *configapi.Repository) ([]repository.Function, error) {
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

	return fns, nil
}

type updatePackageMutation struct {
	cloneTask         *api.Task
	updateTask        *api.Task
	cad               CaDEngine
	referenceResolver ReferenceResolver
	namespace         string
	pkgName           string
}

func (m *updatePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
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
		cad:               m.cad,
		referenceResolver: m.referenceResolver,
	}).FetchResources(ctx, currUpstreamPkgRef, m.namespace)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("error fetching the resources for package %s with ref %+v",
			m.pkgName, *currUpstreamPkgRef)
	}

	upstreamRevision, err := (&PackageFetcher{
		cad:               m.cad,
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
	return result, m.updateTask, nil
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

func (m *mutationReplaceResources) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
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
	task := &api.Task{
		Type:  api.TaskTypePatch,
		Patch: patch,
	}

	return repository.PackageResources{Contents: new}, task, nil
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
func (cad *cadEngine) recloneAndReplay(ctx context.Context, repo repository.Repository, repositoryObj *configapi.Repository, newObj *api.PackageRevision) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cadEngine::recloneAndReplay", trace.WithAttributes())
	defer span.End()

	// For reclone and replay, we create a new package every time
	// the version should be in newObj so we will overwrite.
	draft, err := repo.CreatePackageRevision(ctx, newObj)
	if err != nil {
		return nil, err
	}

	if err := cad.applyTasks(ctx, draft, repositoryObj, newObj); err != nil {
		return nil, err
	}

	if err := draft.UpdateLifecycle(ctx, newObj.Spec.Lifecycle); err != nil {
		return nil, err
	}

	return draft.Close(ctx)
}
