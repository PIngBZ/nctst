package nctst

import (
	"log"
	"net"
	"sync"
)

type OuterConnection struct {
	ID       uint
	ClientID uint
	TunnelID uint
	Addr     string
	Die      chan struct{}

	conn        *net.TCPConn
	receiveChan chan *BufItem
	sendChan    chan *BufItem
	commandChan chan *Command

	dieOnce sync.Once
}

func NewOuterConnection(clientID uint, tunnelID uint, id uint, conn *net.TCPConn, receiveChan chan *BufItem, sendChan chan *BufItem, commandChan chan *Command) *OuterConnection {
	h := &OuterConnection{}
	h.ID = id
	h.ClientID = clientID
	h.TunnelID = tunnelID

	h.Addr = conn.RemoteAddr().String()

	h.conn = conn
	h.receiveChan = receiveChan
	h.sendChan = sendChan
	h.commandChan = commandChan

	h.Die = make(chan struct{})
	once := &sync.Once{}

	go h.receiveLoop(conn, once)
	go h.sendLoop(conn, once)

	log.Printf("new connection %d %d %d\n", clientID, tunnelID, id)
	return h
}

func (h *OuterConnection) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.Die)
		once = true
	})

	if !once {
		return
	}

	if h.conn != nil {
		log.Printf("OuterConnection.Close %s %d %d %d\n", h.conn.RemoteAddr().String(), h.ClientID, h.TunnelID, h.ID)
		h.conn.Close()
	} else {
		log.Printf("OuterConnection.Close nil, %d %d %d\n", h.ClientID, h.TunnelID, h.ID)
	}
}

func (h *OuterConnection) receiveLoop(conn *net.TCPConn, once *sync.Once) {
	defer once.Do(h.Close)

	for {
		buf, err := ReadLBuf(conn)
		if err != nil {
			log.Printf("receiveLoop ReadLenBuf error %d %d %d %+v\n", h.ClientID, h.TunnelID, h.ID, err)
			return
		}

		if IsCommand(buf) {
			select {
			case CommandReceiveChan <- buf:
			case <-h.Die:
				buf.Release()
				return
			default:
				buf.Release()
			}
		} else {
			select {
			case h.receiveChan <- buf:
			case <-h.Die:
				buf.Release()
				return
			}
		}
	}
}

func (h *OuterConnection) sendLoop(conn *net.TCPConn, once *sync.Once) {
	defer once.Do(h.Close)

	for {
		select {
		case <-h.Die:
			return
		case buf := <-h.sendChan:
			if err := WriteUInt(conn, uint32(buf.Size())); err != nil {
				buf.Release()
				log.Printf("sendLoop WriteUInt error: %d %d %d %s %+v\n", h.ClientID, h.TunnelID, h.ID, h.Addr, err)
				return
			}
			_, err := conn.Write(buf.Data())
			buf.Release()
			if err != nil {
				log.Printf("sendLoop WriteUInt error: %d %d %d %s %+v\n", h.ClientID, h.TunnelID, h.ID, h.Addr, err)
				return
			}
		case command := <-h.commandChan:
			if err := SendCommand(conn, command); err != nil {
				log.Printf("sendLoop SendCommand error: %d %d %d %s %+v\n", h.ClientID, h.TunnelID, h.ID, h.Addr, err)
				return
			}
		}
	}
}
