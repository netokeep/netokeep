package main

import (
	"netokeep/internal/local"

	"github.com/spf13/cobra"
)

func createInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Reset up necessary configurations for 'nks'.",
		Long:  "This command will create the necessary configuration files for 'nks' if they don't exist. It will not modify existing configurations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create directories if they don't exist
			if err := local.InitializeDirs(); err != nil {
				return err
			}

			// Create the config files if not exist
			if _, err := local.LoadNksConfig(); err != nil {
				return err
			}
			return nil
		},
	}
}
