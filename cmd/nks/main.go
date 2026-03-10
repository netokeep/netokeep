package main

import (
	"github.com/spf13/cobra"
)

var version = "dev"

func createStartCmd() *cobra.Command {
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			println("start the server.")
		},
	}

	var inPort uint16
	var outPort uint16
	startCmd.Flags().Uint16VarP(&inPort, "inPort", "i", 22, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return startCmd
}

func createStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			println("stop the server.")
		},
	}
	return stopCmd
}

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
	rootCmd.AddCommand(createStartCmd())
	rootCmd.AddCommand(createStopCmd())
	rootCmd.Execute()
}
