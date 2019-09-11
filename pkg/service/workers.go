package service

import (
	"context"
	"fmt"
	"time"

	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap"
)

type TimeoutWorker struct {
	ScanFrequency time.Duration
	ExitOnError   bool
}

func (worker *TimeoutWorker) Run(ctx context.Context, coordinator CoordinatorService) error {
	log.Info("expiration worker ready")
	exitReason := ""
	ticker := time.NewTicker(worker.ScanFrequency)
	defer log.Info("expiration worker stopped", zap.String("reason", exitReason))
	for {
		select {
		case <-ticker.C:
			log.Debug("expiration worker start")
			if err := coordinator.UpdateExpired(); err != nil {
				if worker.ExitOnError {
					exitReason = fmt.Sprintf("error: %v", err)
					return err
				} else {
					log.Warn("expiration worker failed", zap.Error(err))
				}
			}

			log.Debug("expiration worker finished")
			continue

		case <-coordinator.WaitForShutdown():
			exitReason = "coordinator shutdown"
			return nil
		}
	}

	return nil
}
