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

package push

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/commands/alpha/rpkg/util"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
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

const (
	command = "cmdrpkgpush"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "push PACKAGE [DIR]",
		Aliases:    []string{"sink", "write"},
		SuggestFor: []string{},
		Short:      rpkgdocs.PushShort,
		Long:       rpkgdocs.PushShort + "\n" + rpkgdocs.PushLong,
		Example:    rpkgdocs.PushExamples,
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     porch.HidePorchCommands,
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

func (r *runner) preRunE(_ *cobra.Command, _ []string) error {
	const op errors.Op = command + ".preRunE"
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
	const op errors.Op = command + ".runE"

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

	pkgResources := porchapi.PackageRevisionResources{
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
	}

	rv, err := util.GetResourceVersion(&pkgResources)
	if err != nil {
		return errors.E(op, err)
	}
	pkgResources.ResourceVersion = rv
	if err = util.RemoveRevisionMetadata(&pkgResources); err != nil {
		return errors.E(op, err)
	}

	if err := r.client.Update(r.ctx, &pkgResources); err != nil {
		return errors.E(op, err)
	}
	rs := pkgResources.Status.RenderStatus
	if rs.Err != "" {
		r.printer.Printf("Package is updated, but failed to render the package.\n")
		r.printer.Printf("Error: %s\n", rs.Err)
	}
	if len(rs.Result.Items) > 0 {
		for _, result := range rs.Result.Items {
			r.printer.Printf("[RUNNING] %q \n", result.Image)
			printOpt := printer.NewOpt()
			if result.ExitCode != 0 {
				r.printer.OptPrintf(printOpt, "[FAIL] %q\n", result.Image)
			} else {
				r.printer.OptPrintf(printOpt, "[PASS] %q\n", result.Image)
			}
			r.printFnResult(result, printOpt)
		}
	}
	return nil
}

// printFnResult prints given function result in a user friendly
// format on kpt CLI.
func (r *runner) printFnResult(fnResult *porchapi.Result, opt *printer.Options) {
	if len(fnResult.Results) > 0 {
		// function returned structured results
		var lines []string
		for _, item := range fnResult.Results {
			lines = append(lines, str(item))
		}
		ri := &fnruntime.MultiLineFormatter{
			Title:          "Results",
			Lines:          lines,
			TruncateOutput: printer.TruncateOutput,
		}
		r.printer.OptPrintf(opt, "%s", ri.String())
	}
}

// String provides a human-readable message for the result item
func str(i porchapi.ResultItem) string {
	identifier := i.ResourceRef
	var idStringList []string
	if identifier != nil {
		if identifier.APIVersion != "" {
			idStringList = append(idStringList, identifier.APIVersion)
		}
		if identifier.Kind != "" {
			idStringList = append(idStringList, identifier.Kind)
		}
		if identifier.Namespace != "" {
			idStringList = append(idStringList, identifier.Namespace)
		}
		if identifier.Name != "" {
			idStringList = append(idStringList, identifier.Name)
		}
	}
	formatString := "[%s]"
	severity := i.Severity
	// We default Severity to Info when converting a result to a message.
	if i.Severity == "" {
		severity = "info"
	}
	list := []interface{}{severity}
	if len(idStringList) > 0 {
		formatString += " %s"
		list = append(list, strings.Join(idStringList, "/"))
	}
	if i.Field != nil {
		formatString += " %s"
		list = append(list, i.Field.Path)
	}
	formatString += ": %s"
	list = append(list, i.Message)
	return fmt.Sprintf(formatString, list...)
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
