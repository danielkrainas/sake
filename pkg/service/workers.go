package service

import (
	"context"
	"time"
)

type TimeoutWorker struct {
	ScanFrequency time.Duration
	ExitOnError   bool
}

func (worker *TimeoutWorker) Run(ctx context.Context, coordinator CoordinatorService) error {
	for {
		err := coordinator.UpdateExpired()
		if err != nil {
			if worker.ExitOnError {
				return err
			}
		}

		time.Sleep(worker.ScanFrequency)
	}
}
