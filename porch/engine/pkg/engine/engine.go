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
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/cache"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type CaDEngine interface {
	OpenRepository(repositorySpec *configapi.Repository, auth repository.AuthOptions) (repository.Repository, error)
	CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, obj *api.PackageRevision) (repository.PackageRevision, error)
	UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, oldPackage repository.PackageRevision, old, new *api.PackageRevision) (repository.PackageRevision, error)
	UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, oldPackage repository.PackageRevision, old, new *api.PackageRevisionResources) (repository.PackageRevision, error)
	DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, obj repository.PackageRevision) error
	ListFunctions(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions) ([]repository.Function, error)
}

func NewCaDEngine(cache *cache.Cache, functionRunnerAddress string) (CaDEngine, error) {
	runtime, err := createFunctionRuntime(functionRunnerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create function runtime: %w", err)
	}

	return &cadEngine{
		cache:    cache,
		renderer: kpt.NewRenderer(),
		runtime:  runtime,
	}, nil
}

func createFunctionRuntime(address string) (kpt.FunctionRuntime, error) {
	if address == "" {
		klog.Warningf("Using simple kpt function runner (in-process)")
		return kpt.NewSimpleFunctionRuntime(), nil
	}

	klog.Infof("Dialing grpc function runner %q", address)

	cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc function evaluator: %w", err)
	}

	return &grpcRuntime{
		cc:     cc,
		client: evaluator.NewFunctionEvaluatorClient(cc),
	}, err
}

type cadEngine struct {
	cache    *cache.Cache
	renderer fn.Renderer
	runtime  fn.FunctionRuntime
}

var _ CaDEngine = &cadEngine{}

type mutation interface {
	Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error)
}

func (cad *cadEngine) OpenRepository(repositorySpec *configapi.Repository, auth repository.AuthOptions) (repository.Repository, error) {
	return cad.cache.OpenRepository(repositorySpec, auth)
}

func (cad *cadEngine) CreatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, obj *api.PackageRevision) (repository.PackageRevision, error) {
	repo, err := cad.cache.OpenRepository(repositoryObj, auth)
	if err != nil {
		return nil, err
	}
	draft, err := repo.CreatePackageRevision(ctx, obj)
	if err != nil {
		return nil, err
	}

	var mutations []mutation
	for i := range obj.Spec.Tasks {
		task := &obj.Spec.Tasks[i]
		mutation, err := cad.mapTaskToMutation(ctx, obj, task)
		if err != nil {
			return nil, err
		}
		mutations = append(mutations, mutation)
	}

	// Render package after creation.
	mutations = append(mutations, &renderPackageMutation{
		renderer: cad.renderer,
		runtime:  cad.runtime,
	})

	baseResources := repository.PackageResources{}

	return updateDraft(ctx, draft, baseResources, mutations)
}

func (cad *cadEngine) mapTaskToMutation(ctx context.Context, obj *api.PackageRevision, task *api.Task) (mutation, error) {
	switch task.Type {
	case api.TaskTypeClone:
		if task.Clone == nil {
			return nil, fmt.Errorf("clone not set for task of type %q", task.Type)
		}
		return &clonePackageMutation{
			task: task,
			name: obj.Spec.PackageName,
		}, nil

	case api.TaskTypePatch:
		if task.Patch == nil {
			return nil, fmt.Errorf("patch not set for task of type %q", task.Type)
		}
		// TODO: support patch?
		return nil, fmt.Errorf("patch not supported on create")

	case api.TaskTypeEval:
		if task.Eval == nil {
			return nil, fmt.Errorf("eval not set for task of type %q", task.Type)
		}
		return &evalFunctionMutation{
			runtime: cad.runtime,
			task:    task,
		}, nil

	default:
		return nil, fmt.Errorf("task of type %q not supported", task.Type)
	}
}

func (cad *cadEngine) UpdatePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, oldPackage repository.PackageRevision, oldObj, newObj *api.PackageRevision) (repository.PackageRevision, error) {
	repo, err := cad.cache.OpenRepository(repositoryObj, auth)
	if err != nil {
		return nil, err
	}

	var mutations []mutation
	if len(oldObj.Spec.Tasks) != len(newObj.Spec.Tasks) {
		return nil, fmt.Errorf("adding/removing tasks is not yet supported")
	}

	for i := range oldObj.Spec.Tasks {
		oldTask := &oldObj.Spec.Tasks[i]
		newTask := &newObj.Spec.Tasks[i]

		if oldTask.Type != newTask.Type {
			return nil, fmt.Errorf("changing task types is not yet supported")
		}

		unchanged := reflect.DeepEqual(oldTask, newTask)
		if unchanged {
			continue
		}

		switch newTask.Type {
		case api.TaskTypeClone:
			if newTask.Clone == nil {
				return nil, fmt.Errorf("clone not set for task of type %q", newTask.Type)
			}
			if i != 0 {
				return nil, fmt.Errorf("clone only supported as first task")
			}
			mutation := &updatePackageMutation{
				task: newTask,
			}
			mutations = append(mutations, mutation)

		default:
			return nil, fmt.Errorf("updating task of type %q not supported", newTask.Type)
		}
	}

	mutations = append(mutations, &renderPackageMutation{
		renderer: cad.renderer,
		runtime:  cad.runtime,
	})

	draft, err := repo.UpdatePackage(ctx, oldPackage)
	if err != nil {
		return nil, err
	}

	apiResources, err := oldPackage.GetResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get package resources: %w", err)
	}
	resources := repository.PackageResources{
		Contents: apiResources.Spec.Resources,
	}

	return updateDraft(ctx, draft, resources, mutations)
}

