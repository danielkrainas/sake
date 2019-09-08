package service

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/cskr/pubsub"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/golang/protobuf/proto"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
)

type ReplyGroup map[string]func(req *protocol.Reply) error

type RawGroup map[string]func(rawMessage []byte) error

type HubConnector interface {
	CancelAll() error
	CancelGroup(groupKey interface{}) error
	SubReply(groupKey interface{}, finalizer func(), replyGroup ReplyGroup) error
	SubGroup(groupKey interface{}, group RawGroup) error
	Pub(topic string, req *protocol.Request) error
	//Sub(topic string, handler func(rawMessage []byte) error) error
}

type DebugHub struct {
	pubsub  *pubsub.PubSub
	chMutex sync.Mutex
	quits   map[chan interface{}]chan struct{}
	groups  map[interface{}][]chan interface{}
}

var _ HubConnector = &DebugHub{}

func NewDebugHub() *DebugHub {
	return &DebugHub{
		pubsub: pubsub.New(0),
		quits:  make(map[chan interface{}]chan struct{}),
		groups: make(map[interface{}][]chan interface{}),
	}
}

func (hub *DebugHub) CancelAll() error {
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	for ch, quitCh := range hub.quits {
		quitCh <- struct{}{}
		close(quitCh)
		close(ch)
	}

	return nil
}

func (hub *DebugHub) group(groupKey interface{}) []chan interface{} {
	group, ok := hub.groups[groupKey]
	if !ok {
		group = make([]chan interface{}, 0)
	}

	return group
}

func (hub *DebugHub) CancelGroup(groupKey interface{}) error {
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	group := hub.group(groupKey)
	delete(hub.groups, groupKey)
	for _, ch := range group {
		quitCh, ok := hub.quits[ch]
		if ok {
			quitCh <- struct{}{}
			close(quitCh)
		}

		close(ch)
	}

	return nil
}

func (hub *DebugHub) SubReply(groupKey interface{}, finalizer func(), replyGroup ReplyGroup) error {
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	group, ok := hub.groups[groupKey]
	if ok && len(group) > 0 {
		return errors.New("group already exists")
	}

	defer func() {
		if err := recover(); err != nil {
			log.Error("recovering failed subscription action", zap.Any("error", err))
			return
		}

		defer func() {
			if err := recover(); err != nil {
				log.Error("recovering failed finalizer", zap.Any("error", err))
			}
		}()

		if finalizer != nil {
			finalizer()
		}
	}()

	group = make([]chan interface{}, 0)
	for topic, handler := range replyGroup {
		ch := hub.pubsub.Sub(topic)
		hub.groups[groupKey] = group
		quitCh := make(chan struct{})
		hub.quits[ch] = quitCh
		go hub.subscriptionListener(topic, ch, quitCh, hub.replyHandler(handler))
	}

	return nil
}

func (hub *DebugHub) Pub(topic string, req *protocol.Request) error {
	data, _ := MarshalRequest(req)
	hub.pubsub.Pub(data, topic)
	return nil
}

func (hub *DebugHub) PubReply(topic string, reply *protocol.Reply) error {
	data, _ := MarshalReply(reply)
	hub.pubsub.Pub(data, topic)
	return nil
}

func (hub *DebugHub) PubRaw(topic string, data []byte) error {
	hub.pubsub.Pub(data, topic)
	return nil
}

func (hub *DebugHub) Sub(topic string, handler func(data []byte) error) error {
	ch := hub.pubsub.Sub(topic)
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	quitCh := make(chan struct{})
	hub.quits[ch] = quitCh
	go hub.subscriptionListener(topic, ch, quitCh, handler)
	return nil
}

func (hub *DebugHub) SubGroup(groupKey interface{}, handlerGroup RawGroup) error {
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	group, ok := hub.groups[groupKey]
	if ok && len(group) > 0 {
		return errors.New("group already exists")
	}

	defer func() {
		if err := recover(); err != nil {
			log.Error("recovering failed subscription action", zap.Any("error", err))
		}
	}()

	group = make([]chan interface{}, 0)
	for topic, handler := range handlerGroup {
		ch := hub.pubsub.Sub(topic)
		hub.groups[groupKey] = group
		quitCh := make(chan struct{})
		hub.quits[ch] = quitCh
		go hub.subscriptionListener(topic, ch, quitCh, handler)
	}

	return nil
}

func (hub *DebugHub) SubReq(topic string, handler func(req *protocol.Request) error) error {
	ch := hub.pubsub.Sub(topic)
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	quitCh := make(chan struct{})
	hub.quits[ch] = quitCh
	go hub.subscriptionListener(topic, ch, quitCh, hub.requestHandler(handler))
	return nil
}

func (hub *DebugHub) requestHandler(handler func(request *protocol.Request) error) func(data []byte) error {
	return func(data []byte) error {
		request, _ := UnmarshalRequest(data)
		return handler(request)
	}
}

func (hub *DebugHub) replyHandler(handler func(reply *protocol.Reply) error) func(data []byte) error {
	return func(data []byte) error {
		reply, _ := UnmarshalReply(data)
		return handler(reply)
	}
}

func (hub *DebugHub) subscriptionListener(topic string, ch chan interface{}, quitCh chan struct{}, handler func(data []byte) error) {
	for {
		select {
		case <-quitCh:
			return
		case data := <-ch:
			if err := handler(data.([]byte)); err != nil {
				log.Warn("subscriber failed", zap.String("topic", topic))
				ch <- data
			}
		}
	}
}

type StanHub struct {
	DurableName string
	Conn        stan.Conn
	groups      map[interface{}][]stan.Subscription
	groupMutex  sync.Mutex

	subMutex sync.Mutex
	subs     []stan.Subscription
}

