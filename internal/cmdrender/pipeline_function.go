// Copyright 2021 Google LLC
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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package cmdrender

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/cmdrender/runtime"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// newFnRunner returns a fnRunner from the image and configs of
// this function.
func newFnRunner(
	ctx context.Context,
	f *kptfilev1alpha2.Function,
	pkgPath types.UniquePath,
	truncateOutput bool,
) (kio.Filter, error) {
	config, err := newFnConfig(f, pkgPath)
	if err != nil {
		return nil, err
	}

	cfn := &runtime.ContainerFn{
		Path:           pkgPath,
		Image:          f.Image,
		Ctx:            ctx,
		TruncateOutput: truncateOutput,
	}

	return &runtimeutil.FunctionFilter{
		Run:            cfn.Run,
		FunctionConfig: config,
	}, nil
}

func newFnConfig(f *kptfilev1alpha2.Function, pkgPath types.UniquePath) (*yaml.RNode, error) {
	const op errors.Op = "fn.readConfig"
	var fn errors.Fn = errors.Fn(f.Image)

	var node *yaml.RNode
	switch {
	case f.ConfigPath != "":
		path := filepath.Join(string(pkgPath), f.ConfigPath)
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.E(op, fn,
				fmt.Errorf("missing function config %q", f.ConfigPath))
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		node, err = yaml.Parse(string(b))
		if err != nil {
			return nil, errors.E(op, fn, fmt.Errorf("invalid function config %q %w", f.ConfigPath, err))
		}
		// directly use the config from file
		return node, nil
	case !kptfilev1alpha2.IsNodeZero(&f.Config):
		// directly use the inline config
		return yaml.NewRNode(&f.Config), nil
	case len(f.ConfigMap) != 0:
		node = yaml.NewMapRNode(&f.ConfigMap)
		if node == nil {
			return nil, nil
		}
		// create a ConfigMap only for configMap config
		configNode := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
		err := configNode.PipeE(yaml.SetField("data", node))
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		return configNode, nil
	}
	// no need to return ConfigMap if no config given
	return nil, nil
}
