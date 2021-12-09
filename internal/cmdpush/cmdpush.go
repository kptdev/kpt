package cmdpush

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/push"
	"github.com/GoogleContainerTools/kpt/internal/util/upstream"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:        "push [DIR@VERSION] [flags]",
		Args:       cobra.MaximumNArgs(1),
		Short:      docs.GetShort,
		Long:       docs.GetShort + "\n" + docs.GetLong,
		Example:    docs.GetExamples,
		RunE:       r.runE,
		PreRunE:    r.preRunE,
	}

	c.Flags().StringVar(&r.Origin, "origin", "",
		"assigns or changes the location where the package should be pushed. Default is to push it to "+
		"the origin from which the package was pulled.")
	c.Flags().BoolVar(&r.Increment, "increment", false,
		"increment the version of the package when pushed. This will increment the DIR@VERSION if provided, "+
		"otherwise it will increment the origin's version when pulled. The version must be semver or integer, and "+
		"may have an optional leading 'v'")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function
type Runner struct {
	ctx      context.Context
	Push      push.Command
	Command  *cobra.Command
	Origin    string
	Increment bool
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	if err := r.parseArgs(args); err != nil {
		return err
	}

	r.Push.Increment = r.Increment

	return nil
}

func (r *Runner) parseArgs(args []string) error {
	const op errors.Op = "cmdpublish.preRunE"

	var path string
	var ref string

	if len(args) >= 1 {
		parts := strings.Split(args[0], "@")
		if len(parts) > 2 {
			return errors.E(op, errors.InvalidParam, fmt.Errorf("at most 1 version permitted"))
		}

		path = parts[0]
		if len(parts) == 2 {
			ref = parts[1]
		}
	}

	if path == "" {
		// default to current directory
		path = "."
	}

	resolvedPath, err := argutil.ResolveSymlink(r.ctx, path)
	if err != nil {
		return err
	}

	if ref != "" {
		r.Push.Ref = ref
	}

	p, err := pkg.New(resolvedPath)
	if err != nil {
		return errors.E(op, err)
	}
	relPath, err := resolveRelPath(p.UniquePath)
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	if strings.HasPrefix(relPath, pkg.ParentDir) {
		return errors.E(op, p.UniquePath, fmt.Errorf("package path must be under current working directory"))
	}

	r.Push.Pkg = p

	if r.Origin != "" {
		t1, err1 := parse.GitParseArgs(r.ctx, []string {r.Origin, ""})
		if err1 == nil {
			r.Push.Remote = upstream.NewGitOrigin(&t1.Git)
			return nil
		}
	
		t2, err2 := parse.OciParseArgs(r.ctx, []string {r.Origin, ""})
		if err2 == nil {
			r.Push.Remote = upstream.NewOciOrigin(&t2.Oci)
			return nil
		}

		return errors.E(op, fmt.Errorf("%v %v", err1, err2))
	}

	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	const op errors.Op = "cmdpush.runE"
	if err := r.Push.Run(r.ctx); err != nil {
		return errors.E(op, r.Push.Pkg.UniquePath, err)
	}

	return nil
}

func resolveRelPath(path types.UniquePath) (string, error) {
	const op errors.Op = "cmdupdate.resolveRelPath"
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.E(op, errors.IO,
			fmt.Errorf("error looking up current working directory: %w", err))
	}

	relPath, err := filepath.Rel(cwd, path.String())
	if err != nil {
		return "", errors.E(op, errors.IO,
			fmt.Errorf("error resolving the relative path: %w", err))
	}
	return relPath, nil
}
