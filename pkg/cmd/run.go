package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/danielkrainas/sake/pkg/util/log"
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
			log.Fatal(err.Error())
			return
		}*/

		storage, err := service.NewDebugStorage([]*service.Workflow{service.Workflows[1]}, nil)
		if err != nil {
			log.Fatal("storage init failed", zap.Error(err))
		}

		var cache service.CacheService
		cache, err = service.NewInMemoryCache()
		if err != nil {
			log.Fatal("cache init failed", zap.Error(err))
		}

		cache = &service.WriteThruCache{
			CacheService: cache,
			Storage:      storage,
		}

		hub := service.NewDebugHub()
		coordinator, err := service.NewCoordinator(rootContext, hub, cache, storage)
		if err != nil {
			log.Fatal("coordinator init failed", zap.Error(err))
		}

		//coordinator.Register(service.Workflows[1])

		simulateFailure := false
		wg := registerTestListeners(hub, coordinator, simulateFailure)

		// kick off saga
		hub.PubRaw("init-start", []byte{})

		wg.Wait()

		log.Debug("cache contents:")
		cache.TransactAll(rootContext, func(trx *service.Transaction) (*service.Transaction, error) {
			fmt.Printf(" - %s\n", trx.ID)
			return nil, nil
		})
	},
}

func registerTestListeners(hub *service.DebugHub, coordinator *service.Coordinator, simulateFailure bool) *sync.WaitGroup {
	hub.SubReq("start", func(req *protocol.Request) error {
		log.Debug("coordinator called start")
		log.Debug("replying success")
		reply := &protocol.Reply{
			NewData: []byte("started"),
		}

		go hub.PubReply(req.SuccessReplyTopic, reply)
		return nil
	})

	wg := sync.WaitGroup{}
	wg.Add(2)
	if simulateFailure {
		wg.Add(2)
	}

	hub.SubReq("cancel-start", func(req *protocol.Request) error {
		log.Debug("coordinator rollback start")
		log.Debug("replying success")

		go func() {
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
			wg.Done()
		}()

		return nil
	})

	hub.SubReq("middle", func(req *protocol.Request) error {
		log.Debug("coordinator called middle")
		log.Debug("replying success")

		go func() {
			wg.Done()
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
		}()

		return nil
	})

	hub.SubReq("end", func(req *protocol.Request) error {
		log.Debug("coordinator called end")
		if simulateFailure {
			log.Debug("replying failed, should rollback")
		} else {
			log.Debug("replying success")
		}

		go func() {
			wg.Done()
			if simulateFailure {
				hub.PubReply(req.FailureReplyTopic, &protocol.Reply{})
			} else {
				hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
			}
		}()

		return nil
	})

	hub.SubReq("cancel-middle", func(req *protocol.Request) error {
		log.Debug("coordinator rollback middle")
		log.Debug("replying success")

		go func() {
			wg.Done()
			hub.PubReply(req.SuccessReplyTopic, &protocol.Reply{})
		}()

		return nil
	})

	return &wg
}
