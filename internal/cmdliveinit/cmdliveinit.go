// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdliveinit

import (
	"context"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/attribution"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const defaultInventoryName = "inventory"

// InvExistsError defines new error when the inventory
// values have already been set on the ResourceGroup file.
type InvExistsError struct{}

func (i *InvExistsError) Error() string {
	return "inventory information already set for package"
}

// InvInKfExistsError defines new error when the inventory
// values have already been set on the Kptfile and we will warn
// the user to migrate rather than init.
type InvInKfExistsError struct{}

func (i *InvInKfExistsError) Error() string {
	return "inventory information already set within Kptfile for package"
}

func NewRunner(ctx context.Context, factory cmdutil.Factory,
	ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		ctx:       ctx,
		factory:   factory,
		ioStreams: ioStreams,
	}

	cmd := &cobra.Command{
		Use:     "init [PKG_PATH]",
		RunE:    r.runE,
		Short:   livedocs.InitShort,
		Long:    livedocs.InitShort + "\n" + livedocs.InitLong,
		Example: livedocs.InitExamples,
	}
	r.Command = cmd

	cmd.Flags().StringVar(&r.Name, "name", "", "Inventory object name")
	cmd.Flags().BoolVar(&r.Force, "force", false, "Set inventory values even if already set in Kptfile")
	cmd.Flags().BoolVar(&r.Quiet, "quiet", false, "If true, do not print output message for initialization")
	cmd.Flags().StringVar(&r.InventoryID, "inventory-id", "", "Inventory id for the package")
	cmd.Flags().StringVar(&r.RGFile, "rg-file", rgfilev1alpha1.RGFileName, "ResourceGroup object filepath")
	return r
}

func NewCommand(ctx context.Context, f cmdutil.Factory,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, f, ioStreams).Command
}

type Runner struct {
	ctx     context.Context
	Command *cobra.Command

	factory     cmdutil.Factory
	ioStreams   genericclioptions.IOStreams
	Force       bool   // Set inventory values even if already set in Kptfile
	Name        string // Inventory object name
	namespace   string // Inventory object namespace
	RGFile      string // resourcegroup object filepath
	InventoryID string // Inventory object unique identifier label
	Quiet       bool   // Output message during initialization
}

func (r *Runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdliveinit.runE"
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

	p, err := pkg.New(dir)
	if err != nil {
		return errors.E(op, err)
	}

	err = (&ConfigureInventoryInfo{
		Pkg:         p,
		Factory:     r.factory,
		Quiet:       r.Quiet,
		Name:        r.Name,
		InventoryID: r.InventoryID,
		RGFileName:  r.RGFile,
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
	Factory cmdutil.Factory
	Quiet   bool

	Name        string
	Namespace   string
	InventoryID string
	RGFileName  string

	Force bool
}

// Run updates the inventory info in the package given by the Path.
func (c *ConfigureInventoryInfo) Run(ctx context.Context) error {
	const op errors.Op = "cmdliveinit.Run"
	pr := printer.FromContextOrDie(ctx)
	var name, namespace string

	switch c.Namespace {
	case "":
		ns, err := config.FindNamespace(c.Factory.ToRawKubeConfigLoader(), c.Pkg.UniquePath.String())
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, err)
		}
		namespace = strings.TrimSpace(ns)
	default:
		namespace = c.Namespace
	}

	if !c.Quiet {
		pr.Printf("initializing ResourceGroup inventory info (namespace: %s)...", namespace)
	}

	// Autogenerate the name if it is not provided through the flag.
	if c.Name == "" {
		randomSuffix := common.RandomStr()
		name = fmt.Sprintf("%s-%s", defaultInventoryName, randomSuffix)
	} else {
		name = c.Name
	}
	// Finally, create a ResourceGroup containing the inventory information.
	err := createRGFile(c.Pkg, &kptfilev1.Inventory{
		Namespace:   namespace,
		Name:        name,
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
	const op errors.Op = "cmdliveinit.updateResourceGroup"
	// Read the resourcegroup object io io.dir
	rg, err := p.ReadRGFile(filename)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, p.UniquePath, err)
	}

	// Read the Kptfile to ensure that inventory information is not in Kptfile either.
	kf, err := p.Kptfile()
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	// Validate the inventory values don't exist in Kptfile.
	isEmpty := kptfileInventoryEmpty(kf.Inventory)
	if !isEmpty && !force {
		return errors.E(op, p.UniquePath, &InvInKfExistsError{})
	}
	// Set the Kptfile inventory to be nil since we force write to resourcegroup instead.
	kf.Inventory = nil

	// Validate the inventory values don't already exist in Resourcegroup.
	if rg != nil && !force {
		return errors.E(op, p.UniquePath, &InvExistsError{})
	}
	// Initialize new resourcegroup object, as rg should have been nil.
	rg = &rgfilev1alpha1.ResourceGroup{ResourceMeta: rgfilev1alpha1.DefaultMeta}
	// // Finally, set the inventory parameters in the ResourceGroup object and write it.
	rg.Name = inv.Name
	rg.Namespace = inv.Namespace
	if inv.InventoryID != "" {
		rg.Labels = map[string]string{rgfilev1alpha1.RGInventoryIDLabel: inv.InventoryID}
	}
	if err := writeRGFile(p.UniquePath.String(), rg, filename); err != nil {
		return errors.E(op, p.UniquePath, err)
	}

	// Rewrite Kptfile if inventory exists
	if !isEmpty {
		if err := kptfileutil.WriteFile(p.UniquePath.String(), kf); err != nil {
			return errors.E(op, p.UniquePath, err)
		}
	}

	return nil
}

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
	err = ioutil.WriteFile(filepath.Join(dir, filename), b, 0600)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}
	return nil
}

// kptfileInventoryEmpty returns true if the Inventory structure
// in the Kptfile is empty; false otherwise.
func kptfileInventoryEmpty(inv *kptfilev1.Inventory) bool {
	return inv == nil
}
