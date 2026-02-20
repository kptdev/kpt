// Copyright 2020,2026 The kpt Authors
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

package init

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kptdev/kpt/internal/docs/generated/livedocs"
	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/internal/types"
	"github.com/kptdev/kpt/internal/util/attribution"
	"github.com/kptdev/kpt/internal/util/pathutil"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/lib/errors"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// errNameRequired is returned when --name is not provided or blank.
var errNameRequired = fmt.Errorf(
	"--name is required: provide a stable deployment name " +
		"(e.g. --name=my-app-staging) that remains consistent across re-initializations")

// InvExistsError defines new error when the inventory
// values have already been set on the Kptfile.
type InvExistsError struct{}

func (i *InvExistsError) Error() string {
	return "inventory information already set for package"
}

// InvInRGExistsError defines new error when the inventory
// values have already been set on the ResourceGroup file and we will warn
// the user to migrate rather than init. This is part of kpt live STDIN work.
type InvInRGExistsError struct{}

func (i *InvInRGExistsError) Error() string {
	return "inventory information already set for package"
}

// InvInKfExistsError defines new error when the inventory
// values have already been set on the Kptfile and we will warn
// the user to migrate rather than init. This is part of kpt live STDIN work.
type InvInKfExistsError struct{}

func (i *InvInKfExistsError) Error() string {
	return "inventory information already set within Kptfile for package"
}

func NewRunner(ctx context.Context, factory k8scmdutil.Factory,
	ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		ctx:       ctx,
		factory:   factory,
		ioStreams: ioStreams,
	}

	cmd := &cobra.Command{
		Use:     "init [PKG_PATH]",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Short:   livedocs.InitShort,
		Long:    livedocs.InitShort + "\n" + livedocs.InitLong,
		Example: livedocs.InitExamples,
	}
	r.Command = cmd

	cmd.Flags().StringVar(&r.Name, "name", "", "Stable deployment name for this package (required)")
	cmd.Flags().BoolVar(&r.Force, "force", false, "Set inventory values even if already set in Kptfile or ResourceGroup file")
	cmd.Flags().BoolVar(&r.Quiet, "quiet", false, "If true, do not print output message for initialization")
	cmd.Flags().StringVar(&r.InventoryID, "inventory-id", "", "Override the auto-derived inventory ID (advanced)")
	_ = cmd.Flags().MarkHidden("inventory-id")
	cmd.Flags().StringVar(&r.RGFileName, "rg-file", rgfilev1alpha1.RGFileName, "Filename for the ResourceGroup CR")
	return r
}

func NewCommand(ctx context.Context, f k8scmdutil.Factory,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, f, ioStreams).Command
}

type Runner struct {
	ctx     context.Context
	Command *cobra.Command

	factory     k8scmdutil.Factory
	ioStreams   genericclioptions.IOStreams
	Force       bool   // Set inventory values even if already set in Kptfile
	Name        string // Inventory object name
	RGFileName  string // resourcegroup object filename
	InventoryID string // Inventory object unique identifier label
	Quiet       bool   // Output message during initialization
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	dir := filepath.Dir(filepath.Clean(r.RGFileName))
	if dir != "." {
		return fmt.Errorf("rg-file must be a valid filename")
	}
	return nil
}

func (r *Runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdliveinit.runE"
	name, err := validateName(r.Name)
	if err != nil {
		return errors.E(op, err)
	}
	r.Name = name
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return errors.E(op, err)
		}
		args = append(args, cwd)
	}

	dir, err := config.NormalizeDir(args[0])
	if err != nil {
		return errors.E(op, err)
	}

	absPath, _, err := pathutil.ResolveAbsAndRelPaths(dir)
	if err != nil {
		return err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absPath)
	if err != nil {
		return errors.E(op, err)
	}

	err = (&ConfigureInventoryInfo{
		Pkg:         p,
		Factory:     r.factory,
		Quiet:       r.Quiet,
		Name:        r.Name,
		InventoryID: r.InventoryID,
		RGFileName:  r.RGFileName,
		Force:       r.Force,
	}).Run(r.ctx)
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	return nil
}

// ConfigureInventoryInfo contains the functionality for adding and updating
// the inventory information in the Kptfile.
type ConfigureInventoryInfo struct {
	Pkg     *pkg.Pkg
	Factory k8scmdutil.Factory
	Quiet   bool

	Name        string
	InventoryID string
	RGFileName  string

	Force bool
}

