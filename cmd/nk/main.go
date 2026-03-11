package main

import (
	"netokeep/internal/commands/nk"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "nk",
		Version: version,
		Short:   "netokeep client",
		Long:    `Setup the NetoKeep client, listening for SSH connections and forwarding TCP traffic.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(nk.CreateStartCmd())
	rootCmd.Execute()
}
