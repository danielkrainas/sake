package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/danielkrainas/sake/pkg/service"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run the worker daemon",
	Long:  "run the worker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		/*config, err := service.ResolveConfig(configPath)
		if err != nil {
			log.Fatal(rootContext, err.Error())
			return
		}*/

		hub := service.NewTestHub()
		coordinator := service.NewCoordinator(rootContext, hub)

		coordinator.Register(service.Workflows[1])

		simulateFailure := true
		wg := registerTestListeners(hub, simulateFailure)

		// kick off saga
		hub.Publish("init-start", &service.Envelope{})

		wg.Wait()
	},
}

func registerTestListeners(hub service.HubConnector, simulateFailure bool) *sync.WaitGroup {
	hub.Subscribe("start", func(e *service.Envelope) {
		fmt.Println("coordinator called start")
		fmt.Println("replying success")
		ne := &service.Envelope{
			TransactionID: e.TransactionID,
			Data:          []byte("started"),
		}

		go hub.Publish(e.SuccessReplyAddress, ne)
	})

	wg := sync.WaitGroup{}
	wg.Add(2)
	if simulateFailure {
		wg.Add(2)
	}

	hub.Subscribe("cancel-start", func(e *service.Envelope) {
		fmt.Println("coordinator rollback start")
		fmt.Println("replying success")

		go func() {
			hub.Publish(e.SuccessReplyAddress, e)
			wg.Done()
		}()
	})

	hub.Subscribe("middle", func(e *service.Envelope) {
		fmt.Println("coordinator called middle")
		fmt.Println("replying success")

		go func() {
			wg.Done()
			hub.Publish(e.SuccessReplyAddress, e)
		}()
	})

	hub.Subscribe("end", func(e *service.Envelope) {
		fmt.Println("coordinator called end")
		if simulateFailure {
			fmt.Println("replying failed, should rollback")
		} else {
			fmt.Println("replying success")
		}

		go func() {
			wg.Done()
			if simulateFailure {
				hub.Publish(e.FailureReplyAddress, e)
			} else {
				hub.Publish(e.SuccessReplyAddress, e)
			}
		}()
	})

	hub.Subscribe("cancel-middle", func(e *service.Envelope) {
		fmt.Println("coordinator rollback middle")
		fmt.Println("replying success")

		go func() {
			wg.Done()
			hub.Publish(e.SuccessReplyAddress, e)
		}()
	})

	return &wg
}
