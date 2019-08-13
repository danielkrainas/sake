package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
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

		hub := service.NewDebugHub()
		coordinator := service.NewCoordinator(rootContext, hub)

		coordinator.Register(service.Workflows[1])

		simulateFailure := true
		wg := registerTestListeners(hub, simulateFailure)

		// kick off saga
		hub.PubRaw("init-start", []byte{})

		wg.Wait()
	},
}

func registerTestListeners(hub *service.DebugHub, simulateFailure bool) *sync.WaitGroup {
	hub.SubReq("start", func(req *protocol.Request) {
		fmt.Println("coordinator called start")
		fmt.Println("replying success")
		reply := &protocol.Reply{
			NewData: []byte("started"),
		}

		go hub.PubReply(req.SuccessReplyTopic, reply)
	})

	wg := sync.WaitGroup{}
	wg.Add(2)
	if simulateFailure {
		wg.Add(2)
	}

	hub.SubReq("cancel-start", func(req *protocol.Request) {
		fmt.Println("coordinator rollback start")
		fmt.Println("replying success")

		go func() {
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
			wg.Done()
		}()
	})

	hub.SubReq("middle", func(req *protocol.Request) {
		fmt.Println("coordinator called middle")
		fmt.Println("replying success")

		go func() {
			wg.Done()
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
		}()
	})

	hub.SubReq("end", func(req *protocol.Request) {
		fmt.Println("coordinator called end")
		if simulateFailure {
			fmt.Println("replying failed, should rollback")
		} else {
			fmt.Println("replying success")
		}

		go func() {
			wg.Done()
			if simulateFailure {
				hub.PubReply(req.FailureReplyTopic, &protocol.Reply{})
			} else {
				hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
			}
		}()
	})

	hub.SubReq("cancel-middle", func(req *protocol.Request) {
		fmt.Println("coordinator rollback middle")
		fmt.Println("replying success")

		go func() {
			wg.Done()
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
		}()
	})

	return &wg
}
