package main

import "github.com/spf13/cobra"

var version = "dev"

func main() {
	var rootCmd = &cobra.Command{
		Use:     "nk",
		Version: version,
		Short:   "NetoKeep client",
		Long:    "Setup NetoKeep client, listening for SSH connections and forwarding TCP traffic.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(createStartCmd())
	rootCmd.AddCommand(createStopCmd())
	rootCmd.AddCommand(createRestartCmd())
	rootCmd.AddCommand(createStatusCmd())
	rootCmd.AddCommand(createListCmd())
	rootCmd.AddCommand(createClearCmd())
	rootCmd.AddCommand(createCoreCmd())
	rootCmd.Execute()
}
