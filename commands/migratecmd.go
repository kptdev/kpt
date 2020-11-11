// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/provider"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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
}

// NewMigrateRunner returns a pointer to an initial MigrateRunner structure.
func GetMigrateRunner(cmProvider provider.Provider, rgProvider provider.Provider, ioStreams genericclioptions.IOStreams) *MigrateRunner {
	r := &MigrateRunner{
		ioStreams:   ioStreams,
		dryRun:      false,
		initOptions: NewKptInitOptions(cmProvider.Factory(), ioStreams),
		cmProvider:  cmProvider,
		rgProvider:  rgProvider,
		dir:         "",
	}
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
	return GetMigrateRunner(configMapProvider, resourceGroupProvider, ioStreams).Command
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
		return err
	}
	cmObjs, err := mr.retrieveInvObjs(cmInvObj)
	if err != nil {
		return err
	}
	if len(cmObjs) > 0 {
		// Migrate the ConfigMap inventory objects to a ResourceGroup custom resource.
		if err = mr.migrateObjs(cmObjs, bytes.NewReader(stdinBytes), args); err != nil {
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
	// Transform the ResourceGroup custom resource definition in the string rgCrd
	// into an Unstructured object.
	crd, err := stringToUnstructured(rgCrd)
	if err != nil {
		return err
	}
	// Get the client and RESTMapping from the CRD.
	mapper, err := mr.cmProvider.Factory().ToRESTMapper()
	if err != nil {
		return err
	}
	// NOTE: The mapper must be told a preferred version (rgCrdVersion) which matches
	// the version stored in the rgCrd string. Otherwise, the RESTMapper may have
	// a different default version in the mapping, causing version skew errors.
	mapping, err := mapper.RESTMapping(crd.GroupVersionKind().GroupKind(), rgCrdVersion)
	if err != nil {
		return err
	}
	client, err := mr.cmProvider.Factory().UnstructuredClientForMapping(mapping)
	if err != nil {
		return err
	}
	helper := resource.NewHelper(client, mapping)
	klog.V(4).Infof("applying ResourceGroup CRD")
	// Set the "last-applied-annotation" so future applies work correctly.
	if err := util.CreateApplyAnnotation(crd, unstructured.UnstructuredJSONScheme); err != nil {
		return err
	}
	var clearResourceVersion = false
	// Apply the CRD to the cluster and ignore already exists error.
	// Empty namespace, since CRD's are cluster-scoped.
	_, err = helper.Create("", clearResourceVersion, crd)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// stringToUnstructured transforms a single resource represented by
// the passed string into a pointer to an "Unstructured" object,
// or an error if one occurred.
func stringToUnstructured(str string) (*unstructured.Unstructured, error) {
	node, err := yaml.Parse(str)
	if err != nil {
		return nil, err
	}
	b, err := node.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: m}, nil
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
func (mr *MigrateRunner) retrieveConfigMapInv(reader io.Reader, args []string) (*unstructured.Unstructured, error) {
	fmt.Fprint(mr.ioStreams.Out, "  retrieve the current ConfigMap inventory...")
	cmReader, err := mr.cmProvider.ManifestReader(reader, args)
	if err != nil {
		return nil, err
	}
	objs, err := cmReader.Read()
	if err != nil {
		return nil, err
	}
	cmInv, _, err := inventory.SplitUnstructureds(objs)
	if err != nil {
		return nil, err
	}
	return cmInv, nil
}

// retrieveInvObjs returns the object references from the passed
// inventory object by querying the inventory object in the cluster,
// or an error if one occurred.
func (mr *MigrateRunner) retrieveInvObjs(invObj *unstructured.Unstructured) ([]object.ObjMetadata, error) {
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
func (mr *MigrateRunner) migrateObjs(cmObjs []object.ObjMetadata, reader io.Reader, args []string) error {
	fmt.Fprint(mr.ioStreams.Out, "  migrate inventory to ResourceGroup...")
	if len(cmObjs) == 0 {
		fmt.Fprint(mr.ioStreams.Out, "no inventory objects found\n")
		return nil
	}
	rgReader, err := mr.rgProvider.ManifestReader(reader, args)
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
	_, err = rgInvClient.Merge(rgInv, cmObjs)
	if err != nil {
		return err
	}
	fmt.Fprint(mr.ioStreams.Out, "success\n")
	return nil
}

// deleteConfigMapInv removes the passed inventory object from the
// cluster. Returns an error if one occurred.
func (mr *MigrateRunner) deleteConfigMapInv(invObj *unstructured.Unstructured) error {
	fmt.Fprint(mr.ioStreams.Out, "  deleting old ConfigMap inventory object...")
	cmInvClient, err := mr.cmProvider.InventoryClient()
	if err != nil {
		return err
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
			err = os.Remove(cmFilename)
			if err != nil {
				fmt.Fprint(mr.ioStreams.Out, "failed\n")
				return err
			}
			fmt.Fprint(mr.ioStreams.Out, "success\n")
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

var rgCrdVersion = "v1beta1"

// The ResourceGroup custom resource definition.
var rgCrd = `
apiVersion: apiextensions.k8s.io/` + rgCrdVersion + `
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.2.5
  name: resourcegroups.kpt.dev
spec:
  group: kpt.dev
  names:
    kind: ResourceGroup
    listKind: ResourceGroupList
    plural: resourcegroups
    singular: resourcegroup
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: ResourceGroup is the Schema for the resourcegroups API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase.
            More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: ResourceGroupSpec defines the desired state of ResourceGroup
          properties:
            descriptor:
              description: Descriptor regroups the information and metadata about
                a resource group
              properties:
                description:
                  description: Description is a brief description of a group of resources
                  type: string
                links:
                  description: Links are a list of descriptive URLs intended to be
                    used to surface additional information
                  items:
                    properties:
                      description:
                        description: Description explains the purpose of the link
                        type: string
                      url:
                        description: Url is the URL of the link
                        type: string
                    required:
                    - description
                    - url
                    type: object
                  type: array
                revision:
                  description: Revision is an optional revision for a group of resources
                  type: string
                type:
                  description: Type can contain prefix, such as Application/WordPress
                    or Service/Spanner
                  type: string
              type: object
            resources:
              description: Resources contains a list of resources that form the resource group
              items:
                description: ObjMetadata organizes and stores the identifying information
                  for an object. This struct (as a string) is stored in a grouping
                  object to keep track of sets of applied objects.
                properties:
                  group:
                    type: string
                  kind:
                    type: string
                  name:
                    type: string
                  namespace:
                    type: string
                required:
                - group
                - kind
                - name
                - namespace
                type: object
              type: array
          type: object
        status:
          description: ResourceGroupStatus defines the observed state of ResourceGroup
          properties:
            conditions:
              description: Conditions lists the conditions of the current status for
                the group
              items:
                properties:
                  lastTransitionTime:
                    description: last time the condition transit from one status to
                      another
                    format: date-time
                    type: string
                  message:
                    description: human-readable message indicating details about last
                      transition
                    type: string
                  reason:
                    description: one-word CamelCase reason for the condition's last
                      transition
                    type: string
                  status:
                    description: Status of the condition
                    type: string
                  type:
                    description: Type of the condition
                    type: string
                required:
                - status
                - type
                type: object
              type: array
            observedGeneration:
              description: ObservedGeneration is the most recent generation observed.
                It corresponds to the Object's generation, which is updated on mutation
                by the API Server. Everytime the controller does a successful reconcile,
                it sets ObservedGeneration to match ResourceGroup.metadata.generation.
              format: int64
              type: integer
            resourceStatuses:
              description: ResourceStatuses lists the status for each resource in
                the group
              items:
                description: ResourceStatus contains the status of a given resource
                  uniquely identified by its group, kind, name and namespace.
                properties:
                  conditions:
                    items:
                      properties:
                        lastTransitionTime:
                          description: last time the condition transit from one status
                            to another
                          format: date-time
                          type: string
                        message:
                          description: human-readable message indicating details about
                            last transition
                          type: string
                        reason:
                          description: one-word CamelCase reason for the condition’s
                            last transition
                          type: string
                        status:
                          description: Status of the condition
                          type: string
                        type:
                          description: Type of the condition
                          type: string
                      required:
                      - status
                      - type
                      type: object
                    type: array
                  group:
                    type: string
                  kind:
                    type: string
                  name:
                    type: string
                  namespace:
                    type: string
                  status:
                    description: Status describes the status of a resource
                    type: string
                required:
                - group
                - kind
                - name
                - namespace
                - status
                type: object
              type: array
          required:
          - observedGeneration
          type: object
      type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`
