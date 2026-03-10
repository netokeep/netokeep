package main

import (
	"netokeep/internal/commads/nks"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "nks",
		Version: version,
		Short:   "netokeep server",
		Long:    `Setup the NetoKeep server to proxy SSH and HTTP traffic.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(nks.CreateStartCmd())
	rootCmd.AddCommand(nks.CreateStopCmd())
	rootCmd.Execute()
}
