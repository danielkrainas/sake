package service

type HubConnector interface {
	CancelGroup(groupKey interface{}) error
	GroupSubscribe(groupKey interface{}, address string, handler func(e *Envelope))
	Publish(address string, e *Envelope)
	Subscribe(address string, handler func(e *Envelope))
}

type TestHub struct {
	groups        map[interface{}]map[string][]func(e *Envelope)
	subscriptions map[string][]func(e *Envelope)
}

func NewTestHub() *TestHub {
	return &TestHub{
		groups:        make(map[interface{}]map[string][]func(*Envelope), 0),
		subscriptions: make(map[string][]func(*Envelope)),
	}
}

func (hub *TestHub) CancelGroup(groupKey interface{}) error {
	delete(hub.groups, groupKey)
	return nil
}

func (hub *TestHub) Subscribe(address string, handler func(e *Envelope)) {
	subscribers, ok := hub.subscriptions[address]
	if !ok || subscribers == nil {
		subscribers = make([]func(*Envelope), 0)
	}

	subscribers = append(subscribers, handler)
	hub.subscriptions[address] = subscribers
}

func (hub *TestHub) GroupSubscribe(groupKey interface{}, address string, handler func(e *Envelope)) {
	group, ok := hub.groups[groupKey]
	if !ok || group == nil {
		group = make(map[string][]func(*Envelope))
	}

	subscribers, ok := group[address]
	if !ok || subscribers == nil {
		subscribers = make([]func(*Envelope), 0)
	}

	subscribers = append(subscribers, handler)
	group[address] = subscribers
	hub.groups[groupKey] = group
}

func (hub *TestHub) Publish(address string, e *Envelope) {
	//fmt.Println("[hub] publish", address)
	for _, subscriberGroup := range hub.groups {
		subscribers, ok := subscriberGroup[address]
		if ok {
			for _, subscriber := range subscribers {
				subscriber(e)
			}
		}
	}

	if subscribers, ok := hub.subscriptions[address]; ok {
		for _, subscriber := range subscribers {
			subscriber(e)
		}
	}

	//fmt.Println("[hub] done", address)
}
