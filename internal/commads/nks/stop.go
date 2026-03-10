package nks

import "github.com/spf13/cobra"

func CreateStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			println("stop the server.")
		},
	}
	return stopCmd
}
