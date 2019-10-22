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

// Package cndcat contains the fmt command
package cmdtree

import (
	"path/filepath"
	"strings"

	"kpt.dev/kpt/util/argutil"
	"lib.kpt.dev/kio/filters"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "tree DIR",
		Short: "Display package Resource structure",
		Long: `Display package Resource structure.

kpt tree may be used to print Resources in a package or cluster, preserving structure

Args:

  DIR:
    Path to local package directory.

Resource fields may be printed as part of the Resources by specifying the fields as flags.

kpt tree has build-in support for printing common fields, such as replicas, container images,
container names, etc.

kpt tree supports printing arbitrary fields using the '--field' flag.

By default, kpt tree uses the package structure for the tree structure, however when printing
from the cluster, the Resource graph structure may be used instead.
`,
		Example: `# print Resources using package structure
kpt tree my-package/

# print replicas, container name, and container image and fields for Resources
kpt tree my-package --replicas --image --name

# print all common Resource fields
kpt tree my-package/ --all

# print the "foo"" annotation
kpt tree my-package/ --field "metadata.annotations.foo" 

# print the "foo"" annotation
kubectl get all -o yaml | kpt tree my-package/ --structure=graph \
  --field="status.conditions[type=Completed].status"

# print live Resources from a cluster using graph for structure
kubectl get all -o yaml | kpt tree --replicas --name --image --structure=graph


# print live Resources using graph for structure
kubectl get all,applications,releasetracks -o yaml | kpt tree --structure=graph \
  --name --image --replicas \
  --field="status.conditions[type=Completed].status" \
  --field="status.conditions[type=Complete].status" \
  --field="status.conditions[type=Ready].status" \
  --field="status.conditions[type=ContainersReady].status"
`,
		RunE:         r.runE,
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")

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
	c.Flags().BoolVar(&r.includeReconcilers, "include-reconcilers", false,
		"if true, include reconciler Resources in the output.")
	c.Flags().BoolVar(&r.excludeNonReconcilers, "exclude-non-reconcilers", false,
		"if true, exclude non-reconciler Resources in the output.")
	c.Flags().StringVar(&r.structure, "structure", "package",
		"structure to use for the tree.  may be 'package' or 'owners'.")

	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages    bool
	C                     *cobra.Command
	name                  bool
	resources             bool
	ports                 bool
	images                bool
	replicas              bool
	all                   bool
	env                   bool
	args                  bool
	cmd                   bool
	fields                []string
	includeReconcilers    bool
	excludeNonReconcilers bool
	structure             string
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var input kio.Reader
	var root = "."
	if len(args) == 1 {
		root = filepath.Clean(args[0])
		input = kio.LocalPackageReader{PackagePath: args[0]}
	} else {
		input = &kio.ByteReader{Reader: c.InOrStdin()}
	}

	var fields []kio.TreeWriterField
	for _, field := range r.fields {
		path, err := argutil.ParseFieldPath(field)
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
	fltrs := []kio.Filter{&filters.IsReconcilerFilter{
		ExcludeReconcilers:    !r.includeReconcilers,
		IncludeNonReconcilers: !r.excludeNonReconcilers,
	}}

	return kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: fltrs,
		Outputs: []kio.Writer{kio.TreeWriter{
			Root:      root,
			Writer:    c.OutOrStdout(),
			Fields:    fields,
			Structure: kio.TreeStructure(r.structure)}},
	}.Execute()
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