var _ HubConnector = &StanHub{}

func NewStanHub(clusterID string, natsURL string, clientID string, durableName string) (*StanHub, error) {
	conn, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return &StanHub{
		Conn:        conn,
		DurableName: durableName,
		groups:      make(map[interface{}][]stan.Subscription),
		subs:        make([]stan.Subscription, 0),
	}, nil
}

func (hub *StanHub) CancelAll() error {
	hub.subMutex.Lock()
	hub.groupMutex.Lock()
	defer hub.subMutex.Unlock()
	defer hub.groupMutex.Unlock()
	for _, sub := range hub.subs {
		if err := sub.Close(); err != nil {
			return err
		}
	}

	hub.subs = make([]stan.Subscription, 0)
	for _, subs := range hub.groups {
		for _, sub := range subs {
			if err := sub.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (hub *StanHub) CancelGroup(groupKey interface{}) error {
	hub.groupMutex.Lock()
	defer hub.groupMutex.Unlock()
	log.Debug("stan cancel group", zap.Any("group", groupKey))
	group, ok := hub.groups[groupKey]
	if !ok {
		return nil
	}

	for _, sub := range group {
		if err := sub.Unsubscribe(); err != nil {
			return err
		}
	}

	delete(hub.groups, groupKey)
	return nil
}

func (hub *StanHub) SubReply(groupKey interface{}, finalizer func(), replyGroup ReplyGroup) error {
	hub.groupMutex.Lock()
	defer hub.groupMutex.Unlock()
	groupOncer := sync.Once{}
	subs, ok := hub.groups[groupKey]
	if ok && len(subs) > 0 {
		return fmt.Errorf("group already exists")
	}

	subs = make([]stan.Subscription, 0)
	for topic, handler := range replyGroup {
		log.Debug("stan new reply subscriber", zap.String("topic", topic), zap.Any("rgroup", groupKey))
		sub, err := hub.Conn.Subscribe(
			topic,
			hub.replyHandler(&groupOncer, finalizer, handler),
			stan.DurableName(hub.DurableName),
			stan.MaxInflight(1),
			stan.SetManualAckMode(),
		)

		if err != nil {
			return err
		}

		subs = append(subs, sub)
	}

	hub.groups[groupKey] = subs
	return nil
}

func (hub *StanHub) Pub(topic string, req *protocol.Request) error {
	data, err := MarshalRequest(req)
	if err != nil {
		return err
	}

	if err := hub.Conn.Publish(topic, data); err != nil {
		return err
	}

	return nil
}

func (hub *StanHub) Sub(topic string, handler func(rawMessage []byte) error) error {
	hub.subMutex.Lock()
	defer hub.subMutex.Unlock()
	sub, err := hub.Conn.Subscribe(
		topic,
		hub.rawHandler(handler),
		stan.DurableName(hub.DurableName),
		stan.MaxInflight(1),
		stan.SetManualAckMode(),
	)

	if err != nil {
		return err
	}

	hub.subs = append(hub.subs, sub)
	log.Debug("stan new raw subscriber", zap.String("topic", topic))
	return nil
}

func (hub *StanHub) SubGroup(groupKey interface{}, handlerGroup RawGroup) error {
	hub.groupMutex.Lock()
	defer hub.groupMutex.Unlock()
	subs, ok := hub.groups[groupKey]
	if ok && len(subs) > 0 {
		return fmt.Errorf("group already exists")
	}

	subs = make([]stan.Subscription, 0)
	for topic, handler := range handlerGroup {
		log.Debug("stan new raw subscriber", zap.String("topic", topic), zap.Any("group", groupKey))
		sub, err := hub.Conn.Subscribe(
			topic,
			hub.rawHandler(handler),
			stan.DurableName(hub.DurableName),
			stan.MaxInflight(1),
			stan.SetManualAckMode(),
		)

		if err != nil {
			return err
		}

		subs = append(subs, sub)
		hub.groups[groupKey] = subs
	}

	return nil
}

func (hub *StanHub) rawHandler(handler func(rawMessage []byte) error) stan.MsgHandler {
	return stan.MsgHandler(func(msg *stan.Msg) {
		if err := handler(msg.Data); err == nil {
			if err := msg.Ack(); err != nil {
				log.Error("stan ack failure", zap.Error(err))
				return
			}

			log.Debug("stan ack")
		}
	})
}

func (hub *StanHub) replyHandler(oncer *sync.Once, finalizer func(), handler func(reply *protocol.Reply) error) stan.MsgHandler {
	return stan.MsgHandler(func(msg *stan.Msg) {
		oncer.Do(func() {
			reply, err := UnmarshalReply(msg.Data)
			if err != nil {
				log.Error("stan reply unmarhsal failure", zap.Error(err))
				return
			}

			if err := handler(reply); err != nil {
				log.Error("stan handler failure", zap.Error(err))
				return
			}

			if err := msg.Ack(); err != nil {
				log.Error("stan ack failure", zap.Error(err))
				return
			}

			log.Debug("stan ack")
			finalizer()
		})
	})
}

func logCloser(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Error("close failed", zap.Error(err))
	}
}

func UnmarshalReply(data []byte) (*protocol.Reply, error) {
	reply := &protocol.Reply{}
	if err := proto.Unmarshal(data, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func MarshalReply(reply *protocol.Reply) ([]byte, error) {
	data, err := proto.Marshal(reply)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func UnmarshalRequest(data []byte) (*protocol.Request, error) {
	req := &protocol.Request{}
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, err
	}

	return req, nil
}

func MarshalRequest(req *protocol.Request) ([]byte, error) {
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return data, nil
}
