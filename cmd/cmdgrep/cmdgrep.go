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
package cmdgrep

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"kpt.dev/util/argutil"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "grep QUERY [DIR]...",
		Short: "Search for matching Resources in a package",
		Long: `Search for matching Resources in a package.
  QUERY:
    Query to match expressed as 'path.to.field=value'.
    Maps and fields are matched as '.field-name' or '.map-key'
    List elements are matched as '[list-elem-field=field-value]'
    The value to match is expressed as '=value'
    '.' as part of a key or value can be escaped as '\.'

  DIR:
    Path to local package directory.
`,
		Example: `# find Deployment Resources
kpt grep "kind=Deployment" my-package/

# find Resources named nginx
kpt grep "metadata.name=nginx" my-package/

# use tree to display matching Resources
kpt grep "metadata.name=nginx" my-package/ | kpt tree

# look for Resources matching a specific container image
kpt grep "spec.template.spec.containers[name=nginx].image=nginx:1\.7\.9" my-package/ | kpt tree
`,
		PreRunE:      r.preRunE,
		RunE:         r.runE,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")
	c.Flags().BoolVar(&r.KeepAnnotations, "annotate", true,
		"annotate resources with their file origins.")
	c.Flags().BoolVarP(&r.InvertMatch, "invert-match", "v", false,
		" Selected Resources are those not matching any of the specified patterns..")

	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages bool
	KeepAnnotations    bool
	C                  *cobra.Command
	filters.GrepFilter
	Format bool
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.GrepFilter.Compare = func(a, b string) (int, error) {
		qa, err := resource.ParseQuantity(a)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", a, err)
		}
		qb, err := resource.ParseQuantity(b)
		if err != nil {
			return 0, err
		}

		return qa.Cmp(qb), err
	}
	parts, err := argutil.ParseFieldPath(args[0])
	if err != nil {
		return err
	}

	var last []string
	if strings.Contains(parts[len(parts)-1], ">=") {
		last = strings.Split(parts[len(parts)-1], ">=")
		r.MatchType = filters.GreaterThanEq
	} else if strings.Contains(parts[len(parts)-1], "<=") {
		last = strings.Split(parts[len(parts)-1], "<=")
		r.MatchType = filters.LessThanEq
	} else if strings.Contains(parts[len(parts)-1], ">") {
		last = strings.Split(parts[len(parts)-1], ">")
		r.MatchType = filters.GreaterThan
	} else if strings.Contains(parts[len(parts)-1], "<") {
		last = strings.Split(parts[len(parts)-1], "<")
		r.MatchType = filters.LessThan
	} else {
		last = strings.Split(parts[len(parts)-1], "=")
		r.MatchType = filters.Regexp
	}
	if len(last) > 2 {
		return fmt.Errorf(
			"ambiguous match -- multiple of ['<', '>', '<=', '>=', '=' in final path element: %s",
			parts[len(parts)-1])
	}

	if len(last) > 1 {
		r.Value = last[1]
	}

	r.Path = append(parts[:len(parts)-1], last[0])
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var filters = []kio.Filter{r.GrepFilter}

	var inputs []kio.Reader
	for _, a := range args[1:] {
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath:        a,
			IncludeSubpackages: r.IncludeSubpackages,
		})
	}
	if len(inputs) == 0 {
		inputs = append(inputs, kio.ByteReader{Reader: c.InOrStdin()})
	}

	return kio.Pipeline{
		Inputs:  inputs,
		Filters: filters,
		Outputs: []kio.Writer{kio.ByteWriter{
			Writer:                c.OutOrStdout(),
			KeepReaderAnnotations: r.KeepAnnotations,
		}},
	}.Execute()
}
