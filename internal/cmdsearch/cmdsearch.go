// Copyright 2020 Google LLC
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

package cmdsearch

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/search"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/runner"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// NewSearchRunner returns a command SearchRunner.
func NewSearchRunner(name string) *SearchRunner {
	r := &SearchRunner{}
	c := &cobra.Command{
		Use:     "search DIR",
		Short:   shortMessage,
		RunE:    r.runE,
		PreRunE: r.preRunE,
		Args:    cobra.ExactArgs(1),
	}
	c.Flags().StringVar(&r.ByValue, "by-value", "",
		"Match by value of a field.")
	c.Flags().StringVar(&r.ByValueRegex, "by-value-regex", "",
		"Match by Regex for the value of a field. The syntax of the regular "+
			"expressions accepted is the same general syntax used by Go, Perl, Python, and "+
			"other languages. More precisely, it is the syntax accepted by RE2 and described "+
			"at https://golang.org/s/re2syntax. With the exception that it matches the entire "+
			"value of the field by default without requiring start (^) and end ($) characters.")
	c.Flags().StringVar(&r.ByPath, "by-path", "",
		"Match by path expression of a field.")
	c.Flags().StringVar(&r.PutValue, "put-value", "",
		"Set or update the value of the matching fields. Input can be a pattern "+
			"for which the capture groups are resolved using --by-value-regex input.")
	c.Flags().StringVar(&r.PutComment, "put-comment", "",
		"Set or update the line comment for matching fields. Input can be a pattern "+
			"for which the capture groups are resolved using --by-value-regex input.")
	c.Flags().BoolVarP(&r.RecurseSubPackages, "recurse-subpackages", "R", true,
		"search recursively in all the nested subpackages")

	r.Command = c
	return r
}

const shortMessage = `Search and optionally replace fields across all resources. 
Search matchers are provided by flags with --by- prefix. When multiple matchers 
are provided they are AND’ed together. --put- flags are mutually exclusive.
 `

func SearchCommand(name string) *cobra.Command {
	return NewSearchRunner(name).Command
}

// SearchRunner contains the SearchReplace function
type SearchRunner struct {
	Command            *cobra.Command
	ByValue            string
	ByValueRegex       string
	ByPath             string
	PutValue           string
	PutComment         string
	RecurseSubPackages bool
	MatchCount         int
	Writer             io.Writer
}

func (r *SearchRunner) preRunE(c *cobra.Command, args []string) error {
	if c.Flag("put-value").Changed &&
		!c.Flag("by-value").Changed &&
		!c.Flag("by-value-regex").Changed &&
		!c.Flag("by-path").Changed {
		return errors.Errorf(`at least one of ["by-value", "by-value-regex", "by-path"] must be provided`)
	}
	if c.Flag("by-value").Changed &&
		c.Flag("by-value-regex").Changed {
		return errors.Errorf(`only one of ["by-value", "by-value-regex"] can be provided`)
	}
	r.Writer = c.OutOrStdout()
	return nil
}

func (r *SearchRunner) runE(c *cobra.Command, args []string) error {
	e := runner.ExecuteCmdOnPkgs{
		Writer:             ioutil.Discard, // dummy writer, runner need not print any info
		RecurseSubPackages: r.RecurseSubPackages,
		CmdRunner:          r,
		RootPkgPath:        args[0],
		SkipPkgPathPrint:   true,
	}
	err := e.Execute()
	if err != nil {
		return err
	}
	var action string
	if r.PutComment != "" || r.PutValue != "" {
		action = "Mutated"
	} else {
		action = "Matched"
	}
	fmt.Fprintf(r.Writer, "%s %d field(s)\n", action, r.MatchCount)
	return nil
}

func (r *SearchRunner) ExecuteCmd(_ io.Writer, pkgPath string) error {
	s := search.SearchReplace{
		ByValue:      r.ByValue,
		ByValueRegex: r.ByValueRegex,
		ByPath:       r.ByPath,
		Count:        0,
		PutValue:     r.PutValue,
		PutComment:   r.PutComment,
		PackagePath:  pkgPath,
	}
	err := s.Perform(pkgPath)
	r.MatchCount += s.Count
	for _, res := range s.Result {
		fmt.Fprintf(r.Writer, "%s\nfieldPath: %s\nvalue: %s\n\n", filepath.Join(pkgPath, res.FilePath), res.FieldPath, res.Value)
	}
	return errors.Wrap(err)
}
