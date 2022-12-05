package core

import (
	"errors"
	"io"
	"log"
	"sync"
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

	clientGetter func(*ProxyConnector) proxyclient.ProxyClient
	tunnel       *nctst.OuterTunnel

	die     chan struct{}
	dieOnce sync.Once
}

func NewProxyConnector(id uint, proxyID uint, tunnel *nctst.OuterTunnel, clientGetter func(*ProxyConnector) proxyclient.ProxyClient) *ProxyConnector {
	h := &ProxyConnector{}
	h.ID = id
	h.ProxyID = proxyID
	h.clientGetter = clientGetter

	h.tunnel = tunnel

	h.die = make(chan struct{})

	go h.daemon()

	log.Printf("ProxyConnector.New %d %d\n", proxyID, id)
	return h
}

func (h *ProxyConnector) Release() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.die)
		once = true
	})

	if !once {
		return
	}
}

func (c *ProxyConnector) IsReleased() bool {
	select {
	case <-c.die:
		return true
	default:
		return false
	}
}

func (h *ProxyConnector) connect() *nctst.OuterConnection {
	log.Printf("ProxyConnector connecting %d %d\n", h.ProxyID, h.ID)

	var client proxyclient.ProxyClient
	for {
		if client = h.clientGetter(h); client == nil {
			select {
			case <-h.die:
				return nil
			case <-time.After(time.Second * 5):
				continue
			}
		}
		break
	}

	for {
		if err := client.Connect(); err != nil {
			select {
			case <-h.die:
				return nil
			case <-time.After(time.Second * 5):
				continue
			}
		}

		client.SetDeadline(time.Now().Add(time.Second * 5))

		if err := h.sendHandshake(client); err != nil {
			client.Close()
			log.Printf("sendHandshake error %+v\n", err)

			select {
			case <-h.die:
				return nil
			case <-time.After(time.Second * 5):
				continue
			}
		}

		if err := h.receiveHandshakeReply(client); err == ErrorNeedLogin {
			client.Close()
			log.Printf("receiveHandshakeReply error %+v\n", err)
			return nil
		} else if err != nil {
			client.Close()
			log.Printf("receiveHandshakeReply error %+v\n", err)

			select {
			case <-h.die:
				return nil
			case <-time.After(time.Second * 5):
				continue
			}
		}
		break
	}

	if h.IsReleased() {
		client.Close()
		return nil
	}

	client.SetDeadline(time.Time{})
	log.Printf("ProxyConnector connect success %d %d\n", h.ProxyID, h.ID)
	return h.tunnel.AddConn(client, h.ID)
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
	outerConn := h.connect()
	if outerConn == nil {
		return
	}

	for {
		select {
		case <-h.die:
			return
		case <-outerConn.Die:
			h.tunnel.RemoveConn(h.ID)
			outerConn = h.reconnect()
			if outerConn == nil {
				return
			}
		}
	}
}

func (h *ProxyConnector) reconnect() *nctst.OuterConnection {
	log.Printf("ProxyConnector waiting 5s to reconnect %d %d\n", h.ProxyID, h.ID)

	select {
	case <-h.die:
		return nil
	case <-time.After(time.Second * 5):
		return h.connect()
	}
}
