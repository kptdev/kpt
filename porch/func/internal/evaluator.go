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

func NewEvaluatorWithConfig(functions string, config string) (pb.FunctionEvaluatorServer, error) {
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
	return &evaluator{
		cache: cache,
	}, nil
}

func (e *evaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	binary, cached := e.cache[req.Image]
	if !cached {
		return nil, status.Errorf(codes.NotFound, "Unsupported function %q", req.Image)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = bytes.NewReader(req.ResourceList)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		output := stderr.Bytes()
		return nil, status.Errorf(codes.Internal, "Failed to execute function %q: %s (%s)", req.Image, err, string(output))
	}

	outbytes := stdout.Bytes()
	errbytes := stderr.Bytes()

	klog.Infof("Evaluated %q: stdout %d bytes, stderr:\n%s", req.Image, len(outbytes), string(errbytes))

	// TODO: include stderr in the output?
	return &pb.EvaluateFunctionResponse{
		ResourceList: outbytes,
	}, nil
}
