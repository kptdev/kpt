// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdmigrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/cmdliveinit"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// MigrateRunner encapsulates fields for the kpt migrate command.
type MigrateRunner struct {
	ctx       context.Context
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams
	factory   util.Factory

	dir             string
	dryRun          bool
	name            string
	rgFile          string
	force           bool
	rgInvClientFunc func(util.Factory) (inventory.InventoryClient, error)
	cmInvClientFunc func(util.Factory) (inventory.InventoryClient, error)
	cmLoader        manifestreader.ManifestLoader

	cmNotMigrated bool // flag to determine if migration from ConfigMap has occurred
}

// NewRunner returns a pointer to an initial MigrateRunner structure.
func NewRunner(ctx context.Context, factory util.Factory, cmLoader manifestreader.ManifestLoader,
	ioStreams genericclioptions.IOStreams) *MigrateRunner {

	r := &MigrateRunner{
		ctx:             ctx,
		factory:         factory,
		ioStreams:       ioStreams,
		dryRun:          false,
		cmLoader:        cmLoader,
		rgInvClientFunc: rgInvClient,
		cmInvClientFunc: cmInvClient,
		dir:             "",
	}
	cmd := &cobra.Command{
		Use:     "migrate [DIR | -]",
		Short:   livedocs.MigrateShort,
		Long:    livedocs.MigrateShort + "\n" + livedocs.MigrateLong,
		Example: livedocs.MigrateExamples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// default to current working directory
				args = append(args, ".")
			}
			fmt.Fprint(ioStreams.Out, "inventory migration...\n")
			if err := r.Run(ioStreams.In, args); err != nil {
				fmt.Fprint(ioStreams.Out, "failed\n")
				fmt.Fprint(ioStreams.Out, "inventory migration...failed\n")
				return err
			}
			fmt.Fprint(ioStreams.Out, "inventory migration...success\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&r.name, "name", "", "Inventory object name")
	cmd.Flags().BoolVar(&r.force, "force", false, "Set inventory values even if already set in Kptfile")
	cmd.Flags().BoolVar(&r.dryRun, "dry-run", false, "Do not actually migrate, but show steps")
	cmd.Flags().StringVar(&r.rgFile, "rg-file", rgfilev1alpha1.RGFileName, "ResourceGroup object filepath")

	r.Command = cmd
	return r
}

// NewCommand returns the cobra command for the migrate command.
func NewCommand(ctx context.Context, f util.Factory, cmLoader manifestreader.ManifestLoader,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, f, cmLoader, ioStreams).Command
}

// Run executes the migration from the ConfigMap based inventory to the ResourceGroup
// based inventory.
func (mr *MigrateRunner) Run(reader io.Reader, args []string) error {
	// Validate the number of arguments.
	if len(args) > 1 {
		return fmt.Errorf("too many arguments; migrate requires one directory argument (or stdin)")
	}
	// Validate argument is a directory.
	if len(args) == 1 {
		var err error
		mr.dir, err = config.NormalizeDir(args[0])
		if err != nil {
			return err
		}
	}
	// Store the stdin bytes if necessary so they can be used twice.
	var stdinBytes []byte
	var err error
	if len(args) == 0 {
		stdinBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		if len(stdinBytes) == 0 {
			return fmt.Errorf("no arguments means stdin has data; missing bytes on stdin")
		}
	}

	// Apply the ResourceGroup CRD to the cluster, ignoring if it already exists.
	if err := mr.applyCRD(); err != nil {
		return err
	}

	// Check if we need to migrate from ConfigMap.
	if err := mr.migrateCM(stdinBytes, args); err != nil {
		return err
	}
	// Migrate from Kptfile.
	if mr.cmNotMigrated {
		return mr.migrateKptfile(args)
	}

	return nil
}

