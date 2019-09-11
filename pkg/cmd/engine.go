package cmd

import (
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
	rootCmd.AddCommand(engineCmd)
}

var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "run the orchestration engine",
	Long:  "run the orchestration engine",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := service.ResolveConfig(configPath)
		if err != nil {
			log.Fatal("configuration failure", zap.Error(err))
		}

		componentManager, err := factory.ComponentManagerWithCoordinator(rootContext, config)
		if err != nil {
			log.Fatal("initialization failed", zap.Error(err))
			return
		}

		done := make(chan struct{})
		go func() {
			if err := componentManager.Run(); err != nil {
				log.Error("component manager failed", zap.Error(err))
			}

			done <- struct{}{}
			close(done)
		}()

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM)
		signal.Notify(ch, syscall.SIGINT)
		select {
		case <-done:
		case <-ch:
			log.Info("termination signal")
			componentManager.Shutdown()
			<-done
		}
	},
}
