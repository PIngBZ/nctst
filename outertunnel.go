package nctst

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

type OuterTunnel struct {
	ID       uint
	ClientID uint

	Ping  int64
	Speed int

	connections       map[uint]*OuterConnection
	connectionsLocker sync.Mutex

	nextPingID uint

	commandSendChan    chan *Command
	commandReceiveChan chan *Command

	inputChan  chan *BufItem
	outputChan chan *BufItem

	Die chan struct{}

	dieOnce sync.Once
}

func NewOuterTunnel(id uint, clientID uint, inputChan chan *BufItem, outputChan chan *BufItem) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id
	h.ClientID = clientID

	h.connections = make(map[uint]*OuterConnection)

	h.commandSendChan = make(chan *Command, 8)
	h.commandReceiveChan = make(chan *Command, 8)
	h.inputChan = inputChan
	h.outputChan = outputChan

	h.Die = make(chan struct{})

	go h.daemon()
	h.startPing()

	AttachCommandObserver(h.commandReceiveChan)

	log.Printf("new tunnel %d %d\n", clientID, id)
	return h
}

func (h *OuterTunnel) AddConn(conn *net.TCPConn, id uint) (dieSignal chan struct{}) {
	h.Remove(id)

	h.connectionsLocker.Lock()
	defer h.connectionsLocker.Unlock()

	outer := NewOuterConnection(h.ClientID, h.ID, id, conn, h.inputChan, h.outputChan, h.commandSendChan)
	h.connections[id] = outer

	return outer.Die
}

func (h *OuterTunnel) Remove(id uint) {
	h.connectionsLocker.Lock()
	defer h.connectionsLocker.Unlock()

	if outer, ok := h.connections[id]; ok {
		delete(h.connections, id)
		outer.Close()
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
	cmd.ClientID = h.ClientID
	cmd.TunnelID = h.ID
	cmd.ID = h.nextPingID
	h.nextPingID++
	cmd.SendTime = time.Now().UnixNano() / 1e6

	log.Printf("Ping: %d %d %d\n", cmd.ClientID, cmd.TunnelID, cmd.ID)
	h.sendCommand(&Command{Type: Cmd_ping, Item: cmd})
}

func (h *OuterTunnel) sendCommand(command *Command) {
	select {
	case h.commandSendChan <- command:
	default:
	}
}

func (h *OuterTunnel) onReceiveCommand(command *Command) {
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
		log.Printf("updatePing: %d %d %d %d\n", ping.ClientID, ping.TunnelID, ping.ID, h.Ping)
	}
}
