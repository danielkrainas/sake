package service

import (
	"sync"

	"github.com/cskr/pubsub"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/golang/protobuf/proto"
)

type HubConnector interface {
	CancelAll() error
	CancelGroup(groupKey interface{}) error
	SubReply(groupKey interface{}, topic string, handler func(req *protocol.Reply))
	Pub(topic string, req *protocol.Request)
	Sub(topic string, handler func(rawMessage []byte))
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

func (hub *DebugHub) SubReply(groupKey interface{}, topic string, handler func(req *protocol.Reply)) {
	ch := hub.pubsub.Sub(topic)
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	group := hub.group(groupKey)
	hub.groups[groupKey] = group
	quitCh := make(chan struct{})
	hub.quits[ch] = quitCh
	go hub.subscriptionListener(ch, quitCh, hub.replyHandler(handler))
}

func (hub *DebugHub) Pub(topic string, req *protocol.Request) {
	data, _ := MarshalRequest(req)
	hub.pubsub.Pub(data, topic)
}

func (hub *DebugHub) PubReply(topic string, reply *protocol.Reply) {
	data, _ := MarshalReply(reply)
	hub.pubsub.Pub(data, topic)
}

func (hub *DebugHub) PubRaw(topic string, data []byte) {
	hub.pubsub.Pub(data, topic)
}

func (hub *DebugHub) Sub(topic string, handler func(data []byte)) {
	ch := hub.pubsub.Sub(topic)
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	quitCh := make(chan struct{})
	hub.quits[ch] = quitCh
	go hub.subscriptionListener(ch, quitCh, handler)
}

func (hub *DebugHub) SubReq(topic string, handler func(req *protocol.Request)) {
	ch := hub.pubsub.Sub(topic)
	hub.chMutex.Lock()
	defer hub.chMutex.Unlock()
	quitCh := make(chan struct{})
	hub.quits[ch] = quitCh
	go hub.subscriptionListener(ch, quitCh, hub.requestHandler(handler))
}

func (hub *DebugHub) requestHandler(handler func(request *protocol.Request)) func(data []byte) {
	return func(data []byte) {
		request, _ := UnmarshalRequest(data)
		handler(request)
	}
}

func (hub *DebugHub) replyHandler(handler func(reply *protocol.Reply)) func(data []byte) {
	return func(data []byte) {
		reply, _ := UnmarshalReply(data)
		handler(reply)
	}
}

func (hub *DebugHub) subscriptionListener(ch chan interface{}, quitCh chan struct{}, handler func(data []byte)) {
	for {
		select {
		case <-quitCh:
			return
		case data := <-ch:
			handler(data.([]byte))
		}
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