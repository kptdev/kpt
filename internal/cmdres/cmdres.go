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

package cmdres

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/alphadocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "res PACKAGE",
		Aliases:    []string{"resources", "read"},
		SuggestFor: []string{},
		Short:      alphadocs.ResShort,
		Long:       alphadocs.ResLong,
		Example:    "TODO",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     true,
	}
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command
	printer printer.Printer
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdres.preRunE"
	config, err := r.cfg.ToRESTConfig()
	if err != nil {
		return errors.E(op, err)
	}

	scheme, err := createScheme()
	if err != nil {
		return errors.E(op, err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return errors.E(op, err)
	}

	r.client = c
	r.printer = printer.FromContextOrDie(r.ctx)
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdres.runE"

	if len(args) == 0 {
		return errors.E(op, "PACKAGE is a required positional argument")
	}

	packageName := args[0]

	var resources porchapi.PackageRevisionResources
	if err := r.client.Get(r.ctx, client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      packageName,
	}, &resources); err != nil {
		return errors.E(op, err)
	}

	inputs := []kio.Reader{}

	// Create kio readers
	for k, v := range resources.Spec.Resources {
		if !includeFile(k) {
			continue
		}

		inputs = append(inputs, &kio.ByteReader{
			Reader: strings.NewReader(v),
			SetAnnotations: map[string]string{
				kioutil.PathAnnotation: k,
			},
			DisableUnwrapping: true,
		})
	}

	return kio.Pipeline{
		Inputs: inputs,
		Outputs: []kio.Writer{
			kio.ByteWriter{
				Writer:                r.printer.OutStream(),
				KeepReaderAnnotations: true,
				WrappingKind:          kio.ResourceListKind,
				WrappingAPIVersion:    kio.ResourceListAPIVersion,
			},
		},
	}.Execute()
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		porchapi.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

var matchResourceContents = append(kio.MatchAll, v1.KptFileName)

func includeFile(path string) bool {
	for _, m := range matchResourceContents {
		if matched, err := filepath.Match(m, path); err == nil && matched {
			return true
		}
	}
	return false
}