func (cad *cadEngine) DeletePackageRevision(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, oldPackage repository.PackageRevision) error {
	repo, err := cad.cache.OpenRepository(repositoryObj, auth)
	if err != nil {
		return err
	}

	if err := repo.DeletePackageRevision(ctx, oldPackage); err != nil {
		return err
	}

	return nil
}

func (cad *cadEngine) UpdatePackageResources(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions, oldPackage repository.PackageRevision, old, new *api.PackageRevisionResources) (repository.PackageRevision, error) {
	repo, err := cad.cache.OpenRepository(repositoryObj, auth)
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
	}

	apiResources, err := oldPackage.GetResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get package resources: %w", err)
	}
	resources := repository.PackageResources{
		Contents: apiResources.Spec.Resources,
	}

	return updateDraft(ctx, draft, resources, mutations)
}

func updateDraft(ctx context.Context, draft repository.PackageDraft, baseResources repository.PackageResources, mutations []mutation) (repository.PackageRevision, error) {
	for _, m := range mutations {
		applied, task, err := m.Apply(ctx, baseResources)
		if err != nil {
			return nil, err
		}
		if err := draft.UpdateResources(ctx, &api.PackageRevisionResources{
			Spec: api.PackageRevisionResourcesSpec{
				Resources: applied.Contents,
			},
		}, task); err != nil {
			return nil, err
		}
		baseResources = applied
	}

	// Updates are done.
	return draft.Close(ctx)
}

func (cad *cadEngine) ListFunctions(ctx context.Context, repositoryObj *configapi.Repository, auth repository.AuthOptions) ([]repository.Function, error) {
	repo, err := cad.cache.OpenRepository(repositoryObj, auth)
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
	task *api.Task
}

func (m *updatePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	// TODO: load directly from source repository
	dir, err := ioutil.TempDir("", "kpt-pkg-update-*")
	if err != nil {
		return repository.PackageResources{}, nil, err
	}
	defer os.RemoveAll(dir)

	if err := writeResourcesToDirectory(dir, resources); err != nil {
		return repository.PackageResources{}, nil, err
	}

	ref := m.task.Clone.Upstream.Git.Ref

	// TODO: This is a hack
	packageName := filepath.Base(m.task.Clone.Upstream.Git.Directory)
	packageName = strings.TrimPrefix(packageName, ".git")

	packageDir := filepath.Join(dir, packageName)
	if err := kpt.PkgUpdate(ctx, ref, packageDir, kpt.PkgUpdateOpts{}); err != nil {
		return repository.PackageResources{}, nil, err
	}

	loaded, err := loadResourcesFromDirectory(dir)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	return loaded, m.task, nil
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

		contents, err := ioutil.ReadFile(path)
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

type evalFunctionMutation struct {
	runtime fn.FunctionRuntime
	task    *api.Task
}

func (m *evalFunctionMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	e := m.task.Eval

	// TODO: Apply should accept filesystem instead of PackageResources

	runner, err := m.runtime.GetRunner(ctx, &v1.Function{
		Image:     e.Image,
		ConfigMap: e.ConfigMap,
	})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to create function runner: %w", err)
	}

	pr := &packageReader{
		input: resources,
		extra: map[string]string{},
	}

	// r := &kio.LocalPackageReader{
	// 	PackagePath:        "/",
	// 	IncludeSubpackages: true,
	// 	FileSystem:         filesys.FileSystemOrOnDisk{FileSystem: fs},
	// 	WrapBareSeqNode:    true,
	// }

	var rl bytes.Buffer
	w := &kio.ByteWriter{
		Writer:                &rl,
		KeepReaderAnnotations: true,
		FunctionConfig:        &yaml.RNode{},
		WrappingKind:          kio.ResourceListKind,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
	}

	pipeline := kio.Pipeline{
		Inputs:  []kio.Reader{pr},
		Outputs: []kio.Writer{w},
	}

	if err := pipeline.Execute(); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to serialize package: %w", err)
	}

	// Evaluate the function
	var output bytes.Buffer
	if err := runner.Run(&rl, &output); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to evaluate function: %w", err)
	}

	result := repository.PackageResources{
		Contents: map[string]string{},
	}

	if err := (kio.Pipeline{
		Inputs: []kio.Reader{&kio.ByteReader{
			Reader:            &output,
			PreserveSeqIndent: true,
			WrapBareSeqNode:   true,
		}},
		Outputs: []kio.Writer{&packageWriter{
			output: result,
		}},
	}.Execute()); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to de-serialize function result: %w", err)
	}

	// Return extras. TODO: Apply should accept FS.
	for k, v := range pr.extra {
		result.Contents[k] = v
	}

	return result, m.task, nil
}

type mutationReplaceResources struct {
	newResources *api.PackageRevisionResources
	oldResources *api.PackageRevisionResources
}

func (m *mutationReplaceResources) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	patch := &api.PackagePatchTaskSpec{}
	task := &api.Task{
		Type:  "patch",
		Patch: patch,
	}

	old := resources.Contents
	new := m.newResources.Spec.Resources

	for k, newV := range new {
		oldV, ok := old[k]
		// New config or changed config
		if !ok || newV != oldV {
			patch.Patches = append(patch.Patches, k)
		}
	}
	for k := range old {
		// Deleted config
		if _, ok := new[k]; !ok {
			patch.Patches = append(patch.Patches, k)
		}
	}

	return repository.PackageResources{Contents: new}, task, nil
}
