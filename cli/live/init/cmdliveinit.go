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
	"github.com/kptdev/kpt/pkg/lib/krmvalidate"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/kptdev/kpt/pkg/lib/util/attribution"
	pathutil "github.com/kptdev/kpt/pkg/lib/util/path"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/config"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const defaultInventoryName = "inventory"

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

	// Pre-scan for non-KRM files before config.FindNamespace() reads them.
	scanResult, err := krmvalidate.ScanFlatAndDecide(filesys.FileSystemOrOnDisk{}, c.Pkg.UniquePath.String(), pr.Printf)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	var namespace string
	if len(scanResult.SkippedFiles()) > 0 {
		// FindNamespace re-reads the package with no skip hook, so it
		// would hit the excused files. Replicate its priority order
		// instead: explicit --namespace wins, else check agreement.
		loaderNamespace, enforceNamespace, nsErr := c.Factory.ToRawKubeConfigLoader().Namespace()
		if nsErr != nil {
			return errors.E(op, c.Pkg.UniquePath,
				fmt.Errorf("failed to determine inventory namespace: %w; please provide the namespace using the --namespace flag", nsErr))
		}
		namespace = loaderNamespace
		if !enforceNamespace {
			filtered, ferr := filterSkipped(scanResult.RootResources, scanResult.SkipFunc(c.Pkg.UniquePath.String()))
			if ferr != nil {
				return errors.E(op, c.Pkg.UniquePath, ferr)
			}
			ns, ok, aerr := allInSameNamespace(filtered)
			if aerr != nil {
				return errors.E(op, c.Pkg.UniquePath, aerr)
			}
			if ok {
				namespace = ns
			}
		}
	} else {
		namespace, err = config.FindNamespace(c.Factory.ToRawKubeConfigLoader(), c.Pkg.UniquePath.String())
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath,
				fmt.Errorf("failed to determine inventory namespace from package resources: %w; please provide the namespace using the --namespace flag", err))
		}
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
	at := attribution.Attributor{
		PackagePaths: []string{c.Pkg.UniquePath.String()},
		CmdGroup:     "live",
		FileSkipFunc: scanResult.SkipFunc(c.Pkg.UniquePath.String()),
	}
	at.Process()
	return nil
}

// allInSameNamespace is kpt's own copy of cli-utils's unexported
// config.allInSameNamespace, operating on an already-filtered
// resource list instead of re-reading the directory itself.
func allInSameNamespace(nodes []*yaml.RNode) (string, bool, error) {
	nodes, err := (&filters.IsLocalConfig{}).Filter(nodes)
	if err != nil {
		return "", false, err
	}

	var ns string
	for _, node := range nodes {
		rm, err := node.GetMeta()
		if err != nil {
			return "", false, err
		}
		if namespaced, found := openapi.IsNamespaceScoped(rm.TypeMeta); found && !namespaced {
			continue
		}
		if rm.Namespace == "" {
			return "", false, nil
		}
		if ns == "" {
			ns = rm.Namespace
		} else if rm.Namespace != ns {
			return "", false, nil
		}
	}
	if ns == "" {
		return "", false, nil
	}
	return ns, true, nil
}

// filterSkipped drops any node skip reports true for, by its
// kioutil.PathAnnotation. A nil skip leaves nodes unchanged.
func filterSkipped(nodes []*yaml.RNode, skip func(relPath string) bool) ([]*yaml.RNode, error) {
	if skip == nil {
		return nodes, nil
	}
	var out []*yaml.RNode
	for _, n := range nodes {
		path, _, err := kioutil.GetFileAnnotations(n)
		if err != nil {
			return nil, err
		}
		if !skip(path) {
			out = append(out, n)
		}
	}
	return out, nil
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
			return errors.E(op, p.UniquePath, &errors.InvInKfExistsError{})
		}

		// Set the Kptfile inventory to be nil if we force write to resourcegroup instead.
		kf.Inventory = nil
	}

	// Validate the inventory values don't already exist in Resourcegroup.
	if rg != nil && !force {
		return errors.E(op, p.UniquePath, &errors.InvInRGExistsError{})
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
