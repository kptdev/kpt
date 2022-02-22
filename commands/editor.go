package commands

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/cad/function"
	"github.com/GoogleContainerTools/kpt/internal/cad/resource"
	"github.com/spf13/cobra"
)

func EditorCommand(ctx context.Context, name string) *cobra.Command {
	editor := &cobra.Command{
		Use:   "editor",
		Short: `Edit local package resources`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	editor.AddCommand(addCommand(ctx))
	return editor
}

func addCommand(ctx context.Context) *cobra.Command {
	adder := &cobra.Command{
		Use:     "add",
		Short:   `Add a resource or a function to your package`,
		Aliases: []string{"set"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	adder.AddCommand(resource.NewAdd(ctx).Command)
	adder.AddCommand(function.NewAdd(ctx).Command)
	return adder
}
