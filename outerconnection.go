package nctst

import (
	"io"
	"log"
	"sync"
	"time"
)

type OuterConnection struct {
	ID       uint
	ClientID uint
	TunnelID uint
	Die      chan struct{}

	conn        io.ReadWriteCloser
	receiveChan chan *BufItem
	sendChan    chan *BufItem

	commandChan        chan *Command
	commandReceiveChan chan *BufItem

	dieOnce sync.Once
}

func NewOuterConnection(clientID uint, tunnelID uint, id uint, conn io.ReadWriteCloser,
	receiveChan chan *BufItem, sendChan chan *BufItem,
	commandChan chan *Command, commandReceiveChan chan *BufItem) *OuterConnection {

	h := &OuterConnection{}
	h.ID = id
	h.ClientID = clientID
	h.TunnelID = tunnelID

	h.conn = conn
	h.receiveChan = receiveChan
	h.sendChan = sendChan
	h.commandChan = commandChan
	h.commandReceiveChan = commandReceiveChan

	h.Die = make(chan struct{})
	once := &sync.Once{}

	go h.receiveLoop(conn, once)
	go h.sendLoop(conn, once)

	log.Printf("OuterConnection.New %d %d %d\n", clientID, tunnelID, id)
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

outfor:
	for {
		select {
		case cmd := <-h.commandChan:
			time.Sleep(time.Second)
			SendCommand(h.conn, cmd)
		default:
			break outfor
		}
	}

	if h.conn != nil {
		log.Printf("OuterConnection.Close %d %d %d\n", h.ClientID, h.TunnelID, h.ID)
		h.conn.Close()
	} else {
		log.Printf("OuterConnection.Close nil, %d %d %d\n", h.ClientID, h.TunnelID, h.ID)
	}
}

func (h *OuterConnection) receiveLoop(conn io.ReadWriteCloser, once *sync.Once) {
	defer once.Do(h.Close)

	for {
		buf, err := ReadLBuf(conn)
		if err != nil {
			log.Printf("receiveLoop ReadLenBuf error %d %d %d %+v\n", h.ClientID, h.TunnelID, h.ID, err)
			return
		}

		if IsCommand(buf) {
			select {
			case h.commandReceiveChan <- buf:
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

func (h *OuterConnection) sendLoop(conn io.ReadWriteCloser, once *sync.Once) {
	defer once.Do(h.Close)

	for {
		select {
		case <-h.Die:
			return
		case buf := <-h.sendChan:
			_, err := conn.Write(buf.Data())
			buf.Release()
			if err != nil {
				log.Printf("sendLoop WriteUInt error: %d %d %d %+v\n", h.ClientID, h.TunnelID, h.ID, err)
				return
			}
		case command := <-h.commandChan:
			if err := SendCommand(conn, command); err != nil {
				log.Printf("sendLoop SendCommand error: %d %d %d %+v\n", h.ClientID, h.TunnelID, h.ID, err)
				return
			}
		}
	}
}
