package main

import (
	"netokeep/internal/commads/nk"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "nk",
		Version: version,
		Short:   "netokeep client",
		Long:    `Setup the NetoKeep client, receive traffic from server and forward it.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(nk.CreateStartCmd())
	rootCmd.AddCommand(nk.CreateStopCmd())
	rootCmd.Execute()
}
