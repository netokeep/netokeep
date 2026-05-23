package main

import (
	"fmt"
	"netokeep/internal/local"
	"netokeep/pkg/utils"

	"github.com/spf13/cobra"
)

func createUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the netokeep server and client.",
		Long:  "This command will remove all netokeep related files and programs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.PromptConfirm("Are you sure to uninstall Netokeep?", false) {
				return nil
			}

			if err := local.RemoveStateDir(); err != nil {
				return err
			}
			if err := local.RemoveConfigDir(); err != nil {
				return err
			}
			if err := local.RemovePrograms(); err != nil {
				return err
			}

			fmt.Printf("\n✔ Netokeep successfully uninstalled.\n")
			return nil
		},
	}
}