// Run updates the inventory info in the package given by the Path.
func (c *ConfigureInventoryInfo) Run(ctx context.Context) error {
	const op errors.Op = "cmdliveinit.Run"
	pr := printer.FromContextOrDie(ctx)

	namespace, err := config.FindNamespace(c.Factory.ToRawKubeConfigLoader(), c.Pkg.UniquePath.String())
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	namespace = strings.TrimSpace(namespace)
	if !c.Quiet {
		pr.Printf("initializing %q data (namespace: %s)...", c.RGFileName, namespace)
	}

	// Internal callers (e.g. migrate) may pass empty Name with an explicit
	// InventoryID; derive a stable name from the package directory.
	if c.Name == "" {
		if c.InventoryID != "" {
			dirName := filepath.Base(c.Pkg.UniquePath.String())
			if errs := validation.IsDNS1123Label(dirName); len(errs) > 0 {
				return errors.E(op, c.Pkg.UniquePath,
					fmt.Errorf("directory name %q is not a valid Kubernetes resource name and --name was not provided: %s",
						dirName, strings.Join(errs, "; ")))
			}
			c.Name = dirName
		} else {
			return errors.E(op, c.Pkg.UniquePath, errNameRequired)
		}
	}

	// Derive inventory ID from namespace+name unless explicitly overridden.
	if c.InventoryID == "" {
		c.InventoryID, err = generateHash(namespace, c.Name)
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, err)
		}
	}

	// Finally, create a ResourceGroup containing the inventory information.
	err = createRGFile(c.Pkg, &kptfilev1.Inventory{
		Namespace:   namespace,
		Name:        c.Name,
		InventoryID: c.InventoryID,
	}, c.RGFileName, c.Force)
	if !c.Quiet {
		if err == nil {
			pr.Printf("success\n")
		} else {
			pr.Printf("failed\n")
		}
	}
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	// add metrics annotation to package resources to track the usage as the resources
	// will be applied using kpt live group
	at := attribution.Attributor{PackagePaths: []string{c.Pkg.UniquePath.String()}, CmdGroup: "live"}
	at.Process()
	return nil
}

// createRGFile fills in the inventory object values into the resourcegroup object and writes to file storage.
func createRGFile(p *pkg.Pkg, inv *kptfilev1.Inventory, filename string, force bool) error {
	const op errors.Op = "cmdliveinit.createRGFile"
	// Read the resourcegroup object io io.dir
	rg, err := p.ReadRGFile(filename)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, p.UniquePath, err)
	}

	// Read the Kptfile to ensure that inventory information is not in Kptfile either.
	// Ignore error if Kptfile not found as we now support live init without a Kptfile since
	// inventory information is stored in a ResourceGroup object.
	kf, err := p.Kptfile()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.E(op, p.UniquePath, err)
	}
	// Validate the inventory values don't exist in Kptfile.
	isEmpty := true
	if kf != nil {
		isEmpty = kptfileInventoryEmpty(kf.Inventory)
		if !isEmpty && !force {
			return errors.E(op, p.UniquePath, &InvInKfExistsError{})
		}

		// Set the Kptfile inventory to be nil if we force write to resourcegroup instead.
		kf.Inventory = nil
	}

	// Validate the inventory values don't already exist in Resourcegroup.
	if rg != nil && !force {
		return errors.E(op, p.UniquePath, &InvInRGExistsError{})
	}
	// Initialize new ResourceGroup and populate inventory fields.
	rg = &rgfilev1alpha1.ResourceGroup{ResourceMeta: rgfilev1alpha1.DefaultMeta}
	rg.Name = inv.Name
	rg.Namespace = inv.Namespace
	rg.Labels = map[string]string{rgfilev1alpha1.RGInventoryIDLabel: inv.InventoryID}
	if err := writeRGFile(p.UniquePath.String(), rg, filename); err != nil {
		return errors.E(op, p.UniquePath, err)
	}

	// Rewrite Kptfile without inventory existing Kptfile contains inventory info. This
	// is required when a user appends the force flag.
	if !isEmpty {
		if err := kptfileutil.WriteFile(p.UniquePath.String(), kf); err != nil {
			return errors.E(op, p.UniquePath, err)
		}
	}

	return nil
}

// writeRGFile writes a ResourceGroup inventory to local disk.
func writeRGFile(dir string, rg *rgfilev1alpha1.ResourceGroup, filename string) error {
	const op errors.Op = "cmdliveinit.writeRGFile"
	b, err := yaml.MarshalWithOptions(rg, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, filename)); err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}

	// fyi: perm is ignored if the file already exists
	err = os.WriteFile(filepath.Join(dir, filename), b, 0600)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}
	return nil
}

// generateHash returns a deterministic 40-char hex inventory ID from namespace
// and name using SHA-1. Both fields are length-prefixed to prevent collisions
// (e.g. ns="ab", name="cd" vs ns="a", name="bcd").
func generateHash(namespace, name string) (string, error) {
	if namespace == "" || name == "" {
		return "", fmt.Errorf("cannot generate inventory ID: namespace and name must be non-empty")
	}
	h := sha1.New()
	if _, err := fmt.Fprintf(h, "%d:%s:%d:%s", len(namespace), namespace, len(name), name); err != nil {
		return "", fmt.Errorf("failed to write hash input: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// validateName rejects empty, whitespace-only, and non-RFC-1123 names.
// Returns the trimmed name on success.
func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errNameRequired
	}
	if errs := validation.IsDNS1123Label(trimmed); len(errs) > 0 {
		return "", fmt.Errorf("--name %q is not a valid Kubernetes resource name: %s",
			trimmed, strings.Join(errs, "; "))
	}
	return trimmed, nil
}

// kptfileInventoryEmpty returns true if the Inventory structure
// in the Kptfile is empty; false otherwise.
func kptfileInventoryEmpty(inv *kptfilev1.Inventory) bool {
	return inv == nil
}
