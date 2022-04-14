package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "version for this command",
	Run:     Version,
}

func Version(cmd *cobra.Command, args []string) {
	fmt.Println(version)
}
