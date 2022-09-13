package main

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/socks5"
)

var (
	ErrorNeedLogin = errors.New("Error Need Login")
)

type ProxyConnector struct {
	ID      uint
	ProxyID uint
	Address string

	tunnel *nctst.OuterTunnel

	outerDieSignal chan struct{}
}

func NewProxyConnector(id uint, proxyID uint, addr string, tunnel *nctst.OuterTunnel) *ProxyConnector {
	h := &ProxyConnector{}
	h.ID = id
	h.ProxyID = proxyID
	h.Address = addr

	h.tunnel = tunnel

	go h.daemon()

	log.Printf("new proxy connector %d %d\n", proxyID, id)
	return h
}

func (h *ProxyConnector) connect() bool {
	log.Printf("ProxyConnector connecting %d %d\n", h.ProxyID, h.ID)

	client := socks5.Client{
		ProxyAddr:        h.Address,
		DialTimeout:      time.Second * 5,
		HandshakeTimeout: time.Second * 5,
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

		conn.SetDeadline(time.Now().Add(time.Second * 5))

		if err = h.sendHandshake(conn); err != nil {
			conn.Close()
			log.Printf("sendHandshake error %+v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}

		if err = h.receiveHandshakeReply(conn); err == ErrorNeedLogin {
			conn.Close()
			log.Printf("receiveHandshakeReply Error need login, exit\n")
			return false
		} else if err != nil {
			conn.Close()
			log.Printf("sendHandshake error %+v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	conn.SetDeadline(time.Time{})
	h.outerDieSignal = h.tunnel.AddConn(conn, h.ID)

	log.Printf("ProxyConnector connect success %d %d\n", h.ProxyID, h.ID)
	return true
}

func (h *ProxyConnector) sendHandshake(conn *net.TCPConn) error {
	cmd := &nctst.CommandHandshake{}
	cmd.ClientUUID = UUID
	cmd.ClientID = ClientID
	cmd.TunnelID = h.tunnel.ID
	cmd.ConnID = h.ID
	cmd.ConnectKey = connKey
	cmd.Key = config.Key
	return nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_handshake, Item: cmd})
}

func (h *ProxyConnector) receiveHandshakeReply(conn *net.TCPConn) error {
	buf, err := nctst.ReadLBuf(conn)
	if err != nil {
		return err
	}

	if nctst.GetCommandType(buf) != nctst.Cmd_handshakeReply {
		buf.Release()
		return errors.New("receiveHandshakeReply type error")
	}

	command, err := nctst.ReadCommand(buf)
	buf.Release()

	if err != nil {
		return err
	}
	cmd := command.Item.(*nctst.CommandHandshakeReply)

	if cmd.Code == nctst.HandshakeReply_needlogin {
		return ErrorNeedLogin
	}

	return nil
}

func (h *ProxyConnector) daemon() {
	h.connect()
	for {
		select {
		case <-h.outerDieSignal:
			h.tunnel.Remove(h.ID)
			if !h.reconnect() {
				return
			}
		}
	}
}

func (h *ProxyConnector) reconnect() bool {
	log.Printf("ProxyConnector waiting 5s to reconnect %d %d\n", h.ProxyID, h.ID)
	select {
	case <-time.After(time.Second * 5):
		return h.connect()
	}
}
