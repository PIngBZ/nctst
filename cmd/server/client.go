package main

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/xtaci/smux"
)

type Client struct {
	UUID          string
	ID            uint
	kcp           *nctst.Kcp
	smux          *smux.Session
	duplicater    *nctst.Duplicater
	tunnels       map[uint]*nctst.OuterTunnel
	tunnelsLocker sync.Mutex
}

func NewClient(uuid string, id uint, duplicateNum int) *Client {
	h := &Client{}
	h.UUID = uuid
	h.ID = id
	h.kcp = nctst.NewKcp(id)
	h.smux, _ = smux.Server(nctst.NewCompStream(h.kcp), nctst.SmuxConfig())
	h.duplicater = nctst.NewDuplicater(duplicateNum, h.kcp.OutputChan)
	h.tunnels = make(map[uint]*nctst.OuterTunnel)
	go h.smuxLoop()

	log.Printf("new client %s %d %d\n", uuid, id, duplicateNum)
	return h
}

func (h *Client) AddConn(conn *net.TCPConn, tunnelID uint, connID uint) {
	h.tunnelsLocker.Lock()
	tunnel, ok := h.tunnels[tunnelID]
	if !ok {
		tunnel = nctst.NewOuterTunnel(tunnelID, h.ID, h.kcp.InputChan, h.duplicater.Output)
		h.tunnels[tunnelID] = tunnel
	}
	h.tunnelsLocker.Unlock()

	tunnel.AddConn(conn, connID)
}

func (h *Client) smuxLoop() {
	for {
		stream, err := h.smux.AcceptStream()
		if err != nil {
			log.Printf("accept stream: %d %+v\n", h.ID, err)
			continue
		}

		log.Printf("AcceptStream %d %d\n", h.ID, stream.ID())
		go h.connectTarget(stream)
	}
}

func (h *Client) connectTarget(stream *smux.Stream) {
	target, err := net.DialTimeout("tcp", config.Target, time.Second*3)
	if err != nil {
		log.Printf("connectTarget: %d %+v\n", h.ID, err)
		stream.Close()
		return
	}

	target.SetDeadline(time.Time{})
	stream.SetDeadline(time.Time{})

	go nctst.Transfer(stream, target)
}
