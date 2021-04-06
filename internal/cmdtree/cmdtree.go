// Copyright 2019 Google LLC
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

package cmdtree

import (
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/runner"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func GetTreeRunner(name string) *TreeRunner {
	r := &TreeRunner{}
	c := &cobra.Command{
		Use:     "tree [DIR | -<STDIN>]",
		Short:   pkgdocs.TreeShort,
		Long:    pkgdocs.TreeLong,
		Example: pkgdocs.TreeExamples,
		RunE:    r.runE,
		Args:    cobra.MaximumNArgs(1),
	}

	// TODO(pwittrock): Figure out if these are the right things to expose, and consider making it
	// a list of options instead of individual flags
	c.Flags().BoolVar(&r.name, "name", false, "print name field")
	c.Flags().BoolVar(&r.resources, "resources", false, "print resources field")
	c.Flags().BoolVar(&r.ports, "ports", false, "print ports field")
	c.Flags().BoolVar(&r.images, "image", false, "print image field")
	c.Flags().BoolVar(&r.replicas, "replicas", false, "print replicas field")
	c.Flags().BoolVar(&r.args, "args", false, "print args field")
	c.Flags().BoolVar(&r.cmd, "command", false, "print command field")
	c.Flags().BoolVar(&r.env, "env", false, "print env field")
	c.Flags().BoolVar(&r.all, "all", false, "print all field infos")
	c.Flags().StringSliceVar(&r.fields, "field", []string{}, "print field")
	c.Flags().BoolVar(&r.includeLocal, "include-local", false,
		"if true, include local-config in the output.")
	c.Flags().BoolVar(&r.excludeNonLocal, "exclude-non-local", false,
		"if true, exclude non-local-config in the output.")
	c.Flags().StringVar(&r.structure, "graph-structure", "",
		"Graph structure to use for printing the tree.  may be any of: "+
			strings.Join(kio.GraphStructures, ","))

	r.Command = c
	return r
}

func NewCommand(name string) *cobra.Command {
	return GetTreeRunner(name).Command
}

// TreeRunner contains the run function
type TreeRunner struct {
	Command         *cobra.Command
	name            bool
	resources       bool
	ports           bool
	images          bool
	replicas        bool
	all             bool
	env             bool
	args            bool
	cmd             bool
	fields          []string
	includeLocal    bool
	excludeNonLocal bool
	structure       string
}

func (r *TreeRunner) runE(c *cobra.Command, args []string) error {
	var input kio.Reader
	var root = "."
	if len(args) == 0 {
		args = append(args, root)
	}
	if args[0] == "-" {
		input = &kio.ByteReader{Reader: c.InOrStdin()}
	} else {
		root = filepath.Clean(args[0])
		input = kio.LocalPackageReader{PackagePath: args[0]}
	}

	var fields []kio.TreeWriterField
	for _, field := range r.fields {
		path, err := runner.ParseFieldPath(field)
		if err != nil {
			return err
		}
		fields = append(fields, newField(path...))
	}

	if r.name || (r.all && !c.Flag("name").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "name"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "name"),
		)
	}
	if r.images || (r.all && !c.Flag("image").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "image"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "image"),
		)
	}

	if r.cmd || (r.all && !c.Flag("command").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "command"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "command"),
		)
	}
	if r.args || (r.all && !c.Flag("args").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "args"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "args"),
		)
	}
	if r.env || (r.all && !c.Flag("env").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "env"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "env"),
		)
	}

	if r.replicas || (r.all && !c.Flag("replicas").Changed) {
		fields = append(fields,
			newField("spec", "replicas"),
		)
	}
	if r.resources || (r.all && !c.Flag("resources").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "resources"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "resources"),
		)
	}
	if r.ports || (r.all && !c.Flag("ports").Changed) {
		fields = append(fields,
			newField("spec", "containers", "[name=.*]", "ports"),
			newField("spec", "template", "spec", "containers", "[name=.*]", "ports"),
			newField("spec", "ports"),
		)
	}

	// show reconcilers in tree
	fltrs := []kio.Filter{&filters.IsLocalConfig{
		IncludeLocalConfig:    r.includeLocal,
		ExcludeNonLocalConfig: r.excludeNonLocal,
	}}

	return runner.HandleError(c, kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: fltrs,
		Outputs: []kio.Writer{kio.TreeWriter{
			Root:            root,
			Writer:          c.OutOrStdout(),
			Fields:          fields,
			Structure:       kio.TreeStructure(r.structure),
			OpenAPIFileName: kptfilev1alpha2.KptFileName,
		}},
	}.Execute())
}

func newField(val ...string) kio.TreeWriterField {
	if strings.HasPrefix(strings.Join(val, "."), "spec.template.spec.containers") {
		return kio.TreeWriterField{
			Name:        "spec.template.spec.containers",
			PathMatcher: yaml.PathMatcher{Path: val, StripComments: true},
			SubName:     val[len(val)-1],
		}
	}

	if strings.HasPrefix(strings.Join(val, "."), "spec.containers") {
		return kio.TreeWriterField{
			Name:        "spec.containers",
			PathMatcher: yaml.PathMatcher{Path: val, StripComments: true},
			SubName:     val[len(val)-1],
		}
	}

	return kio.TreeWriterField{
		Name:        strings.Join(val, "."),
		PathMatcher: yaml.PathMatcher{Path: val, StripComments: true},
	}
}
