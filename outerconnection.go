package nctst

import (
	"log"
	"net"
	"sync"
)

type OuterConnection struct {
	TunnelID int
	ID       int
	Addr     string
	Die      chan struct{}

	conn        *net.TCPConn
	receiveChan chan *BufItem
	sendChan    chan *BufItem

	dieOnce sync.Once
}

func NewOuterConnection(tunnelID int, id int, conn *net.TCPConn, receiveChan chan *BufItem, sendChan chan *BufItem) *OuterConnection {
	h := &OuterConnection{}
	h.TunnelID = tunnelID
	h.ID = id
	h.Addr = conn.RemoteAddr().String()

	h.conn = conn
	h.receiveChan = receiveChan
	h.sendChan = sendChan

	h.Die = make(chan struct{})

	once := &sync.Once{}
	go h.receiveLoop(conn, once)
	go h.sendLoop(conn, once)

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
		log.Printf("OuterConnection.Close %s\n", h.conn.RemoteAddr().String())
		h.conn.Close()
	} else {
		log.Println("OuterConnection.Close nil")
	}
}

func (h *OuterConnection) receiveLoop(conn *net.TCPConn, once *sync.Once) {
	defer once.Do(h.Close)

	for {
		l, err := ReadUInt(h.conn)
		if err != nil {
			log.Printf("%s %+v\n", h.Addr, err)
			return
		}

		if l > KCP_DATA_BUF_SIZE+uint32(Cmd_max) {
			log.Printf("%s error len %d\n", h.Addr, l)
			return
		}

		if l > KCP_DATA_BUF_SIZE {
			l, err = ReadUInt(h.conn)
			if err != nil {
				log.Printf("%s %+v\n", h.Addr, err)
				return
			}
			if l > 1024 {
				log.Printf("%s error command len %d\n", h.Addr, l)
				return
			}
		}

		buf := DataBufPool.Get()
		if _, err = buf.ReadNFromReader(conn, int(l)); err != nil {
			buf.Release()
			log.Printf("%s %+v\n", h.Addr, err)
			return
		}

		if l > KCP_DATA_BUF_SIZE {
			select {
			case CommandReceiveChan <- buf:
			case <-h.Die:
				buf.Release()
				return
			default:
				buf.Release()
			}
			return
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
		var buf *BufItem
		select {
		case <-h.Die:
			return
		case buf = <-h.sendChan:
			if err := WriteUInt(conn, uint32(buf.Size())); err != nil {
				buf.Release()
				log.Printf("WriteUInt error: %s %+v\n", h.Addr, err)
				return
			}
			_, err := conn.Write(buf.Data())
			buf.Release()
			if err != nil {
				log.Printf("WriteUInt error: %s %+v\n", h.Addr, err)
				return
			}
		}
	}
}
