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

// Package cmdupdate contains the update command
package cmdupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:        "update [PKG_PATH@VERSION] [flags]",
		Short:      docs.UpdateShort,
		Long:       docs.UpdateShort + "\n" + docs.UpdateLong,
		Example:    docs.UpdateExamples,
		RunE:       r.runE,
		Args:       cobra.MaximumNArgs(1),
		PreRunE:    r.preRunE,
		SuggestFor: []string{"rebase", "replace"},
	}

	c.Flags().StringVarP(&r.Update.Repo, "repo", "r", "",
		"git repo url for updating contents.  defaults to the repo the package was fetched from.")
	c.Flags().StringVar(&r.strategy, "strategy", string(kptfilev1alpha2.ResourceMerge),
		"update strategy for preserving changes to the local package -- must be one of: "+
			strings.Join(update.StrategiesAsStrings(), ","))
	c.Flags().BoolVar(&r.Update.DryRun, "dry-run", false,
		"print the git patch rather than merging it.")
	c.Flags().BoolVar(&r.Update.Verbose, "verbose", false,
		"print verbose logging information.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function.
// TODO, support listing versions
type Runner struct {
	strategy string
	Update   update.Command
	Command  *cobra.Command
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = append(args, pkg.CurDir)
	}
	if r.strategy == "" {
		r.Update.Strategy = kptfilev1alpha2.ResourceMerge
	} else {
		r.Update.Strategy = kptfilev1alpha2.UpdateStrategyType(r.strategy)
	}

	parts := strings.Split(args[0], "@")
	if len(parts) > 2 {
		return errors.Errorf("at most 1 version permitted")
	}

	p, err := pkg.New(parts[0])
	if err != nil {
		return err
	}

	r.Update.FullPackagePath = string(p.UniquePath)

	relPath, err := resolveRelPath(p.UniquePath)
	if err != nil {
		return err
	}
	if strings.HasPrefix(relPath, pkg.ParentDir) {
		return errors.Errorf("package path must be under current working directory")
	}

	if len(parts) > 1 {
		r.Update.Ref = parts[1]
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	if len(r.Update.Ref) > 0 {
		fmt.Fprintf(c.ErrOrStderr(), "updating package %s to %s\n",
			r.Update.FullPackagePath, r.Update.Ref)
	} else {
		fmt.Fprintf(c.ErrOrStderr(), "updating package %s\n",
			r.Update.FullPackagePath)
	}
	if err := r.Update.Run(); err != nil {
		return err
	}

	return nil
}

func resolveRelPath(path pkg.UniquePath) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(cwd, path.String())
	if err != nil {
		return "", err
	}
	return relPath, nil
}
