package main

import (
	"net"
	"sync"
	"time"

	"github.com/dearzhp/nctst"
	"github.com/haochen233/socks5"
)

type ProxyConnector struct {
	ID      int
	Address string

	tunnel      *nctst.OuterTunnel
	receiveChan chan *nctst.BufItem

	outer *nctst.OuterConnection

	die     chan struct{}
	dieOnce sync.Once
}

func NewProxyConnector(id int, addr string, tunnel *nctst.OuterTunnel, receiveChan chan *nctst.BufItem) *ProxyConnector {
	h := &ProxyConnector{}
	h.ID = id
	h.Address = addr

	h.tunnel = tunnel
	h.receiveChan = receiveChan

	h.die = make(chan struct{})

	go h.daemon()

	return h
}

func (h *ProxyConnector) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.die)
		once = true
	})

	if !once {
		return
	}

	if h.outer != nil {
		h.outer.Close()
	}
}

func (h *ProxyConnector) IsClosed() bool {
	select {
	case <-h.die:
		return true
	default:
	}
	return false
}

func (h *ProxyConnector) connect() {
	client := socks5.Client{
		ProxyAddr: h.Address,
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: &socks5.NoAuth{},
		},
	}
	var conn *net.TCPConn
	for {
		if h.IsClosed() {
			return
		}

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

	if h.IsClosed() {
		return
	}

	conn.SetDeadline(time.Time{})
	h.outer = nctst.NewOuterConnection(h.tunnel.ID, h.ID, conn, h.receiveChan, h.tunnel.OutputChan)
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
	h.reconnect()

	for {
		select {
		case <-h.die:
			return
		case <-h.outer.Die:
			h.reconnect()
		}
	}
}

func (h *ProxyConnector) reconnect() {
	select {
	case <-h.die:
		return
	case <-time.After(time.Second * 5):
		h.connect()
	}
}
