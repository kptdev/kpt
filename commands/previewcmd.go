package commands

import (
	"github.com/GoogleContainerTools/kpt/internal/util/setters"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/preview"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// Get PreviewRunner returns a wrapper around the cli-utils preview command PreviewRunner. Sets
// up the Run on this wrapped runner to be the PreviewRunnerWrapper run.
func GetPreviewRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *PreviewRunnerWrapper {
	previewRunner := preview.GetPreviewRunner(provider, loader, ioStreams)
	w := &PreviewRunnerWrapper{
		previewRunner: previewRunner,
		factory:       provider.Factory(),
	}
	// Set the wrapper run to be the RunE function for the wrapped command.
	previewRunner.Command.RunE = w.RunE
	return w
}

// PreviewRunnerWrapper encapsulates the cli-utils preview command PreviewRunner as well
// as structures necessary to run.
type PreviewRunnerWrapper struct {
	previewRunner *preview.PreviewRunner
	factory       cmdutil.Factory
}

// Command returns the wrapped PreviewRunner cobraCommand structure.
func (w *PreviewRunnerWrapper) Command() *cobra.Command {
	return w.previewRunner.Command
}

// RunE checks if required setters are set as a pre-step if Kptfile
// exists in the package path. Then the wrapped PreviewRunner is
// invoked. Returns an error if one happened.
func (w *PreviewRunnerWrapper) RunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		if err := setters.CheckForRequiredSetters(args[0]); err != nil {
			return err
		}
	}
	return w.previewRunner.RunE(cmd, args)
}
