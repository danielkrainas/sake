package cmd

import (
	"fmt"

	gobagcontext "github.com/danielkrainas/gobag/context"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show version information",
	Long:  "show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sake v%s\n", gobagcontext.GetVersion(rootContext))
	},
}
