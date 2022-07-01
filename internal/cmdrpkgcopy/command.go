// Copyright 2022 Google LLC
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

package cmdrpkgcopy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgcopy"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	r.Command = &cobra.Command{
		Use:     "copy SOURCE_PACKAGE NAME",
		Aliases: []string{"edit"},
		Short:   rpkgdocs.CopyShort,
		Long:    rpkgdocs.CopyShort + "\n" + rpkgdocs.CopyLong,
		Example: rpkgdocs.CopyExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command.Flags().StringVar(&r.revision, "revision", "", "Revision of the copied package.")
	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	copy porchapi.PackageEditTaskSpec

	revision string // Target package revision
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client

	if len(args) < 1 {
		return errors.E(op, fmt.Errorf("SOURCE_PACKAGE is a required positional argument"))
	}
	if len(args) > 1 {
		return errors.E(op, fmt.Errorf("too many arguments; SOURCE_PACKAGE is the only accepted positional arguments"))
	}

	r.copy.Source = &porchapi.PackageRevisionRef{
		Name: args[0],
	}
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	revisionSpec, err := r.getPackageRevisionSpec()
	if err != nil {
		return errors.E(op, err)
	}

	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: *r.cfg.Namespace,
		},
		Spec: *revisionSpec,
	}
	if err := r.client.Create(r.ctx, pr); err != nil {
		return errors.E(op, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s created\n", pr.Name)
	return nil
}

func (r *runner) getPackageRevisionSpec() (*porchapi.PackageRevisionSpec, error) {
	packageRevision := porchapi.PackageRevision{}
	err := r.client.Get(r.ctx, types.NamespacedName{
		Name:      r.copy.Source.Name,
		Namespace: *r.cfg.Namespace,
	}, &packageRevision)
	if err != nil {
		return nil, err
	}

	if r.revision == "" {
		var err error
		r.revision, err = r.defaultPackageRevision(
			packageRevision.Spec.PackageName,
			packageRevision.Spec.RepositoryName,
		)
		if err != nil {
			return nil, err
		}
	}

	spec := &porchapi.PackageRevisionSpec{
		PackageName:    packageRevision.Spec.PackageName,
		Revision:       r.revision,
		RepositoryName: packageRevision.Spec.RepositoryName,
	}
	spec.Tasks = packageRevision.Spec.Tasks
	return spec, nil
}

// defaultPackageRevision attempts to return a default package revision number
// of "latest + 1" given a package name, repository, and namespace.
func (r *runner) defaultPackageRevision(packageName, repository string) (string, error) {
	// get all package revisions
	packageRevisionList := porchapi.PackageRevisionList{}
	err := r.client.List(r.ctx, &packageRevisionList, client.InNamespace(*r.cfg.Namespace))
	if err != nil {
		return "", err
	}

	var latestRevision string
	allRevisions := make(map[string]bool) // this is a map for quick access

	for _, rev := range packageRevisionList.Items {
		if packageName != rev.Spec.PackageName ||
			repository != rev.Spec.RepositoryName {
			continue
		}

		if latest, ok := rev.Labels[porchapi.LatestPackageRevisionKey]; ok {
			if latest == porchapi.LatestPackageRevisionValue {
				latestRevision = rev.Spec.Revision
			}
		}
		allRevisions[rev.Spec.Revision] = true
	}
	if latestRevision == "" {
		return "", fmt.Errorf("no published packages exist; explicit --revision flag is required")
	}

	next, err := nextRevisionNumber(latestRevision)
	if err != nil {
		return "", err
	}
	if _, ok := allRevisions[next]; ok {
		return "", fmt.Errorf("default revision %q already exists; explicit --revision flag is required", next)
	}
	return next, err
}

func nextRevisionNumber(latestRevision string) (string, error) {
	if !semver.IsValid(latestRevision) {
		return "", fmt.Errorf("invalid revision format %s; explicit --revision flag is required", latestRevision)
	}

	parts := strings.Split(latestRevision[1:], ".")
	lastIndex := len(parts) - 1
	i, err := strconv.Atoi(parts[lastIndex])
	if err != nil {
		return "", err
	}
	i++
	parts[lastIndex] = strconv.Itoa(i)
	next := "v" + strings.Join(parts, ".")
	return next, nil
}
