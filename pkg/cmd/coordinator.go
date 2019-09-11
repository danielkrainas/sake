package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/danielkrainas/sake/pkg/factory"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/util/log"
)

func init() {
	rootCmd.AddCommand(coordinatorCmd)
}

var coordinatorCmd = &cobra.Command{
	Use:   "coordinator",
	Short: "run the saga engine coordinator",
	Long:  "run the saga engine coordinator",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := service.ResolveConfig(configPath)
		if err != nil {
			log.Fatal(err.Error())
			return
		}

		ctx, cancel := context.WithCancel(rootContext)
		coordinator, err := factory.Coordinator(ctx, config)
		if err != nil {
			log.Fatal("initialization failed", zap.Error(err))
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM)
		select {
		case <-coordinator.WaitForShutdown():
		case <-c:
			cancel()
		}
	},
}
