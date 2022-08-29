package nctst

import "sync"

type OuterTunnel struct {
	ID int

	Ping  int
	Speed int

	connections sync.Map

	SendChan chan *BufItem
}

func NewOuterTunnel(id int, sendChan chan *BufItem) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id
	h.SendChan = make(chan *BufItem, 8)

	go h.daemon(sendChan)

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

func (h *OuterTunnel) daemon(sendChan chan *BufItem) {
	for buf := range sendChan {
		select {
		case h.SendChan <- buf:
		default:
		}
	}
}
