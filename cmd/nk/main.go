package main

import (
	"github.com/spf13/cobra"
)

var version = "dev"

func createStartCmd() *cobra.Command {
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			println("start the client.")
		},
	}

	var remoteAddr string
	var outPort uint16
	startCmd.Flags().StringVarP(&remoteAddr, "remoteAddress", "r", "", "remote address to recive traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}

func createStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			println("stop the client.")
		},
	}
	return stopCmd
}

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
	rootCmd.AddCommand(createStartCmd())
	rootCmd.AddCommand(createStopCmd())
	rootCmd.Execute()
}