func (mr *MigrateRunner) migrateKptfile(args []string) error {
	klog.V(4).Infoln("attempting to migrate from Kptfile inventory")
	fmt.Fprint(mr.ioStreams.Out, "  reading existing Kptfile...")
	if !mr.dryRun {
		p, err := pkg.New(args[0])
		if err != nil {
			return err
		}
		kf, err := p.Kptfile()
		if err != nil {
			return err
		}

		if _, err := kptfileutil.ValidateInventory(kf.Inventory); err != nil {
			// Invalid Kptfile.
			return err
		}

		err = (&cmdliveinit.ConfigureInventoryInfo{
			Pkg:         p,
			Factory:     mr.factory,
			Quiet:       true,
			Name:        kf.Inventory.Name,
			Namespace:   kf.Inventory.Namespace,
			InventoryID: kf.Inventory.InventoryID,
			RGFileName:  mr.rgFile,
			Force:       mr.force,
		}).Run(mr.ctx)

		if err != nil {
			var invExistsError *cmdliveinit.InvExistsError
			if errors.As(err, &invExistsError) {
				fmt.Fprint(mr.ioStreams.Out, "values already exist...")
			} else {
				return err
			}
		}

		// Rewrite Kptfile without inventory info.
		kf.Inventory = nil
		err = kptfileutil.WriteFile(p.UniquePath.String(), kf)
		if err != nil {
			return err
		}
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// migrateCM migrates from ConfigMap to resourcegroup object.
func (mr *MigrateRunner) migrateCM(stdinBytes []byte, args []string) error {
	// Create the inventory clients for reading inventories based on RG and
	// ConfigMap.
	rgInvClient, err := mr.rgInvClientFunc(mr.factory)
	if err != nil {
		return err
	}
	cmInvClient, err := mr.cmInvClientFunc(mr.factory)
	if err != nil {
		return err
	}
	// Retrieve the current ConfigMap inventory objects.
	cmInvObj, err := mr.retrieveConfigMapInv(bytes.NewReader(stdinBytes), args)
	if err != nil {
		if _, ok := err.(inventory.NoInventoryObjError); ok {
			// No ConfigMap inventory means the migration has already run before.
			klog.V(4).Infoln("swallowing no ConfigMap inventory error")
			mr.cmNotMigrated = true
			return nil
		}
		klog.V(4).Infof("error retrieving ConfigMap inventory object: %s", err)
		return err
	}
	cmInventoryID := cmInvObj.ID()
	klog.V(4).Infof("previous inventoryID: %s", cmInventoryID)
	// Create ResourceGroup object file locallly (e.g. namespace, name, id).
	if err := mr.createRGfile(mr.ctx, args, cmInventoryID); err != nil {
		return err
	}
	cmObjs, err := mr.retrieveInvObjs(cmInvClient, cmInvObj)
	if err != nil {
		return err
	}
	if len(cmObjs) > 0 {
		// Migrate the ConfigMap inventory objects to a ResourceGroup custom resource.
		if err = mr.migrateObjs(rgInvClient, cmObjs, bytes.NewReader(stdinBytes), args); err != nil {
			return err
		}
		// Delete the old ConfigMap inventory object.
		if err = mr.deleteConfigMapInv(cmInvClient, cmInvObj); err != nil {
			return err
		}
	}
	return mr.deleteConfigMapFile()
}

// applyCRD applies the ResourceGroup custom resource definition, returning an
// error if one occurred. Ignores "AlreadyExists" error. Uses the definition
// stored in the "rgCrd" variable.
func (mr *MigrateRunner) applyCRD() error {
	fmt.Fprint(mr.ioStreams.Out, "  ensuring ResourceGroup CRD exists in cluster...")
	// Simply return early if this is a dry run
	if mr.dryRun {
		fmt.Fprintln(mr.ioStreams.Out, "success")
		return nil
	}
	// Install the ResourceGroup CRD to the cluster.
	err := live.InstallResourceGroupCRD(mr.factory)
	if err == nil {
		fmt.Fprintln(mr.ioStreams.Out, "success")
	} else {
		fmt.Fprintln(mr.ioStreams.Out, "failed")
	}
	return err
}

// createRGfile writes the inventory information into the resourcegroup object.
func (mr *MigrateRunner) createRGfile(ctx context.Context, args []string, prevID string) error {
	fmt.Fprint(mr.ioStreams.Out, "  creating ResourceGroup object file...")
	if !mr.dryRun {
		p, err := pkg.New(args[0])
		if err != nil {
			return err
		}
		err = (&cmdliveinit.ConfigureInventoryInfo{
			Pkg:         p,
			Factory:     mr.factory,
			Quiet:       true,
			InventoryID: prevID,
			RGFileName:  mr.rgFile,
			Force:       mr.force,
		}).Run(ctx)

		if err != nil {
			var invExistsError *cmdliveinit.InvExistsError
			if errors.As(err, &invExistsError) {
				fmt.Fprint(mr.ioStreams.Out, "values already exist...")
			} else {
				return err
			}
		}
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// retrieveConfigMapInv retrieves the ConfigMap inventory object or
// an error if one occurred.
func (mr *MigrateRunner) retrieveConfigMapInv(reader io.Reader, args []string) (inventory.InventoryInfo, error) {
	fmt.Fprint(mr.ioStreams.Out, "  retrieve the current ConfigMap inventory...")
	cmReader, err := mr.cmLoader.ManifestReader(reader, args[0])
	if err != nil {
		return nil, err
	}
	objs, err := cmReader.Read()
	if err != nil {
		return nil, err
	}
	cmInv, _, err := mr.cmLoader.InventoryInfo(objs)
	if err != nil {
		// No ConfigMap inventory means the migration has already run before.
		if _, ok := err.(inventory.NoInventoryObjError); ok { //nolint
			fmt.Fprintln(mr.ioStreams.Out, "no ConfigMap inventory...completed")
		}
	} else {
		fmt.Fprintf(mr.ioStreams.Out, "success (inventory-id: %s)\n", cmInv.ID())
	}
	return cmInv, err
}

// retrieveInvObjs returns the object references from the passed
// inventory object by querying the inventory object in the cluster,
// or an error if one occurred.
func (mr *MigrateRunner) retrieveInvObjs(cmInvClient inventory.InventoryClient,
	invObj inventory.InventoryInfo) ([]object.ObjMetadata, error) {
	fmt.Fprint(mr.ioStreams.Out, "  retrieve ConfigMap inventory objs...")
	cmObjs, err := cmInvClient.GetClusterObjs(invObj, mr.dryRunStrategy())
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(mr.ioStreams.Out, "success (%d inventory objects)\n", len(cmObjs))
	return cmObjs, nil
}

// migrateObjs stores the passed objects in the ResourceGroup inventory
// object and applies the inventory object to the cluster. Returns
// an error if one occurred.
func (mr *MigrateRunner) migrateObjs(rgInvClient inventory.InventoryClient,
	cmObjs []object.ObjMetadata, reader io.Reader, args []string) error {
	if err := validateParams(reader, args); err != nil {
		return err
	}
	fmt.Fprint(mr.ioStreams.Out, "  migrate inventory to ResourceGroup...")
	if len(cmObjs) == 0 {
		fmt.Fprint(mr.ioStreams.Out, "no inventory objects found\n")
		return nil
	}
	if mr.dryRun {
		fmt.Fprintln(mr.ioStreams.Out, "success")
		return nil
	}

	path := args[0]
	var err error
	if args[0] != "-" {
		path, err = argutil.ResolveSymlink(mr.ctx, path)
		if err != nil {
			return err
		}
	}

	_, inv, err := live.Load(mr.factory, path, reader)
	if err != nil {
		return err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return err
	}

	_, err = rgInvClient.Merge(invInfo, cmObjs, mr.dryRunStrategy())
	if err != nil {
		return err
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// deleteConfigMapInv removes the passed inventory object from the
// cluster. Returns an error if one occurred.
func (mr *MigrateRunner) deleteConfigMapInv(cmInvClient inventory.InventoryClient,
	invObj inventory.InventoryInfo) error {
	fmt.Fprint(mr.ioStreams.Out, "  deleting old ConfigMap inventory object...")
	if err := cmInvClient.DeleteInventoryObj(invObj, mr.dryRunStrategy()); err != nil {
		return err
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// deleteConfigMapFile deletes the ConfigMap template inventory file. This file
// is usually named "inventory-template.yaml". This operation only happens if
// the input was a directory argument (otherwise there is nothing to delete).
// Returns an error if one occurs while deleting the file. Does NOT return an
// error if the inventory template file is missing.
func (mr *MigrateRunner) deleteConfigMapFile() error {
	// Only delete the file if the input was a directory argument.
	if len(mr.dir) > 0 {
		cmFilename, _, err := common.ExpandDir(mr.dir)
		if err != nil {
			return err
		}
		if len(cmFilename) > 0 {
			fmt.Fprintf(mr.ioStreams.Out, "deleting inventory template file: %s...", cmFilename)
			if !mr.dryRun {
				err = os.Remove(cmFilename)
				if err != nil {
					fmt.Fprint(mr.ioStreams.Out, "failed\n")
					return err
				}
			}
			fmt.Fprint(mr.ioStreams.Out, "success\n")
		}
	}
	return nil
}

// dryRunStrategy returns the strategy to use based on user config
func (mr *MigrateRunner) dryRunStrategy() common.DryRunStrategy {
	if mr.dryRun {
		return common.DryRunClient
	}
	return common.DryRunNone
}

// findResourceGroupInv returns the ResourceGroup inventory object from the
// passed slice of objects, or nil and and error if there is a problem.
func findResourceGroupInv(objs []*unstructured.Unstructured) (*unstructured.Unstructured, error) {
	for _, obj := range objs {
		isInv, err := live.IsResourceGroupInventory(obj)
		if err != nil {
			return nil, err
		}
		if isInv {
			return obj, nil
		}
	}
	return nil, fmt.Errorf("resource group inventory object not found")
}

// validateParams validates input parameters and returns error if any
func validateParams(reader io.Reader, args []string) error {
	if reader == nil && len(args) == 0 {
		return fmt.Errorf("unable to build ManifestReader without both reader or args")
	}
	if len(args) > 1 {
		return fmt.Errorf("expected one directory argument allowed; got (%s)", args)
	}
	return nil
}

func rgInvClient(factory util.Factory) (inventory.InventoryClient, error) {
	return inventory.NewInventoryClient(factory, live.WrapInventoryObj, live.InvToUnstructuredFunc)
}

func cmInvClient(factory util.Factory) (inventory.InventoryClient, error) {
	return inventory.NewInventoryClient(factory, inventory.WrapInventoryObj, inventory.InvInfoToConfigMap)
}
