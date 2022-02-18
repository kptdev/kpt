package function

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

const gcloudName = "./gcloud-config.yaml"

// This will be replaced by variant constructor
func (r *Setter) GetGcloudFnConfigPath() string {
	return filepath.Join(r.Dest, gcloudName)
}

// THis will be supported by variant constructor
var IncludeMetaResourcesFlag = true

func NewAdd(ctx context.Context) *Setter {
	r := &Setter{ctx: ctx}
	c := &cobra.Command{
		Use:   "fn [--validator=kubeval] [--mutator=set-namespace]",
		Short: `Add KRM function mutators or validators to kpt hydration pipeline`,
		Example: `
  # validate all resources by running kubeval as a container runtime.
  $ kpt editor add fn --validator=kubeval
`,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	c.Flags().StringVarP(&r.validator, "validator", "v", "", "KRM validator function")
	c.RegisterFlagCompletionFunc("validator", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.FetchFunctionImages(), cobra.ShellCompDirectiveDefault
	})
	c.Flags().StringVarP(&r.mutator, "mutator", "m", "", "KRM mutator function")
	c.RegisterFlagCompletionFunc("mutator", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.FetchFunctionImages(), cobra.ShellCompDirectiveDefault
	})
	r.Command = c
	return r
}

type Setter struct {
	validator string
	mutator   string

	// The kpt package directory
	Dest      string
	kf        *kptfile.KptFile
	Command   *cobra.Command
	ctx       context.Context
	fnResults *fnresult.ResultList
}

func (r *Setter) preRunE(c *cobra.Command, args []string) error {
	if r.validator == "" && r.mutator == "" {
		return fmt.Errorf("must specify a flag `mutator` or a `validator`")
	}
	if r.validator != "" && r.mutator != "" {
		return fmt.Errorf("only accept one of `mutator` or `validator`")
	}
	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r.Dest = wd
	} else {
		// resolve and validate the provided path
		r.Dest = args[0]
	}
	var err error
	r.Dest, err = argutil.ResolveSymlink(r.ctx, r.Dest)
	if err != nil {
		return err
	}
	r.kf, err = pkg.ReadKptfile(r.Dest)
	if err != nil {
		return err
	}
	return nil
}

func (r *Setter) getFunctionSpec(execPath string) (*runtimeutil.FunctionSpec, []string, error) {
	fn := &runtimeutil.FunctionSpec{}
	var execArgs []string
	s, err := shlex.Split(execPath)
	if err != nil {
		return nil, nil, fmt.Errorf("exec command %q must be valid: %w", execPath, err)
	}
	if len(s) > 0 {
		fn.Exec.Path = s[0]
		execArgs = s[1:]
	}
	return fn, execArgs, nil
}

func (r *Setter) runE(c *cobra.Command, _ []string) error {
	kptFile, err := pkg.ReadKptfile(r.Dest)
	if err != nil {
		return err
	}
	if kptFile.Pipeline == nil {
		kptFile.Pipeline = &kptfile.Pipeline{}
	}
	if r.mutator != "" {
		if kptFile.Pipeline.Mutators == nil {
			kptFile.Pipeline.Mutators = []kptfile.Function{}
		} else {
			for _, m := range kptFile.Pipeline.Mutators {
				if m.Name == r.mutator || m.Image == r.mutator {
					return fmt.Errorf("mutator function already exists in %v/Kptfile", r.Dest)
				}
			}
		}
		newMutator := kptfile.Function{Image: r.mutator, ConfigPath: gcloudName}
		kptFile.Pipeline.Mutators = append(kptFile.Pipeline.Mutators, newMutator)
	} else {
		if kptFile.Pipeline.Validators == nil {
			kptFile.Pipeline.Validators = []kptfile.Function{}
		} else {
			for _, m := range kptFile.Pipeline.Validators {
				if m.Name == r.validator || m.Image == r.validator {
					return fmt.Errorf("validator function already exists in %v/Kptfile", r.Dest)
				}
			}
		}
		newValidator := kptfile.Function{Image: r.validator, ConfigPath: gcloudName}
		kptFile.Pipeline.Validators = append(kptFile.Pipeline.Validators, newValidator)
	}
	if err = kptfileutil.WriteFile(r.Dest, kptFile); err != nil {
		return err
	}
	pr := printer.FromContextOrDie(r.ctx)
	pr.Printf("Kptfile is updated.\n")
	return nil
}
