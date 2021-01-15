// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/kpt/pkg/client"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// MigrateRunner encapsulates fields for the kpt migrate command.
type MigrateRunner struct {
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams

	dir         string
	dryRun      bool
	initOptions *KptInitOptions
	cmProvider  provider.Provider
	rgProvider  provider.Provider
	cmLoader    manifestreader.ManifestLoader
	rgLoader    manifestreader.ManifestLoader
}

// NewMigrateRunner returns a pointer to an initial MigrateRunner structure.
func GetMigrateRunner(cmProvider provider.Provider, rgProvider provider.Provider,
	cmLoader manifestreader.ManifestLoader, rgLoader manifestreader.ManifestLoader,
	ioStreams genericclioptions.IOStreams) *MigrateRunner {
	r := &MigrateRunner{
		ioStreams:   ioStreams,
		dryRun:      false,
		initOptions: NewKptInitOptions(cmProvider.Factory(), ioStreams),
		cmProvider:  cmProvider,
		rgProvider:  rgProvider,
		cmLoader:    cmLoader,
		rgLoader:    rgLoader,
		dir:         "",
	}
	r.initOptions.Quiet = true // Do not print output during migration
	cmd := &cobra.Command{
		Use:                   "migrate DIRECTORY",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Migrate inventory from ConfigMap to ResourceGroup custom resource"),
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().StringVar(&r.initOptions.name, "name", "", "Inventory object name")
	cmd.Flags().BoolVar(&r.initOptions.force, "force", false, "Set inventory values even if already set in Kptfile")
	cmd.Flags().BoolVar(&r.dryRun, "dry-run", false, "Do not actually migrate, but show steps")

	r.Command = cmd
	return r
}

// NewCmdMigrate returns the cobra command for the migrate command.
func NewCmdMigrate(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	configMapProvider := provider.NewProvider(f)
	resourceGroupProvider := live.NewResourceGroupProvider(f)
	cmLoader := manifestreader.NewManifestLoader(f)
	rgLoader := live.NewResourceGroupManifestLoader(f)
	return GetMigrateRunner(configMapProvider, resourceGroupProvider, cmLoader, rgLoader, ioStreams).Command
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
	// Update the Kptfile with the resource group values (e.g. namespace, name, id).
	if err := mr.updateKptfile(args); err != nil {
		return err
	}
	// Retrieve the current ConfigMap inventory objects.
	cmInvObj, err := mr.retrieveConfigMapInv(bytes.NewReader(stdinBytes), args)
	if err != nil {
		if _, ok := err.(inventory.NoInventoryObjError); ok {
			// No ConfigMap inventory means the migration has already run before.
			klog.V(4).Infoln("swallowing no ConfigMap inventory error")
			return nil
		}
		klog.V(4).Infof("error retrieving ConfigMap inventory object: %s", err)
		return err
	}
	cmObjs, err := mr.retrieveInvObjs(cmInvObj)
	if err != nil {
		return err
	}
	if len(cmObjs) > 0 {
		// Migrate the ConfigMap inventory objects to a ResourceGroup custom resource.
		if err = mr.migrateObjs(cmObjs, cmInvObj.ID(), bytes.NewReader(stdinBytes), args); err != nil {
			return err
		}
		// Delete the old ConfigMap inventory object.
		if err = mr.deleteConfigMapInv(cmInvObj); err != nil {
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
	var err error
	// Simply return early if this is a dry run
	if mr.dryRun {
		fmt.Fprintln(mr.ioStreams.Out, "success")
		return nil
	}
	// Apply the ResourceGroup CRD to the cluster, swallowing an "AlreadyExists" error.
	err = live.ApplyResourceGroupCRD(mr.cmProvider.Factory())
	if apierrors.IsAlreadyExists(err) {
		fmt.Fprint(mr.ioStreams.Out, "already installed...")
		err = nil
	}
	if err != nil {
		fmt.Fprintln(mr.ioStreams.Out, "failed")
	} else {
		fmt.Fprintln(mr.ioStreams.Out, "success")
	}
	return err
}

// updateKptfile installs the "inventory" fields in the Kptfile.
func (mr *MigrateRunner) updateKptfile(args []string) error {
	fmt.Fprint(mr.ioStreams.Out, "  updating Kptfile inventory values...")
	if !mr.dryRun {
		if err := mr.initOptions.Run(args); err != nil {
			if _, exists := err.(*InvExistsError); exists {
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
	cmReader, err := mr.cmLoader.ManifestReader(reader, args)
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
	}
	return cmInv, err
}

// retrieveInvObjs returns the object references from the passed
// inventory object by querying the inventory object in the cluster,
// or an error if one occurred.
func (mr *MigrateRunner) retrieveInvObjs(invObj inventory.InventoryInfo) ([]object.ObjMetadata, error) {
	cmInvClient, err := mr.cmProvider.InventoryClient()
	if err != nil {
		return nil, err
	}
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
func (mr *MigrateRunner) migrateObjs(cmObjs []object.ObjMetadata, oldID string, reader io.Reader, args []string) error {
	fmt.Fprint(mr.ioStreams.Out, "  migrate inventory to ResourceGroup...")
	if len(cmObjs) == 0 {
		fmt.Fprintln(mr.ioStreams.Out, "no inventory objects found")
		return nil
	}
	if mr.dryRun {
		fmt.Fprintln(mr.ioStreams.Out, "success")
		return nil
	}
	rgReader, err := mr.rgLoader.ManifestReader(reader, args)
	if err != nil {
		return err
	}
	objs, err := rgReader.Read()
	if err != nil {
		return err
	}
	// Filter the ConfigMap inventory object.
	rgInv, err := findResourceGroupInv(objs)
	if err != nil {
		return err
	}
	rgInvClient, err := mr.rgProvider.InventoryClient()
	if err != nil {
		return err
	}
	inv := live.WrapInventoryInfoObj(rgInv)
	err = updateOwningInventoryAnnotation(mr.rgProvider.Factory(), cmObjs, oldID, inv.ID())
	if err != nil {
		return err
	}
	_, err = rgInvClient.Merge(inv, cmObjs)
	if err != nil {
		fmt.Fprintln(mr.ioStreams.Out, "failed", err.Error())
		return err
	}
	fmt.Fprintln(mr.ioStreams.Out, "success")
	return nil
}

// deleteConfigMapInv removes the passed inventory object from the
// cluster. Returns an error if one occurred.
func (mr *MigrateRunner) deleteConfigMapInv(invObj inventory.InventoryInfo) error {
	fmt.Fprint(mr.ioStreams.Out, "  deleting old ConfigMap inventory object...")
	cmInvClient, err := mr.cmProvider.InventoryClient()
	if err != nil {
		return err
	}
	if mr.dryRun {
		cmInvClient.SetDryRunStrategy(common.DryRunClient)
	}
	if err = cmInvClient.DeleteInventoryObj(invObj); err != nil {
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
					fmt.Fprintln(mr.ioStreams.Out, "failed")
					return err
				}
			}
			fmt.Fprintln(mr.ioStreams.Out, "success")
		}
	}
	return nil
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

func updateOwningInventoryAnnotation(f cmdutil.Factory, objMetas []object.ObjMetadata, old, new string) error {
	d, err := f.DynamicClient()
	if err != nil {
		return err
	}
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return err
	}
	c := client.NewClient(d, mapper)
	for _, meta := range objMetas {
		obj, err := c.Get(context.TODO(), meta)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		changed, err := client.ReplaceOwningInventoryID(obj, old, new)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		err = c.Update(context.TODO(), meta, obj, &metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
