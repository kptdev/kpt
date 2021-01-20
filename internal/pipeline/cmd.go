// Package cmdget contains the get command
package pipeline

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

// NewRunner returns a command runner
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "run [DIR]",
		Short:   "run",
		Long:    "run",
		Example: "run",
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function pipeline run command
type Runner struct {
	pkgPath string
	Command *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(c.OutOrStderr(), "error getting current dir: %v \n", err)
			return err
		}
		r.pkgPath = wd
	} else {
		// resolve and validate the provided path
		r.pkgPath = args[0]
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	klog.Infof("running pipeline command")
	executor := Executor{
		PkgPath: r.pkgPath,
	}
	err := executor.Execute()
	if err != nil {
		fmt.Fprintf(c.OutOrStderr(), "failed to run pipeline %v \n", err)
		return err
	}
	return nil
}
