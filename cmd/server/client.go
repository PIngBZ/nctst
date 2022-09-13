package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/socks5"
	"github.com/xtaci/smux"
)

type Client struct {
	UUID    string
	ID      uint
	ConnKey string

	kcp            *nctst.Kcp
	smux           *smux.Session
	listener       net.Listener
	duplicater     *nctst.Duplicater
	tunnels        map[uint]*nctst.OuterTunnel
	tunnelsLocker  sync.Mutex
	tunnelsListVer uint32
}

func NewClient(uuid string, id uint, compress bool, duplicateNum int, tarType string) *Client {
	h := &Client{}
	h.UUID = uuid
	h.ID = id
	k := md5.Sum([]byte(uuid))
	h.ConnKey = hex.EncodeToString(k[:])

	h.kcp = nctst.NewKcp(id)
	if compress {
		h.smux, _ = smux.Server(nctst.NewCompStream(h.kcp), nctst.SmuxConfig())
	} else {
		h.smux, _ = smux.Server(h.kcp, nctst.SmuxConfig())
	}
	h.listener = NewSmuxWrapper(h.smux)

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

	if tarType == "socks5" {
		go h.listenAndServeSocks5()
	} else {
		go h.listenAndServeTCP()
	}

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

func (h *Client) listenAndServeTCP() {
	for {
		stream, err := h.listener.Accept()
		if err != nil {
			log.Printf("accept stream: %d %+v\n", h.ID, err)
			continue
		}

		log.Printf("AcceptStream %d %d\n", h.ID, stream.(*smux.Stream).ID())
		go h.connectTarget(stream)
	}
}

func (h *Client) connectTarget(conn net.Conn) {
	target, err := net.DialTimeout("tcp", config.Target, time.Second*3)
	if err != nil {
		log.Printf("connectTarget: %d %+v\n", h.ID, err)
		conn.Close()
		return
	}

	target.SetDeadline(time.Time{})
	conn.SetDeadline(time.Time{})

	go nctst.Transfer(conn, target)
}

func (h *Client) listenAndServeSocks5() {
	srv := &socks5.Server{
		Addr:                  config.Listen,
		Authenticators:        nil,
		DisableSocks4:         true,
		Transporter:           h,
		DialTimeout:           time.Second * 5,
		HandshakeReadTimeout:  time.Second * 5,
		HandshakeWriteTimeout: time.Second * 5,
	}

	srv.Serve(h.listener)
}

func (t *Client) TransportStream(client io.ReadWriteCloser, remote io.ReadWriteCloser) <-chan error {
	nctst.Transfer(client, remote)
	return nil
}

func (t *Client) TransportUDP(server *socks5.UDPConn, request *socks5.Request) error {
	return nil
}
