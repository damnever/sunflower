package pubsub

import (
	"sync"
)

type EventType byte

const (
	chSize                    = 16
	EventOpenTunnel EventType = iota
	EventCloseTunnel
	EventRejectAgent
)

type Event struct {
	Type       EventType
	TunnelHash string
}

type Publisher interface {
	Pub(ahash string, evts ...*Event)
}

type Subscriber interface {
	Sub(ahash string) <-chan *Event
	Unsub(ahash string)
}

type PubSub struct {
	sync.RWMutex
	registry map[string]chan *Event
}

func New() *PubSub {
	return &PubSub{
		registry: map[string]chan *Event{},
	}
}

func (ps *PubSub) Pub(ahash string, evts ...*Event) {
	ps.RLock()
	ch, in := ps.registry[ahash]
	if !in {
		ps.RUnlock()
		return
	}
	ps.RUnlock()

	for _, evt := range evts {
		ch <- evt
	}
}

func (ps *PubSub) Sub(ahash string) <-chan *Event {
	ps.RLock()
	if ch, in := ps.registry[ahash]; in {
		ps.RUnlock()
		return ch
	}
	ps.RUnlock()

	ps.Lock()
	defer ps.Unlock()
	if ch, in := ps.registry[ahash]; in { // double check
		return ch
	}
	ch := make(chan *Event, 16)
	ps.registry[ahash] = ch
	return ch
}

func (ps *PubSub) Unsub(ahash string) {
	ps.Lock()
	if ch, in := ps.registry[ahash]; in {
		close(ch)
		delete(ps.registry, ahash)
	}
	ps.Unlock()
}
