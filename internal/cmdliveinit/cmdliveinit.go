// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdliveinit

import (
	"crypto/sha1"
	"fmt"
	"strconv"
	"strings"
	"time"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
)

const defaultInventoryName = "inventory"

// InvExistsError defines new error when the inventory
// values have already been set on the Kptfile.
type InvExistsError struct{}

func (i *InvExistsError) Error() string {
	return InvExistsErrorMsg
}

var InvExistsErrorMsg = `ResourceGroup configuration has already been created. Changing
them after a package has been applied to the cluster can lead to
undesired results. Use the --force flag to suppress this error.
`

// KptInitOptions encapsulates fields for kpt init command. This init command
// fills in inventory values in the Kptfile.
type KptInitOptions struct {
	factory     cmdutil.Factory
	ioStreams   genericclioptions.IOStreams
	dir         string // Directory with Kptfile
	Force       bool   // Set inventory values even if already set in Kptfile
	Name        string // Inventory object name
	namespace   string // Inventory object namespace
	InventoryID string // Inventory object unique identifier label
	Quiet       bool   // Output message during initialization
}

// NewKptInitOptions returns a pointer to an initial KptInitOptions structure.
func NewKptInitOptions(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *KptInitOptions {
	return &KptInitOptions{
		factory:   f,
		ioStreams: ioStreams,
		Quiet:     false,
	}
}

// Complete fills in fields for KptInitOptions based on the passed "args".
func (io *KptInitOptions) Run(args []string) error {
	// Set the init options directory.
	if len(args) != 1 {
		return fmt.Errorf("need one 'directory' arg; have %d", len(args))
	}
	dir, err := config.NormalizeDir(args[0])
	if err != nil {
		return err
	}
	io.dir = dir
	// Set the init options inventory object namespace.
	ns, err := config.FindNamespace(io.factory.ToRawKubeConfigLoader(), io.dir)
	if err != nil {
		return err
	}
	io.namespace = strings.TrimSpace(ns)
	if !io.Quiet {
		fmt.Fprintf(io.ioStreams.Out, "initializing Kptfile inventory info (namespace: %s)...", io.namespace)
	}
	// Set the init options default inventory object name, if not set by flag.
	if io.Name == "" {
		randomSuffix := common.RandomStr(time.Now().UTC().UnixNano())
		io.Name = fmt.Sprintf("%s-%s", defaultInventoryName, randomSuffix)
	}
	// Set the init options inventory id label, if not already set.
	if io.InventoryID == "" {
		id, err := generateID(io.Name, io.namespace, time.Now())
		if err != nil {
			return err
		}
		io.InventoryID = id
	}
	// Finally, update these values in the Inventory section of the Kptfile.
	err = io.updateKptfile()
	if !io.Quiet {
		if err == nil {
			fmt.Fprintln(io.ioStreams.Out, "success")
		} else {
			fmt.Fprintln(io.ioStreams.Out, "failed")
		}
	}
	return err
}

// generateID returns the string which is a SHA1 hash of the passed namespace
// and name, with the unix timestamp string concatenated. Returns an error
// if either the namespace or name are empty.
func generateID(namespace string, name string, t time.Time) (string, error) {
	hashStr, err := generateHash(namespace, name)
	if err != nil {
		return "", err
	}
	timeStr := strconv.FormatInt(t.UTC().UnixNano(), 10)
	return fmt.Sprintf("%s-%s", hashStr, timeStr), nil
}

// generateHash returns the SHA1 hash of the concatenated "namespace:name" string,
// or an error if either namespace or name is empty.
func generateHash(namespace string, name string) (string, error) {
	if len(namespace) == 0 || len(name) == 0 {
		return "", fmt.Errorf("can not generate hash with empty namespace or name")
	}
	str := fmt.Sprintf("%s:%s", namespace, name)
	h := sha1.New()
	if _, err := h.Write([]byte(str)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", (h.Sum(nil))), nil
}

// Run fills in the inventory object values into the Kptfile.
func (io *KptInitOptions) updateKptfile() error {
	// Read the Kptfile io io.dir
	kf, err := kptfileutil.ReadFile(io.dir)
	if err != nil {
		return err
	}
	// Validate the inventory values don't already exist
	isEmpty := kptfileInventoryEmpty(kf.Inventory)
	if !isEmpty && !io.Force {
		return &InvExistsError{}
	}
	// Check the new inventory values are valid.
	if err := io.validate(); err != nil {
		return err
	}
	// Finally, set the inventory parameters in the Kptfile and write it.
	kf.Inventory = &kptfilev1alpha2.Inventory{
		Namespace:   io.namespace,
		Name:        io.Name,
		InventoryID: io.InventoryID,
	}
	if err := kptfileutil.WriteFile(io.dir, kf); err != nil {
		return err
	}
	return nil
}

// validate ensures the inventory object parameters are valid.
func (io *KptInitOptions) validate() error {
	// name is required
	if len(io.Name) == 0 {
		return fmt.Errorf("inventory name is missing")
	}
	// namespace is required
	if len(io.namespace) == 0 {
		return fmt.Errorf("inventory namespace is missing")
	}
	// inventoryID is required
	if len(io.InventoryID) == 0 {
		return fmt.Errorf("inventoryID is missing")
	}
	return nil
}

// kptfileInventoryEmpty returns true if the Inventory structure
// in the Kptfile is empty; false otherwise.
func kptfileInventoryEmpty(inv *kptfilev1alpha2.Inventory) bool {
	return inv == nil
}

// NewCmdInit returns the cobra command for the init command.
func NewCmdInit(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	io := NewKptInitOptions(f, ioStreams)
	cmd := &cobra.Command{
		Use:                   "init [PKG_PATH]",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Initialize inventory parameters into Kptfile"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// default to current working directory
				args = append(args, ".")
			}
			return io.Run(args)
		},
	}
	cmd.Flags().StringVar(&io.Name, "name", "", "Inventory object name")
	cmd.Flags().BoolVar(&io.Force, "force", false, "Set inventory values even if already set in Kptfile")
	cmd.Flags().BoolVar(&io.Quiet, "quiet", false, "If true, do not print output message for initialization")
	return cmd
}
