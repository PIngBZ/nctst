package nctst

import (
	"sync"
	"time"

	"sync/atomic"
)

type OuterTunnel struct {
	ID int

	Ping  int
	Speed int

	alive       atomic.Bool
	connections sync.Map

	OutputChan chan *BufItem
}

func NewOuterTunnel(id int) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id
	h.OutputChan = make(chan *BufItem, 8)
	h.alive.Store(true)

	return h
}

func (h *OuterTunnel) Run() {
	go h.daemon()
}

func (h *OuterTunnel) IsAlive() bool {
	return h.alive.Load()
}

func (h *OuterTunnel) Add(id int, outerConn *OuterConnection) {
	h.Remove(id)
	h.connections.Store(id, outerConn)
}

func (h *OuterTunnel) Remove(id int) {
	if conn, ok := h.connections.LoadAndDelete(id); ok {
		conn.(*OuterConnection).Close()
	}
}

func (h *OuterTunnel) TrySend(buf *BufItem) bool {
	select {
	case h.OutputChan <- buf:
		h.alive.Store(true)
		return true
	default:
		h.alive.Store(false)
		return false
	}
}

func (h *OuterTunnel) daemon() {
	for {
		ticker := time.NewTicker(time.Second * 5)
		select {
		case <-ticker.C:
			h.ping()
		}
	}
}

func (h *OuterTunnel) ping() {
	//cmd := &CommandPing{}

}
