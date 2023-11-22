// Copyright 2023 The kpt Authors
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

package plan

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// NewCommand builds a cobra command for the plan operation.
func NewCommand() *cobra.Command {
	opt := &PlanOptions{}

	cmd := &cobra.Command{
		Use: "plan [DIR]...",
		// Short:
		// Long:
		// Example:
		PreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			objects, err := readObjects(ctx, cmd, args)
			if err != nil {
				return err
			}
			opt.Objects = objects

			opt.Out = cmd.OutOrStdout()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			restConfig, err := buildRESTConfig()
			if err != nil {
				return err
			}
			opt.RESTConfig = restConfig

			return RunPlan(cmd.Context(), opt)
		},
	}

	return cmd
}

// buildRESTConfig gets the kube config (rest.Config) for the current kubernetes cluster.
func buildRESTConfig() (*rest.Config, error) {
	restConfig, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("getting kubeconfig: %w", err)
	}
	return restConfig, nil
}

// readObjects reads objects from the arguments supplied on the command line
func readObjects(ctx context.Context, cmd *cobra.Command, args []string) ([]*unstructured.Unstructured, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("must pass file or - (for stdin) with objects to apply")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("expected exactly 1 arg, got %d", len(args))
	}

	if args[0] == "-" {
		objs, err := parseObjects(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("error reading objects: %w", err)
		}
		return objs, nil
	} else {
		return loadObjectsFromFilesystem(args[0])
	}
}

// PlanOptions holds options for a plan operation.
type PlanOptions struct {
	Out     io.Writer
	Objects []*unstructured.Unstructured

	RESTConfig *rest.Config
}

// RunPlan executes a plan operation.
func RunPlan(ctx context.Context, opt *PlanOptions) error {
	target, err := buildTarget(ctx, opt.RESTConfig)
	if err != nil {
		return err
	}

	p := &Planner{}

	plan, err := p.BuildPlan(ctx, opt.Objects, target)
	if err != nil {
		return err
	}

	printPlan(ctx, plan, opt.Out)

	return nil
}

func buildTarget(ctx context.Context, restConfig *rest.Config) (*ClusterTarget, error) {
	return NewClusterTarget(restConfig)
}
