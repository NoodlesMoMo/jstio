package model

import (
	"errors"
	"sync"
)

var (
	notifierInstance_ *EventNotifier
	notifierOnce      = sync.Once{}
)

var (
	ErrBusOverFlow = errors.New("event_notifier: buffer too many")
)

type Event = int

const (
	// None Event
	EventNone = iota

	// Application CURD
	EventApplicationCreate
	EventApplicationUpdate
	EventApplicationDelete

	// route CURD
	EventResRouteCreate
	EventResRouteUpdate
	EventResRouteDelete

	// listener CURD
	EventResListenerCreate
	EventResListenerUpdate
	EventResListenerDelete

	// endpoint CURD
	EventResEndpointCreate
	EventResEndpointUpdate
	EventResEndpointDelete

	// cluster CURD
	EventResClusterCreate
	EventResClusterUpdate
	EventResClusterDelete

	EventResEndpointReFetch
)

type EventData struct {
	E    Event
	Data interface{}
}

type EventNotifier struct {
	ch chan EventData
}

func NewEventNotifier() *EventNotifier {
	return &EventNotifier{
		ch: make(chan EventData, 1024),
	}
}

func (en *EventNotifier) Push(e Event, data interface{}) error {
	select {
	case en.ch <- EventData{e, data}:
	default:
		return ErrBusOverFlow
	}

	return nil
}

func (en *EventNotifier) Bus() <-chan EventData {
	return en.ch
}

func GetEventNotifier() *EventNotifier {
	if notifierInstance_ != nil {
		return notifierInstance_
	}

	notifierOnce.Do(func() {
		notifierInstance_ = NewEventNotifier()
	})

	return notifierInstance_
}
