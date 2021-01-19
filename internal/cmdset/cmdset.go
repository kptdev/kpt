package cmdset

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/runner"
	"github.com/GoogleContainerTools/kpt/internal/util/setters"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// NewSetRunner returns a command runner.
func NewSetRunner(parent string) *SetRunner {
	r := &SetRunner{}
	c := &cobra.Command{
		Use:     "set DIR --value [SETTER_NAME=SETTER_VALUE]",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}
	r.Command = c
	c.Flags().StringVar(&r.KeyValue, "value", "",
		"optional flag, the value of the setter to be set to")
	c.Flags().StringVar(&r.SetBy, "set-by", "",
		"annotate the field with who set it")
	c.Flags().StringVar(&r.Description, "description", "",
		"annotate the field with a description of its value")
	c.Flags().BoolVarP(&r.RecurseSubPackages, "recurse-subpackages", "R", false,
		"sets recursively in all the nested subpackages")

	return r
}

func SetCommand(parent string) *cobra.Command {
	return NewSetRunner(parent).Command
}

type SetRunner struct {
	Command            *cobra.Command
	Set                setters.FieldSetter
	SetBy              string
	Description        string
	KeyValue           string
	Name               string
	Value              string
	ListValues         []string
	RecurseSubPackages bool
	Writer             io.Writer
}

func (r *SetRunner) preRunE(c *cobra.Command, args []string) error {
	r.Writer = c.OutOrStdout()
	valueFlagSet := c.Flag("value").Changed

	if !valueFlagSet {
		return errors.Errorf("value flag must be set")
	}

	// r.KeyValue holds SETTER_NAME=SETTER_VALUE, split them into name and value
	keyValue := strings.SplitN(r.KeyValue, "=", 2)
	if len(keyValue) < 2 {
		return errors.Errorf(`input to value flag must follow the format "SETTER_NAME=SETTER_VALUE"`)
	}

	r.Name = keyValue[0]
	setterValue := keyValue[1]

	// get the list values from the input
	r.ListValues = setters.ListValues(setterValue, ",")
	// check if the input is not a list
	if r.ListValues == nil {
		// the input is a scalar value
		r.Value = setterValue
	}

	return nil
}

func (r *SetRunner) runE(c *cobra.Command, args []string) error {
	e := runner.ExecuteCmdOnPkgs{
		NeedKptFile:        true,
		RootPkgPath:        args[0],
		RecurseSubPackages: r.RecurseSubPackages,
		CmdRunner:          r,
	}
	err := e.Execute()
	if err != nil {
		return runner.HandleError(c, err)
	}
	return nil
}

func (r *SetRunner) ExecuteCmd(pkgPath string) error {
	r.Set = setters.FieldSetter{
		Name:               r.Name,
		Value:              r.Value,
		ListValues:         r.ListValues,
		Description:        r.Description,
		SetBy:              r.SetBy,
		Count:              0,
		OpenAPIPath:        filepath.Join(pkgPath, kptfile.KptFileName),
		OpenAPIFileName:    kptfile.KptFileName,
		ResourcesPath:      pkgPath,
		RecurseSubPackages: r.RecurseSubPackages,
		IsSet:              true,
	}
	count, err := r.Set.Set()
	fmt.Fprintf(r.Writer, "\n%s/\n", pkgPath)
	if err != nil {
		// return err if RecurseSubPackages is false
		if !r.Set.RecurseSubPackages {
			return err
		}
		// print error message and continue if RecurseSubPackages is true
		fmt.Fprintf(r.Writer, "%s\n", err.Error())
	} else {
		fmt.Fprintf(r.Writer, "set %d field(s) of setter %q to value %q\n", count, r.Set.Name, r.Set.Value)
	}
	return nil
}
