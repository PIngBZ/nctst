package nctst

import "sync"

type OuterTunnel struct {
	ID int

	Ping  int
	Speed int

	commandChan chan *BufItem
	connections sync.Map
}

func NewOuterTunnel(id int, sendChan chan *BufItem) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id

	return h
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

func (h *OuterTunnel) daemon() {
	for {

	}
}
