package proxyclient

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/PIngBZ/nctst"
)

type ProxyClient interface {
	net.Conn

	Connect() error
	Ping(finished func(ProxyClient, uint, error))
	LastPing() uint
}

func NewProxyClient(addr string, serverIP string, serverPort int) ProxyClient {
	if addr[:6] == "socks5" {
		host, port, err := nctst.SplitHostPort(addr[7:])
		if err != nil {
			log.Printf("NewProxyServer SplitHostPort %+v\n", err)
			return nil
		}
		return NewSocks5Client(addr[7:], host, port, serverIP, serverPort)
	} else if addr[:6] == "trojan" {
		s := addr[7:]
		sp := strings.Split(s, "@")

		host, port, err := nctst.SplitHostPort(sp[1])
		if err != nil {
			log.Printf("NewProxyServer SplitHostPort %+v\n", err)
			return nil
		}

		return NewTrojanClient(sp[0], host, port, serverIP, serverPort)
	} else {
		log.Println("unknown proxy type: " + addr)
		return nil
	}
}

type proxyClient struct {
	ServerName string
	ServerIP   string
	ServerPort int

	TargetHost string
	TargetPort int

	Conn     net.Conn
	TestPing uint
}

func (h *proxyClient) LocalAddr() net.Addr {
	return nil
}

func (h *proxyClient) RemoteAddr() net.Addr {
	return nil
}

func (h *proxyClient) SetDeadline(t time.Time) error {
	if h.Conn == nil {
		return io.ErrClosedPipe
	}

	return h.Conn.SetDeadline(t)
}

func (h *proxyClient) SetReadDeadline(t time.Time) error {
	if h.Conn == nil {
		return io.ErrClosedPipe
	}

	return h.Conn.SetReadDeadline(t)
}

func (h *proxyClient) SetWriteDeadline(t time.Time) error {
	if h.Conn == nil {
		return io.ErrClosedPipe
	}

	return h.Conn.SetWriteDeadline(t)
}

func (h *proxyClient) LastPing() uint {
	return h.TestPing
}

func (hh *proxyClient) Ping(finished func(ProxyClient, uint, error)) {
	var i interface{} = hh
	var h = i.(ProxyClient)

	defer h.Close()

	if err := h.Connect(); err != nil {
		finished(h, 0, err)
		return
	}

	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_idle, Item: &nctst.CommandIdle{}}); err != nil {
		finished(h, 0, err)
		return
	}

	time.Sleep(time.Millisecond * 500)

	cmd := &nctst.CommandTestPing{}
	cmd.SendTime = time.Now().UnixNano() / 1e6
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		finished(h, 0, err)
		return
	}

	buf, err := nctst.ReadLBuf(h)
	if err != nil {
		finished(h, 0, err)
		return
	}

	command, err := nctst.ReadCommand(buf)
	if err != nil {
		finished(h, 0, err)
		return
	}

	if command.Type != nctst.Cmd_testping {
		finished(h, 0, errors.New("testping ret type error"))
		return
	}

	ret := command.Item.(*nctst.CommandTestPing)

	ping := uint(time.Now().UnixNano()/1e6 - ret.SendTime)

	hh.TestPing = ping
	finished(h, ping, nil)
}
