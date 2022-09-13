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

	receiveChan chan *BufItem
	sendChan    chan *BufItem
	outputChan  chan *BufItem

	DirectChan chan *BufItem

	Die chan struct{}

	dieOnce sync.Once
}

func NewOuterTunnel(id uint, clientID uint, receiveChan chan *BufItem, sendChan chan *BufItem) *OuterTunnel {
	h := &OuterTunnel{}
	h.ID = id
	h.ClientID = clientID

	h.connections = make(map[uint]*OuterConnection)

	h.commandSendChan = make(chan *Command, 8)
	h.commandReceiveChan = make(chan *Command, 8)
	h.receiveChan = receiveChan
	h.sendChan = sendChan
	h.outputChan = make(chan *BufItem, 2)
	h.DirectChan = make(chan *BufItem, 8)

	h.Die = make(chan struct{})

	go h.transferLoop()
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

	outer := NewOuterConnection(h.ClientID, h.ID, id, conn, h.receiveChan, h.outputChan, h.commandSendChan)
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

func (h *OuterTunnel) transferLoop() {
	for {
		select {
		case buf := <-h.DirectChan:
			h.outputChan <- buf
		default:
			select {
			case buf := <-h.DirectChan:
				h.outputChan <- buf
			case buf := <-h.sendChan:
				h.outputChan <- buf
			}
		}
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
	if ping.ClientID != h.ClientID || ping.TunnelID != h.ID {
		return
	}

	switch ping.Step {
	case 1:
		ping.Step = 2
		h.sendCommand(&Command{Type: Cmd_ping, Item: ping})
	case 2:
		h.Ping = time.Now().UnixNano()/1e6 - ping.SendTime
		log.Printf("updatePing: client %d tunnel %d id %d ping %d\n", ping.ClientID, ping.TunnelID, ping.ID, h.Ping)
	}
}
