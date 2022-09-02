package nctst

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"sync/atomic"
)

type OuterTunnel struct {
	ID int

	Ping  int64
	Speed int

	alive       atomic.Bool
	connections sync.Map
	nextPingID  int

	OutputChan      chan *BufItem
	CommandSendChan chan *Command

	commandReceiveChan chan *Command

	Die chan struct{}

	dieOnce sync.Once
}

func NewOuterTunnel(id int) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id
	h.OutputChan = make(chan *BufItem, 32)
	h.CommandSendChan = make(chan *Command, 8)
	h.commandReceiveChan = make(chan *Command, 8)
	h.Die = make(chan struct{})
	h.alive.Store(true)

	log.Printf("Tunnel created %d", id)
	return h
}

func (h *OuterTunnel) Run() {
	AttachCommandObserver(h.commandReceiveChan)
	go h.daemon()
	h.startPing()
}

func (h *OuterTunnel) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.Die)
		once = true
	})

	if !once {
		return
	}

	RemoveCommandObserver(h.commandReceiveChan)

	h.connections.Range(func(key, value any) bool {
		value.(*OuterConnection).Close()
		return true
	})
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
	case <-time.After(time.Millisecond * 20):
		h.alive.Store(false)
		return false
	}
}

func (h *OuterTunnel) daemon() {
	for {
		ticker := time.NewTicker(time.Second * time.Duration(rand.Intn(5)+5))
		select {
		case <-h.Die:
			return
		case <-ticker.C:
			h.startPing()
		case command := <-h.commandReceiveChan:
			h.onReceiveCommand(command)
		}
	}
}

func (h *OuterTunnel) startPing() {
	cmd := &CommandPing{}
	cmd.Step = 1
	cmd.TunnelID = h.ID
	cmd.ID = h.nextPingID
	h.nextPingID++
	cmd.SendTime = time.Now().UnixNano() / 1e6

	log.Printf("Ping: %d %d", cmd.TunnelID, cmd.ID)
	h.sendCommand(&Command{Type: Cmd_ping, Item: cmd})
}

func (h *OuterTunnel) sendCommand(command *Command) {
	select {
	case h.CommandSendChan <- command:
	default:
	}
}

func (h *OuterTunnel) onReceiveCommand(command *Command) {
	h.alive.Store(true)

	switch command.Type {
	case Cmd_ping:
		h.onReceivePing(command.Item.(*CommandPing))
	}
}

func (h *OuterTunnel) onReceivePing(ping *CommandPing) {
	switch ping.Step {
	case 1:
		ping.Step = 2
		h.sendCommand(&Command{Type: Cmd_ping, Item: ping})
	case 2:
		h.Ping = (int64(Min(int(h.Ping), 5000)) + time.Now().UnixNano()/1e6 - ping.SendTime) / 2
		log.Printf("updatePing: tunnel%d %d %d", ping.TunnelID, ping.ID, h.Ping)
	}
}
