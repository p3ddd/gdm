package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "version for this command",
	Run:     VersionFunc,
}

func VersionFunc(cmd *cobra.Command, args []string) {
	fmt.Println(version)
}
