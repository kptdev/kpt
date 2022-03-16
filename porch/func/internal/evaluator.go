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

package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

type evaluator struct {
	pb.UnimplementedFunctionEvaluatorServer

	builtin *builtinEvaluator
	exec    *executableEvaluator
}

func NewEvaluatorWithConfig(functions string, config string) (pb.FunctionEvaluatorServer, error) {
	builtinEval, err := newBuiltInEvaluator()
	if err != nil {
		return nil, err
	}
	execEval, err := newExecutableEvaluatorWithConfig(functions, config)
	if err != nil {
		return nil, err
	}
	return &evaluator{
		builtin: builtinEval,
		exec:    execEval,
	}, nil
}

func (e *evaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	// try to evaluate a KRM function in the following order:
	// 1) builtin 2) executable 3) pod evaluator (not implemented yet)
	resp, err := e.builtin.EvaluateFunction(ctx, req)
	var ufe UnsupportedFunctionError
	if !errors.As(err, &ufe) {
		return nil, err
	}
	resp, err = e.exec.EvaluateFunction(ctx, req)
	if errors.As(err, &ufe) {
		return nil, status.Errorf(codes.NotFound, ufe.Error())
	}
	return resp, err
}

type executableEvaluator struct {
	pb.UnimplementedFunctionEvaluatorServer

	// Fast-path function cache
	cache map[string]string
}

type configuration struct {
	Functions []function `yaml:"functions"`
}

type function struct {
	Function string   `yaml:"function"`
	Images   []string `yaml:"images"`
}

func newExecutableEvaluatorWithConfig(functions string, config string) (*executableEvaluator, error) {
	cache := map[string]string{}

	if config != "" {
		bytes, err := ioutil.ReadFile(config)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration file %q: %w", config, err)
		}
		var cfg configuration
		if err := yaml.Unmarshal(bytes, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse configuration file %q: %w", config, err)
		}

		for _, fn := range cfg.Functions {
			for _, img := range fn.Images {
				if _, exists := cache[img]; exists {
					klog.Warningf("Ignoring duplicate image %q (%s)", img, fn.Function)
				} else {
					abs, err := filepath.Abs(filepath.Join(functions, fn.Function))
					if err != nil {
						return nil, fmt.Errorf("failed to determine path to the cached function %q: %w", img, err)
					}
					klog.Infof("Caching %s as %s", img, abs)
					cache[img] = abs
				}
			}
		}
	}
	return &executableEvaluator{
		cache: cache,
	}, nil
}

func (e *executableEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	binary, cached := e.cache[req.Image]
	if !cached {
		return nil, status.Errorf(codes.NotFound, "Unsupported function %q", req.Image)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = bytes.NewReader(req.ResourceList)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to execute function %q: %s (%s)", req.Image, err, stderr.String())
	}

	outbytes := stdout.Bytes()

	klog.Infof("Evaluated %q: stdout %d bytes, stderr:\n%s", req.Image, len(outbytes), stderr.String())

	// TODO: include stderr in the output?
	return &pb.EvaluateFunctionResponse{
		ResourceList: outbytes,
		Log:          stderr.Bytes(),
	}, nil
}

type UnsupportedFunctionError struct {
	Image string
}

func (e UnsupportedFunctionError) Error() string {
	return fmt.Sprintf("Unsupported function %q", e.Image)
}
