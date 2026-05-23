package main

import "github.com/spf13/cobra"

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
	rootCmd.AddCommand(createInstallCmd())
	rootCmd.AddCommand(createUninstallCmd())
	rootCmd.AddCommand(createStartCmd())
	rootCmd.AddCommand(createCoreCmd())
	rootCmd.AddCommand(createStopCmd())
	rootCmd.AddCommand(createStatusCmd())
	rootCmd.Execute()
}
