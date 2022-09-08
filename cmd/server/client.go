package main

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/xtaci/smux"
)

type Client struct {
	UUID           string
	ID             uint
	kcp            *nctst.Kcp
	smux           *smux.Session
	duplicater     *nctst.Duplicater
	tunnels        map[uint]*nctst.OuterTunnel
	tunnelsLocker  sync.Mutex
	tunnelsListVer uint32
}

func NewClient(uuid string, id uint, compress bool, duplicateNum int) *Client {
	h := &Client{}
	h.UUID = uuid
	h.ID = id
	h.kcp = nctst.NewKcp(id)
	if compress {
		h.smux, _ = smux.Server(nctst.NewCompStream(h.kcp), nctst.SmuxConfig())
	} else {
		h.smux, _ = smux.Server(h.kcp, nctst.SmuxConfig())
	}

	h.tunnels = make(map[uint]*nctst.OuterTunnel)
	h.tunnelsListVer = 100
	h.duplicater = nctst.NewDuplicater(duplicateNum, h.kcp.OutputChan, func(v uint32) (uint32, []*nctst.OuterTunnel) {
		if v == atomic.LoadUint32(&h.tunnelsListVer) {
			return v, nil
		}

		h.tunnelsLocker.Lock()
		defer h.tunnelsLocker.Unlock()

		tunnels := make([]*nctst.OuterTunnel, 0, len(h.tunnels))
		for _, tunnel := range h.tunnels {
			tunnels = append(tunnels, tunnel)
		}
		return atomic.LoadUint32(&h.tunnelsListVer), tunnels
	})
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
		atomic.AddUint32(&h.tunnelsListVer, 1)
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
	target, err := net.DialTimeout("tcp", config.Target, time.Second*5)
	if err != nil {
		log.Printf("connectTarget: %d %+v\n", h.ID, err)
		stream.Close()
		return
	}

	target.SetDeadline(time.Time{})
	stream.SetDeadline(time.Time{})

	go nctst.Transfer(stream, target)
}
