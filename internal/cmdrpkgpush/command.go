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

package cmdrpkgpush

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const pushLong = `
kpt alpha rpkg push PACKAGE [DIR]

Args:

PACKAGE:
	Name of the package where to push the resources.

DIR:
	Optional path to a local directory to read resources from.


Flags:

--namespace
	Namespace containing the package.

`

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "push PACKAGE [DIR]",
		Aliases:    []string{"sink", "write"},
		SuggestFor: []string{},
		Short:      "Pushes package resources into a remote package.",
		Long:       pushLong,
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
	const op errors.Op = "cmdrpkgpush.preRunE"
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
	const op errors.Op = "cmdrpkgpush.runE"

	if len(args) == 0 {
		return errors.E(op, "PACKAGE is a required positional argument")
	}

	packageName := args[0]
	var resources map[string]string
	var err error

	if len(args) > 1 {
		resources, err = readFromDir(args[1])
	} else {
		resources, err = readFromReader(cmd.InOrStdin())
	}
	if err != nil {
		return errors.E(op, err)
	}

	if err := r.client.Update(r.ctx, &porchapi.PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResources",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      packageName,
			Namespace: *r.cfg.Namespace,
		},
		Spec: porchapi.PackageRevisionResourcesSpec{
			Resources: resources,
		},
	}); err != nil {
		return errors.E(op, err)
	}
	return nil
}

func readFromDir(dir string) (map[string]string, error) {
	resources := map[string]string{}
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		resources[rel] = string(contents)
		return nil
	}); err != nil {
		return nil, err
	}
	return resources, nil
}

func readFromReader(in io.Reader) (map[string]string, error) {
	rw := &resourceWriter{
		resources: map[string]string{},
	}

	if err := (kio.Pipeline{
		Inputs: []kio.Reader{&kio.ByteReader{
			Reader:            in,
			PreserveSeqIndent: true,
			WrapBareSeqNode:   true,
		}},
		Outputs: []kio.Writer{rw},
	}.Execute()); err != nil {
		return nil, err
	}
	return rw.resources, nil
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

type resourceWriter struct {
	resources map[string]string
}

var _ kio.Writer = &resourceWriter{}

func (w *resourceWriter) Write(nodes []*yaml.RNode) error {
	paths := map[string][]*yaml.RNode{}
	for _, node := range nodes {
		path := getPath(node)
		paths[path] = append(paths[path], node)
	}

	buf := &bytes.Buffer{}
	for path, nodes := range paths {
		bw := kio.ByteWriter{
			Writer: buf,
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				kioutil.IndexAnnotation,
			},
		}
		if err := bw.Write(nodes); err != nil {
			return err
		}
		w.resources[path] = buf.String()
		buf.Reset()
	}
	return nil
}

func getPath(node *yaml.RNode) string {
	ann := node.GetAnnotations()
	if path, ok := ann[kioutil.PathAnnotation]; ok {
		return path
	}
	ns := node.GetNamespace()
	if ns == "" {
		ns = "non-namespaced"
	}
	name := node.GetName()
	if name == "" {
		name = "unnamed"
	}
	// TODO: harden for escaping etc.
	return path.Join(ns, fmt.Sprintf("%s.yaml", name))
}
