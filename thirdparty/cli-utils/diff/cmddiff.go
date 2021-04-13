// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package diff

import (
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/cmd/diff"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/pkg/common"
)

const tmpDirPrefix = "diff-cmd"

// NewCmdDiff returns cobra command to implement client-side diff of package
// directory. For each local config file, get the resource in the cluster
// and diff the local config resource against the resource in the cluster.
func NewCmdDiff(f util.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	options := diff.NewDiffOptions(ioStreams)
	cmd := &cobra.Command{
		Use:                   "diff [DIR | -]",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Diff local config against cluster applied version"),
		Args:                  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// default to the current working directory
				args = append(args, ".")
			}
			cleanupFunc, err := Initialize(options, f, args)
			defer cleanupFunc()
			util.CheckErr(err)
			util.CheckErr(options.Run())
		},
	}

	return cmd
}

// Initialize fills in the DiffOptions in preparation for DiffOptions.Run().
// Returns a cleanup function for removing temp files after expanding stdin, or
// error if there is an error filling in the options or if there
// is not one argument that is a directory.
func Initialize(o *diff.DiffOptions, f util.Factory, args []string) (func(), error) {
	cleanupFunc := func() {}
	// Validate the only argument is a (package) directory path.
	filenameFlags, err := common.DemandOneDirectory(args)
	if err != nil {
		return cleanupFunc, err
	}
	// Process input from stdin
	if len(args) == 0 {
		tmpDir, err := createTempDir()
		if err != nil {
			return cleanupFunc, err
		}
		cleanupFunc = func() {
			os.RemoveAll(tmpDir)
		}
		filenameFlags.Filenames = &[]string{tmpDir}
		klog.V(6).Infof("stdin diff command temp dir: %s", tmpDir)
		if err := common.FilterInputFile(os.Stdin, tmpDir); err != nil {
			return cleanupFunc, err
		}
	} else {
		// We do not want to diff the inventory object. So we expand
		// the config file paths, excluding the inventory object.
		filenameFlags, err = common.ExpandPackageDir(filenameFlags)
		if err != nil {
			return cleanupFunc, err
		}
	}
	o.FilenameOptions = filenameFlags.ToOptions()

	o.OpenAPISchema, err = f.OpenAPISchema()
	if err != nil {
		return cleanupFunc, err
	}

	o.DiscoveryClient, err = f.ToDiscoveryClient()
	if err != nil {
		return cleanupFunc, err
	}

	o.DynamicClient, err = f.DynamicClient()
	if err != nil {
		return cleanupFunc, err
	}

	o.DryRunVerifier = resource.NewDryRunVerifier(o.DynamicClient, o.DiscoveryClient)

	o.CmdNamespace, o.EnforceNamespace, err = f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return cleanupFunc, err
	}

	o.Builder = f.NewBuilder()

	// We don't support server-side apply diffing yet.
	o.ServerSideApply = false
	o.ForceConflicts = false

	return cleanupFunc, nil
}

func createTempDir() (string, error) {
	// Create a temporary file with the passed prefix in
	// the default temporary directory.
	tmpDir, err := ioutil.TempDir("", tmpDirPrefix)
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}
