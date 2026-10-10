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
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	rgfilev1alpha1 "github.com/kptdev/kpt/api/resourcegroup/v1alpha1"
	"github.com/kptdev/kpt/internal/docs/generated/livedocs"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/lib/errors"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/kptdev/kpt/pkg/lib/util/attribution"
	pathutil "github.com/kptdev/kpt/pkg/lib/util/path"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	k8scmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const defaultInventoryName = "inventory"

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

	cmd.Flags().StringVar(&r.Name, "name", "", "Inventory object name")
	cmd.Flags().BoolVar(&r.Force, "force", false, "Set inventory values even if already set in Kptfile or ResourceGroup file")
	cmd.Flags().BoolVar(&r.Quiet, "quiet", false, "If true, do not print output message for initialization")
	cmd.Flags().StringVar(&r.InventoryID, "inventory-id", "", "Inventory id for the package")
	cmd.Flags().StringVar(&r.RGFileName, "rg-file", rgfilev1alpha1.RGFileName, "Name of the file holding the ResourceGroup resource.")
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

	namespace, err := findNamespace(c.Factory.ToRawKubeConfigLoader(), c.Pkg.UniquePath.String())
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	namespace = strings.TrimSpace(namespace)
	if !c.Quiet {
		pr.Printf("initializing %q data (namespace: %s)...", c.RGFileName, namespace)
	}

	// Autogenerate the name if it is not provided through the flag.
	if c.Name == "" {
		randomSuffix := common.RandomStr()
		c.Name = fmt.Sprintf("%s-%s", defaultInventoryName, randomSuffix)
	}

	// Autogenerate the inventory ID if not provided through the flag.
	if c.InventoryID == "" {
		c.InventoryID, err = generateID(namespace, c.Name, time.Now())
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

// namespaceLoader provides the namespace set by the --namespace flag or,
// failing that, the namespace of the current kubeconfig context.
type namespaceLoader interface {
	Namespace() (string, bool, error)
}

// findNamespace returns the namespace for the inventory object. The namespace
// set by the --namespace flag takes precedence. Otherwise, if all the resources
// in the package are in the same namespace, that namespace is returned. If
// they are not, the namespace of the current kubeconfig context is returned.
//
// This is config.FindNamespace of cli-utils, except that it does not fail if
// the package contains YAML that is a bare sequence instead of a KRM resource.
func findNamespace(loader namespaceLoader, dir string) (string, error) {
	namespace, enforceNamespace, err := loader.Namespace()
	if err != nil {
		return "", err
	}
	if enforceNamespace {
		klog.V(6).Infof("enforcing namespace: %s", namespace)
		return namespace, nil
	}

	ns, allInSameNs, err := allInSameNamespace(dir)
	if err != nil {
		return "", err
	}
	if allInSameNs {
		klog.V(6).Infof("all in same namespace: %s", ns)
		return ns, nil
	}
	klog.V(6).Infof("returning namespace: %s", namespace)
	return namespace, nil
}

// allInSameNamespace goes through all the resources in the package and checks
// their namespace. If they all have the namespace set and it has the same
// value for all of them, that namespace is returned and the second return
// value is true. Otherwise, no namespace is returned and the second return
// value is false.
func allInSameNamespace(dir string) (string, bool, error) {
	nodes, err := (&kio.LocalPackageReader{
		PackagePath:     dir,
		WrapBareSeqNode: true,
	}).Read()
	if err != nil {
		return "", false, err
	}

	// Filter out any resources with the LocalConfig annotation.
	nodes, err = (&filters.IsLocalConfig{}).Filter(nodes)
	if err != nil {
		return "", false, err
	}

	var ns string
	for _, node := range nodes {
		// Skip bare sequences, such as the JSON 6902 patches used by
		// kustomize. They are not resources, so they have no namespace.
		if node.Field(yaml.BareSeqNodeWrappingKey) != nil {
			continue
		}
		rm, err := node.GetMeta()
		if err != nil {
			return "", false, err
		}
		// Skip cluster-scoped resources. If the scope is not known, assume
		// that the resource is namespaced.
		namespaced, found := openapi.IsNamespaceScoped(rm.TypeMeta)
		if found && !namespaced {
			klog.V(6).Infof("cluster-scoped resource %s--skip namespace calc", rm.TypeMeta)
			continue
		}
		if rm.Namespace == "" {
			klog.V(6).Infof("one resource missing namespace (%s): return empty namespace", rm.Name)
			return "", false, nil
		}
		if ns == "" {
			ns = rm.Namespace
		} else if rm.Namespace != ns {
			klog.V(6).Infof("two namespaces not same: %s versus %s", rm.Namespace, ns)
			return "", false, nil
		}
	}
	return ns, ns != "", nil
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
	// Initialize new resourcegroup object, as rg should have been nil.
	rg = &rgfilev1alpha1.ResourceGroup{ResourceMeta: rgfilev1alpha1.DefaultMeta}
	// // Finally, set the inventory parameters in the ResourceGroup object and write it.
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
		return errors.E(op, errors.IO, kptfilev1.UniquePath(dir), err)
	}

	// fyi: perm is ignored if the file already exists
	err = os.WriteFile(filepath.Join(dir, filename), b, 0600)
	if err != nil {
		return errors.E(op, errors.IO, kptfilev1.UniquePath(dir), err)
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
func kptfileInventoryEmpty(inv *kptfilev1.Inventory) bool {
	return inv == nil
}
