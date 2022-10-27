package main

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

var (
	ErrorNeedLogin = errors.New("error need login, exit, perhaps server restarted")
)

type ProxyConnector struct {
	ID      uint
	ProxyID uint

	client proxyclient.ProxyClient
	tunnel *nctst.OuterTunnel

	outerConnection *nctst.OuterConnection
}

func NewProxyConnector(id uint, proxyID uint, client proxyclient.ProxyClient, tunnel *nctst.OuterTunnel) *ProxyConnector {
	h := &ProxyConnector{}
	h.ID = id
	h.ProxyID = proxyID
	h.client = client

	h.tunnel = tunnel

	go h.daemon()

	log.Printf("ProxyConnector.New %d %d\n", proxyID, id)
	return h
}

func (h *ProxyConnector) connect() {
	log.Printf("ProxyConnector connecting %d %d\n", h.ProxyID, h.ID)

	for {
		var err error
		err = h.client.Connect()

		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}

		h.client.SetDeadline(time.Now().Add(time.Second * 5))

		if err = h.sendHandshake(h.client); err != nil {
			h.client.Close()
			log.Printf("sendHandshake error %+v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}

		if err = h.receiveHandshakeReply(h.client); err == ErrorNeedLogin {
			h.client.Close()
			nctst.CheckError(err)
			return
		} else if err != nil {
			h.client.Close()
			log.Printf("receiveHandshakeReply error %+v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	h.client.SetDeadline(time.Time{})
	h.outerConnection = h.tunnel.AddConn(h.client, h.ID)

	log.Printf("ProxyConnector connect success %d %d\n", h.ProxyID, h.ID)
}

func (h *ProxyConnector) sendHandshake(conn io.Writer) error {
	if err := nctst.WriteUInt(conn, nctst.NEW_CONNECTION_KEY); err != nil {
		return err
	}

	cmd := &nctst.CommandHandshake{}
	cmd.ClientUUID = UUID
	cmd.ClientID = ClientID
	cmd.TunnelID = h.tunnel.ID
	cmd.ConnID = h.ID
	cmd.ConnectKey = connKey
	return nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_handshake, Item: cmd})
}

func (h *ProxyConnector) receiveHandshakeReply(conn io.Reader) error {
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
		case <-h.outerConnection.Die:
			h.tunnel.RemoveConn(h.ID)
			h.reconnect()
		}
	}
}

func (h *ProxyConnector) reconnect() {
	log.Printf("ProxyConnector waiting 5s to reconnect %d %d\n", h.ProxyID, h.ID)

	time.Sleep(time.Second * 5)
	h.connect()
}
