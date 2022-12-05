package core

import "sync/atomic"

var (
	Status           = NewClientStatus()
	publishObservers atomic.Value
)

type ClientStatusStep int

const (
	ClientStatusStep_Init ClientStatusStep = iota
	ClientStatusStep_GetProxyList
	ClientStatusStep_Login
	ClientStatusStep_StartUpstream
	ClientStatusStep_StartMapLocal
	ClientStatusStep_StartLocalService
	ClientStatusStep_CheckingConnection
	ClientStatusStep_Running
	ClientStatusStep_Failed
)

type ClientStatus struct {
	ping int
	stat ClientStatusStep
}

func NewClientStatus() *ClientStatus {
	h := &ClientStatus{}
	publishObservers.Store(make([]chan *ClientStatus, 0))
	return h
}

func AttachStatusObserver(observer chan *ClientStatus) {
	observers := publishObservers.Load().([]chan *ClientStatus)
	for _, item := range observers {
		if item == observer {
			return
		}
	}
	publishObservers.Store(append(observers, observer))
}

func DetachStatusObserver(observer chan *ClientStatus) {
	observers := publishObservers.Load().([]chan *ClientStatus)
	for idx, item := range observers {
		if item == observer {
			publishObservers.Store(append(observers[:idx], observers[idx+1:]...))
			return
		}
	}
}

func (h *ClientStatus) GetStat() ClientStatusStep {
	return h.stat
}

func (h *ClientStatus) GetPing() int {
	return h.ping
}

func (h *ClientStatus) notifyChanged() {
	data := *h
	observers := publishObservers.Load().([]chan *ClientStatus)
	for _, observer := range observers {
		select {
		case observer <- &data:
		default:
		}
	}
}

func (h *ClientStatus) setStat(stat ClientStatusStep) {
	h.stat = stat
	h.notifyChanged()
}

func (h *ClientStatus) setPing(ping int) {
	h.ping = ping
	h.notifyChanged()
}
