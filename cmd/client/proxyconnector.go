package main

import (
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/haochen233/socks5"
)

type ProxyConnector struct {
	ID      int
	ProxyID int
	Address string

	tunnel      *nctst.OuterTunnel
	receiveChan chan *nctst.BufItem

	outer *nctst.OuterConnection
}

func NewProxyConnector(id int, proxyID int, addr string, tunnel *nctst.OuterTunnel, receiveChan chan *nctst.BufItem) *ProxyConnector {
	h := &ProxyConnector{}
	h.ID = id
	h.ProxyID = proxyID
	h.Address = addr

	h.tunnel = tunnel
	h.receiveChan = receiveChan

	go h.daemon()

	log.Printf("ProxyConnector created %d %d", proxyID, id)
	return h
}

func (h *ProxyConnector) connect() {
	log.Printf("ProxyConnector connecting %d %d", h.ProxyID, h.ID)
	client := socks5.Client{
		ProxyAddr: h.Address,
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: &socks5.NoAuth{},
		},
	}
	var conn *net.TCPConn
	for {
		var err error
		conn, err = client.Connect(socks5.Version5, config.ServerIP+":"+config.ServerPort)
		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		if err = h.sendHandshake(conn); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	log.Printf("ProxyConnector connect success %d %d", h.ProxyID, h.ID)
	conn.SetDeadline(time.Time{})
	h.outer = nctst.NewOuterConnection(h.tunnel.ID, h.ID, conn, h.receiveChan, h.tunnel.OutputChan, h.tunnel.CommandSendChan)
	h.tunnel.Add(h.ID, h.outer)
}

func (h *ProxyConnector) sendHandshake(conn *net.TCPConn) error {
	cmd := &nctst.CommandHandshake{}
	cmd.TunnelID = h.tunnel.ID
	cmd.ConnID = h.ID
	cmd.Duplicate = config.Duplicate
	cmd.Key = config.Key
	return nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_handshake, Item: cmd})
}

func (h *ProxyConnector) daemon() {
	h.connect()

	for {
		select {
		case <-h.outer.Die:
			h.tunnel.Remove(h.outer.ID)
			h.reconnect()
		}
	}
}

func (h *ProxyConnector) reconnect() {
	log.Printf("ProxyConnector waiting 5s to reconnect %d %d", h.ProxyID, h.ID)
	select {
	case <-time.After(time.Second * 5):
		h.connect()
	}
}
