package cmd

import (
	"fmt"

	"tokyo/pkg/profile"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newToolCommand(profile.ClaudeTool()))
	rootCmd.AddCommand(newToolCommand(profile.CodexTool()))
}

func newToolCommand(t profile.Tool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   t.Name,
		Short: fmt.Sprintf("Manage %s configuration profiles", t.DisplayName),
	}

	cmd.AddCommand(
		newSwitchCommand(t),
		newCurrentCommand(t),
		newListCommand(t),
		newSaveCommand(t),
		newDeleteCommand(t),
	)

	return cmd
}

func newSwitchCommand(t profile.Tool) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: fmt.Sprintf("Switch %s to a profile", t.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return profile.Switch(t, args[0])
		},
	}
}

func newCurrentCommand(t profile.Tool) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: fmt.Sprintf("Show current %s profile", t.DisplayName),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := profile.Current(t)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		},
	}
}

func newListCommand(t profile.Tool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s profiles", t.DisplayName),
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := profile.List(t)
			if err != nil {
				return err
			}
			for _, p := range profiles {
				fmt.Fprintln(cmd.OutOrStdout(), p)
			}
			return nil
		},
	}
}

func newSaveCommand(t profile.Tool) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "save <profile>",
		Short: fmt.Sprintf("Save current %s configuration as a profile", t.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return profile.Save(t, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing profile")

	return cmd
}

func newDeleteCommand(t profile.Tool) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <profile>",
		Short: fmt.Sprintf("Delete a %s profile", t.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cleared, err := profile.Delete(t, args[0])
			if err != nil {
				return err
			}
			if cleared {
				fmt.Fprintln(cmd.OutOrStdout(), "Deleted active profile; current profile is now <custom>.")
			}
			return nil
		},
	}
}
