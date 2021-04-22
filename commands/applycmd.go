// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/apply"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// Get ApplyRunner returns a wrapper around the cli-utils apply command ApplyRunner. Sets
// up the Run on this wrapped runner to be the ApplyRunnerWrapper run.
func GetApplyRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *ApplyRunnerWrapper {
	applyRunner := apply.GetApplyRunner(provider, loader, ioStreams)
	w := &ApplyRunnerWrapper{
		applyRunner: applyRunner,
		factory:     provider.Factory(),
	}
	// Set the wrapper run to be the RunE function for the wrapped command.
	applyRunner.Command.RunE = w.RunE
	return w
}

// ApplyRunnerWrapper encapsulates the cli-utils apply command ApplyRunner as well
// as structures necessary to run.
type ApplyRunnerWrapper struct {
	applyRunner *apply.ApplyRunner
	factory     cmdutil.Factory
}

// Command returns the wrapped ApplyRunner cobraCommand structure.
func (w *ApplyRunnerWrapper) Command() *cobra.Command {
	return w.applyRunner.Command
}

// RunE runs the ResourceGroup CRD installation as a pre-step.
func (w *ApplyRunnerWrapper) RunE(cmd *cobra.Command, args []string) error {
	// Install ResourceGroup CRD if it does not already exist, and reset
	// the RESTMapper to make the CRD available.
	err := live.ApplyResourceGroupCRD(w.factory)
	if err == nil {
		if err := live.ResetRESTMapper(w.factory); err != nil {
			return err
		}
	} else if !apierrors.IsAlreadyExists(err) {
		return err
	}
	klog.V(4).Infoln("wrapper applyRunner run...")
	if len(args) == 0 {
		// default to the current working directory
		args = append(args, ".")
	}
	return w.applyRunner.RunE(cmd, args)
}
