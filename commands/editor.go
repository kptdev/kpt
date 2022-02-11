package commands

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/cad"
	"github.com/spf13/cobra"
)

func EditorCommand(ctx context.Context, name string) *cobra.Command {
	/*
		editor := &cobra.Command{
			Use:   "editor",
			Short: `Edit local package resources`,
			// Aliases: []string{"set", "add"},
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		}
			editor.AddCommand()
			return editor
	*/
	return cad.NewSetter(ctx).Command
}
