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

package cmdsub

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/internal/util/sub"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type Help struct {
	Substitutions []kptfile.Substitution
	Kptfile       kptfile.KptFile
}

func (r *Help) preRunE(c *cobra.Command, args []string) error {
	// available substitutions are in the Kptfile
	var err error
	r.Kptfile, err = kptfileutil.ReadFile(args[0])
	if err != nil {
		return errors.WrapPrefixf(err, "failed reading %s",
			filepath.Join(args[0], kptfile.KptFileName))
	}

	// find the substitution matching the one specified by the user
	for i := range r.Kptfile.Substitutions {
		s := r.Kptfile.Substitutions[i]
		if len(args) == 1 {
			r.Substitutions = append(r.Substitutions, s)
			continue
		}
		if s.Name == args[1] {
			r.Substitutions = append(r.Substitutions, s)
			continue
		}
	}
	return nil
}

func (r *Help) runE(c *cobra.Command, args []string) error {
	rw := &kio.LocalPackageReader{
		PackagePath: args[0],
	}
	var fltrs []kio.Filter
	var subs []*sub.Sub
	for i := range r.Substitutions {
		s := &sub.Sub{Substitution: r.Substitutions[i]}
		subs = append(subs, s)
		fltrs = append(fltrs, s)
	}
	// check the substitions
	err := kio.Pipeline{
		Inputs:  []kio.Reader{rw},
		Filters: fltrs,
	}.Execute()
	if err != nil {
		return err
	}
	remaining := false
	table := tablewriter.NewWriter(c.OutOrStdout())
	table.SetRowLine(false)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator(" ")
	table.SetCenterSeparator(" ")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{
		"NAME", "DESCRIPTION", "TYPE", "MARKER", "REMAINING", "PERFORMED", "PERFORMED VALUE",
	})
	for i := range subs {
		s := subs[i]
		remaining = remaining || s.Count > 0
		table.Append([]string{
			s.Name,
			"'" + s.Description + "'",
			string(s.Type),
			s.Marker,
			fmt.Sprintf("%d", s.Count),
			fmt.Sprintf("%d", s.Performed),
			s.PerformedValue,
		})
	}
	table.Render()

	if remaining {
		os.Exit(1)
	}
	return nil
}
