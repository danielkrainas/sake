package main

import (
	"sync"

	sake "github.com/danielkrainas/sake/pkg/service"
	sakeprotocol "github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/danielkrainas/sake/pkg/util/log"
	stan "github.com/nats-io/go-nats-streaming"

	"go.uber.org/zap"
)

const (
	ClusterID   = "test-cluster"
	ClientID    = "sandbox-client"
	DurableName = "sandbox-client-durable"
	NatsAddr    = "nats://0.0.0.0"
)

func main() {
	conn, err := stan.Connect(ClusterID, ClientID, stan.NatsURL(NatsAddr))
	if err != nil {
		log.Fatal("stan failed", zap.Error(err))
	}

	wg := addSagaListeners(conn)
	conn.Publish("init-start", []byte{})
	log.Debug("waiting for finish")
	wg.Wait()
}

func subscribe(conn stan.Conn, topic string, handler func(req *sakeprotocol.Request)) {
	_, err := conn.Subscribe(
		topic,
		stan.MsgHandler(func(msg *stan.Msg) {
			req, _ := sake.UnmarshalRequest(msg.Data)
			handler(req)
		}),
		//stan.DurableName(DurableName),
		//stan.MaxInflight(1),
		//stan.SetManualAckMode(),
	)

	if err != nil {
		log.Fatal("failed add subsriber", zap.String("topic", topic))
	}
}

func marshalReply(reply *sakeprotocol.Reply) []byte {
	data, _ := sake.MarshalReply(reply)
	return data
}

func addSagaListeners(conn stan.Conn) sync.WaitGroup {
	simulateFailure := true
	wg := sync.WaitGroup{}
	wg.Add(2)
	if simulateFailure {
		wg.Add(2)
	}

	subscribe(conn, "start", func(req *sakeprotocol.Request) {
		log.Debug("coordinator called start")
		log.Debug("replying success")
		conn.Publish(req.SuccessReplyTopic, marshalReply(&sakeprotocol.Reply{
			NewData: []byte("started"),
		}))
	})

	subscribe(conn, "cancel-start", func(req *sakeprotocol.Request) {
		log.Debug("coordinator rollback start")
		log.Debug("replying success")
		conn.Publish(req.SuccessReplyTopic, marshalReply(&sakeprotocol.Reply{}))
		wg.Done()
	})

	subscribe(conn, "middle", func(req *sakeprotocol.Request) {
		log.Debug("coordinator called middle")
		log.Debug("replying success")
		conn.Publish(req.SuccessReplyTopic, marshalReply(&sakeprotocol.Reply{}))
		wg.Done()
	})

	subscribe(conn, "end", func(req *sakeprotocol.Request) {
		log.Debug("coordinator called end")
		if simulateFailure {
			log.Debug("replying failed, should rollback")
		} else {
			log.Debug("replying success")
		}

		replyData := marshalReply(&sakeprotocol.Reply{})
		if simulateFailure {
			conn.Publish(req.FailureReplyTopic, replyData)
		} else {
			conn.Publish(req.SuccessReplyTopic, replyData)
		}

		wg.Done()
	})

	subscribe(conn, "cancel-middle", func(req *sakeprotocol.Request) {
		log.Debug("coordinator rollback middle")
		log.Debug("replying success")

		conn.Publish(req.SuccessReplyTopic, marshalReply(&sakeprotocol.Reply{}))
		wg.Done()
	})

	return wg
}
