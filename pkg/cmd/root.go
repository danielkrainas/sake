package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	configPath string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "")
}

var rootContext context.Context

var rootCmd = &cobra.Command{
	Use:   "sake",
	Short: "sake",
	Long:  "sake",
}

func Execute(ctx context.Context) {
	SetContext(ctx)
	rootCmd.Execute()
}

func SetContext(ctx context.Context) {
	rootContext = ctx
}
