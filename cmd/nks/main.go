package main

import (
	"netokeep/internal/commands/nks"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "nks",
		Version: version,
		Short:   "netokeep server",
		Long:    `Setup the NetoKeep server, receiving SSH connections and proxying TCP traffic.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(nks.CreateStartCmd())
	rootCmd.Execute()
}
