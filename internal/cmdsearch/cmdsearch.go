package cmdsearch

import (
	"fmt"
	"io"

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
	c.Flags().StringVar(&r.PutLiteral, "put-literal", "",
		"Set or update the value of the matching fields with the given literal value.")
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
	PutLiteral         string
	RecurseSubPackages bool
}

func (r *SearchRunner) preRunE(c *cobra.Command, args []string) error {
	if c.Flag("put-literal").Changed &&
		!c.Flag("by-value").Changed &&
		!c.Flag("by-value-regex").Changed &&
		!c.Flag("by-path").Changed {
		return errors.Errorf(`at least one of ["by-value", "by-value-regex", "by-path"] must be provided`)
	}
	if c.Flag("by-value").Changed &&
		c.Flag("by-value-regex").Changed {
		return errors.Errorf(`only one of ["by-value", "by-value-regex"] can be provided`)
	}
	return nil
}

func (r *SearchRunner) runE(c *cobra.Command, args []string) error {
	e := runner.ExecuteCmdOnPkgs{
		Writer:             c.OutOrStdout(),
		RecurseSubPackages: r.RecurseSubPackages,
		CmdRunner:          r,
		RootPkgPath:        args[0],
	}
	return e.Execute()
}

func (r *SearchRunner) ExecuteCmd(w io.Writer, pkgPath string) error {
	s := search.SearchReplace{
		ByValue:      r.ByValue,
		ByValueRegex: r.ByValueRegex,
		ByPath:       r.ByPath,
		Count:        0,
		PutLiteral:   r.PutLiteral,
	}
	err := s.Perform(pkgPath)
	fmt.Fprintf(w, "matched %d field(s)\n", s.Count)
	for filePath, nodeVals := range s.Match {
		for _, nodeVal := range nodeVals {
			fmt.Fprintf(w, "%s:  %s\n", filePath, nodeVal)
		}
	}
	return errors.Wrap(err)
}
