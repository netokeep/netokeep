package main

import (
	"fmt"

	"github.com/spf13/cobra"
)


func createStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the NetoKeep client",
		Long:  "Start the NetoKeep client, listening for SSH connections and forwarding TCP traffic.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("NetoKeep Client Start Function.")
		},
	}
}
