package engine

import (
	"bytes"
	"context"
	"fmt"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type evalFunctionMutation struct {
	runtime fn.FunctionRuntime
	task    *api.Task
}

func (m *evalFunctionMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	e := m.task.Eval

	// TODO: Apply should accept filesystem instead of PackageResources

	runner, err := m.runtime.GetRunner(ctx, &v1.Function{
		Image: e.Image,
	})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to create function runner: %w", err)
	}

	var functionConfig *yaml.RNode
	if m.task.Eval.ConfigMap != nil {
		if cm, err := kpt.NewConfigMap(m.task.Eval.ConfigMap); err != nil {
			return repository.PackageResources{}, nil, fmt.Errorf("failed to create function config: %w", err)
		} else {
			functionConfig = cm
		}
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
		FunctionConfig:        functionConfig,
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
