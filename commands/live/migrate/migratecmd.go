// Copyright 2020 The kpt Authors
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

package migrate

import (
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	initialization "github.com/GoogleContainerTools/kpt/commands/live/init"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
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
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// MigrateRunner encapsulates fields for the kpt migrate command.
type Runner struct {
	ctx       context.Context
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams
	factory   util.Factory

	dir             string
	dryRun          bool
	name            string
	rgFile          string
	force           bool
	rgInvClientFunc func(util.Factory) (inventory.Client, error)
	cmInvClientFunc func(util.Factory) (inventory.Client, error)
	cmLoader        manifestreader.ManifestLoader
	cmNotMigrated   bool // flag to determine if migration from ConfigMap has occurred
}

// NewRunner returns a pointer to an initial MigrateRunner structure.
func NewRunner(
	ctx context.Context,
	factory util.Factory,
	cmLoader manifestreader.ManifestLoader,
	ioStreams genericclioptions.IOStreams,
) *Runner {
	r := &Runner{
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
	cmd.Flags().StringVar(&r.rgFile, "rg-file", rgfilev1alpha1.RGFileName, "The file path to the ResourceGroup object.")

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
func (mr *Runner) Run(reader io.Reader, args []string) error {
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
		stdinBytes, err = io.ReadAll(reader)
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

	// Check if we need to migrate from ConfigMap to ResourceGroup.
	if err := mr.migrateCMToRG(stdinBytes, args); err != nil {
		return err
	}

	// Migrate from Kptfile instead.
	if mr.cmNotMigrated {
		return mr.migrateKptfileToRG(args)
	}

	return nil
}

// applyCRD applies the ResourceGroup custom resource definition, returning an
// error if one occurred. Ignores "AlreadyExists" error. Uses the definition
// stored in the "rgCrd" variable.
func (mr *Runner) applyCRD() error {
	fmt.Fprint(mr.ioStreams.Out, "  ensuring ResourceGroup CRD exists in cluster...")
	// Simply return early if this is a dry run
	if mr.dryRun {
		fmt.Fprintln(mr.ioStreams.Out, "success")
		return nil
	}
	// Install the ResourceGroup CRD to the cluster.

	err := (&live.ResourceGroupInstaller{
		Factory: mr.factory,
	}).InstallRG(mr.ctx)
	if err == nil {
		fmt.Fprintln(mr.ioStreams.Out, "success")
	} else {
		fmt.Fprintln(mr.ioStreams.Out, "failed")
	}
	return err
}

// retrieveConfigMapInv retrieves the ConfigMap inventory object or
// an error if one occurred.
func (mr *Runner) retrieveConfigMapInv(reader io.Reader, args []string) (inventory.Info, error) {
	fmt.Fprint(mr.ioStreams.Out, "  retrieve the current ConfigMap inventory...")
	cmReader, err := mr.cmLoader.ManifestReader(reader, args[0])
	if err != nil {
		return nil, err
	}
	objs, err := cmReader.Read()
	if err != nil {
		return nil, err
	}
	cmInvObj, _, err := inventory.SplitUnstructureds(objs)
	if err != nil {
		fmt.Fprintln(mr.ioStreams.Out, "no ConfigMap inventory...completed")
		return nil, err
	}

	// cli-utils treats any resource that contains the inventory-id label as an inventory object. We should
	// ignore any inventories that are stored as ResourceGroup resources since they do not need migration.
	if cmInvObj.GetKind() == rgfilev1alpha1.ResourceGroupGVK().Kind {
		// No ConfigMap inventory means the migration has already run before.
		fmt.Fprintln(mr.ioStreams.Out, "no ConfigMap inventory...completed")
		return nil, &inventory.NoInventoryObjError{}
	}

	cmInv := inventory.WrapInventoryInfoObj(cmInvObj)
	fmt.Fprintf(mr.ioStreams.Out, "success (inventory-id: %s)\n", cmInv.ID())
	return cmInv, nil
}

// retrieveInvObjs returns the object references from the passed
// inventory object by querying the inventory object in the cluster,
// or an error if one occurred.
func (mr *Runner) retrieveInvObjs(cmInvClient inventory.Client,
	invObj inventory.Info) ([]object.ObjMetadata, error) {
	fmt.Fprint(mr.ioStreams.Out, "  retrieve ConfigMap inventory objs...")
	cmObjs, err := cmInvClient.GetClusterObjs(invObj)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(mr.ioStreams.Out, "success (%d inventory objects)\n", len(cmObjs))
	return cmObjs, nil
}

// migrateObjs stores the passed objects in the ResourceGroup inventory
// object and applies the inventory object to the cluster. Returns
// an error if one occurred.
func (mr *Runner) migrateObjs(rgInvClient inventory.Client,
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
func (mr *Runner) deleteConfigMapInv(cmInvClient inventory.Client,
	invObj inventory.Info) error {
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
func (mr *Runner) deleteConfigMapFile() error {
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
func (mr *Runner) dryRunStrategy() common.DryRunStrategy {
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

func rgInvClient(factory util.Factory) (inventory.Client, error) {
	return inventory.NewClient(factory, live.WrapInventoryObj, live.InvToUnstructuredFunc, inventory.StatusPolicyAll, live.ResourceGroupGVK)
}

func cmInvClient(factory util.Factory) (inventory.Client, error) {
	return inventory.NewClient(factory, inventory.WrapInventoryObj, inventory.InvInfoToConfigMap, inventory.StatusPolicyAll, live.ResourceGroupGVK)
}

// migrateKptfileToRG extracts inventory information from a package's Kptfile
// into an external resourcegroup file.
func (mr *Runner) migrateKptfileToRG(args []string) error {
	const op errors.Op = "migratecmd.migrateKptfileToRG"
	klog.V(4).Infoln("attempting to migrate from Kptfile inventory")
	fmt.Fprint(mr.ioStreams.Out, "  reading existing Kptfile...")
	if !mr.dryRun {
		dir, _, err := pathutil.ResolveAbsAndRelPaths(args[0])
		if err != nil {
			return err
		}
		p, err := pkg.New(filesys.FileSystemOrOnDisk{}, dir)
		if err != nil {
			return err
		}
		kf, err := p.Kptfile()
		if err != nil {
			return err
		}

		if _, err := kptfileutil.ValidateInventory(kf.Inventory); err != nil {
			// Kptfile does not contain inventory: migration is not needed.
			return nil
		}

		// Make sure resourcegroup file does not exist.
		_, rgFileErr := os.Stat(filepath.Join(dir, mr.rgFile))
		switch {
		case rgFileErr == nil:
			return errors.E(op, errors.IO, types.UniquePath(dir), "the resourcegroup file already exists and inventory information cannot be migrated")
		case err != nil && !goerrors.Is(err, os.ErrNotExist):
			return errors.E(op, errors.IO, types.UniquePath(dir), err)
		}

		err = (&initialization.ConfigureInventoryInfo{
			Pkg:         p,
			Factory:     mr.factory,
			Quiet:       true,
			Name:        kf.Inventory.Name,
			InventoryID: kf.Inventory.InventoryID,
			RGFileName:  mr.rgFile,
			Force:       true,
		}).Run(mr.ctx)

		if err != nil {
			return err
		}
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// migrateCMToRG migrates from ConfigMap to resourcegroup object.
func (mr *Runner) migrateCMToRG(stdinBytes []byte, args []string) error {
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
		if _, ok := err.(*inventory.NoInventoryObjError); ok {
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

// createRGfile writes the inventory information into the resourcegroup object.
func (mr *Runner) createRGfile(ctx context.Context, args []string, prevID string) error {
	fmt.Fprint(mr.ioStreams.Out, "  creating ResourceGroup object file...")
	if !mr.dryRun {
		dir, _, err := pathutil.ResolveAbsAndRelPaths(args[0])
		if err != nil {
			return err
		}
		p, err := pkg.New(filesys.FileSystemOrOnDisk{}, dir)
		if err != nil {
			return err
		}
		err = (&initialization.ConfigureInventoryInfo{
			Pkg:         p,
			Factory:     mr.factory,
			Quiet:       true,
			InventoryID: prevID,
			RGFileName:  mr.rgFile,
			Force:       mr.force,
		}).Run(ctx)

		if err != nil {
			var invExistsError *initialization.InvExistsError
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
