// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdliveinit

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
)

const defaultInventoryName = "inventory"

// InvExistsError defines new error when the inventory
// values have already been set on the Kptfile.
type InvExistsError struct{}

func (i *InvExistsError) Error() string {
	return "inventory information already set for package"
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
	InventoryID string

	Force bool
}

// Run updates the inventory info in the package given by the Path.
func (c *ConfigureInventoryInfo) Run(ctx context.Context) error {
	const op errors.Op = "cmdliveinit.Run"
	pr := printer.FromContextOrDie(ctx)

	var name, namespace, inventoryID string

	ns, err := config.FindNamespace(c.Factory.ToRawKubeConfigLoader(), c.Pkg.UniquePath.String())
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	namespace = strings.TrimSpace(ns)
	if !c.Quiet {
		pr.Printf("initializing Kptfile inventory info (namespace: %s)...", namespace)
	}

	// Autogenerate the name if it is not provided through the flag.
	if c.Name == "" {
		randomSuffix := common.RandomStr()
		name = fmt.Sprintf("%s-%s", defaultInventoryName, randomSuffix)
	} else {
		name = c.Name
	}
	// Generate the inventory id if one is not specified through a flag.
	if c.InventoryID == "" {
		id, err := generateID(namespace, name, time.Now())
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, err)
		}
		inventoryID = id
	} else {
		inventoryID = c.InventoryID
	}
	// Finally, update these values in the Inventory section of the Kptfile.
	err = updateKptfile(c.Pkg, &kptfilev1alpha2.Inventory{
		Namespace:   namespace,
		Name:        name,
		InventoryID: inventoryID,
	}, c.Force)
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
	return nil
}

// Run fills in the inventory object values into the Kptfile.
func updateKptfile(p *pkg.Pkg, inv *kptfilev1alpha2.Inventory, force bool) error {
	const op errors.Op = "cmdliveinit.updateKptfile"
	// Read the Kptfile io io.dir
	kf, err := p.Kptfile()
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	// Validate the inventory values don't already exist
	isEmpty := kptfileInventoryEmpty(kf.Inventory)
	if !isEmpty && !force {
		return errors.E(op, p.UniquePath, &InvExistsError{})
	}
	// Finally, set the inventory parameters in the Kptfile and write it.
	kf.Inventory = inv
	if err := kptfileutil.WriteFile(p.UniquePath.String(), *kf); err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	return nil
}

// generateID returns the string which is a SHA1 hash of the passed namespace
// and name, with the unix timestamp string concatenated. Returns an error
// if either the namespace or name are empty.
func generateID(namespace string, name string, t time.Time) (string, error) {
	const op errors.Op = "cmdliveinit.generateID"
	hashStr, err := generateHash(namespace, name)
	if err != nil {
		return "", errors.E(op, err)
	}
	timeStr := strconv.FormatInt(t.UTC().UnixNano(), 10)
	return fmt.Sprintf("%s-%s", hashStr, timeStr), nil
}

// generateHash returns the SHA1 hash of the concatenated "namespace:name" string,
// or an error if either namespace or name is empty.
func generateHash(namespace string, name string) (string, error) {
	const op errors.Op = "cmdliveinit.generateHash"
	if len(namespace) == 0 || len(name) == 0 {
		return "", errors.E(op,
			fmt.Errorf("can not generate hash with empty namespace or name"))
	}
	str := fmt.Sprintf("%s:%s", namespace, name)
	h := sha1.New()
	if _, err := h.Write([]byte(str)); err != nil {
		return "", errors.E(op, err)
	}
	return fmt.Sprintf("%x", (h.Sum(nil))), nil
}

// kptfileInventoryEmpty returns true if the Inventory structure
// in the Kptfile is empty; false otherwise.
func kptfileInventoryEmpty(inv *kptfilev1alpha2.Inventory) bool {
	return inv == nil
}
