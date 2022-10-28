package proxyclient

import (
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

type ProxyInfo struct {
	nctst.AddrInfo
	Type      string `json:"type"`
	LoginName string `json:"loginname"`
	Password  string `json:"password"`
	ConnNum   int    `json:"connnum"`
}

type ProxyListGroup struct {
	Name      string       `json:"name"`
	ProxyList []*ProxyInfo `json:"proxylist"`
}

type ProxyFileInfo struct {
	SelectPerGroup int               `json:"selectpergroup"`
	ConnPerServer  int               `json:"connperserver"`
	ProxyGroups    []*ProxyListGroup `json:"proxygroups"`
}

type ProxyClient interface {
	net.Conn

	Connect() error
	Ping(self ProxyClient, finished func(ProxyClient, uint32, error)) bool
	LastPing() uint32
}

func NewProxyClient(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	if server.Type == "socks5" {
		return NewSocks5Client(server, target)
	} else if server.Type == "trojan" {
		return NewTrojanClient(server, target)
	} else {
		log.Println("unknown proxy type: " + server.Type)
		return nil
	}
}

type proxyClient struct {
	Server *ProxyInfo

	Target *nctst.AddrInfo

	Conn     net.Conn
	TestPing uint32
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

func (h *proxyClient) LastPing() uint32 {
	return h.TestPing
}

func (hh *proxyClient) Ping(self ProxyClient, finished func(ProxyClient, uint32, error)) bool {
	var h = self
	hh.TestPing = 100000

	defer h.Close()

	if err := h.Connect(); err != nil {
		finished(h, 0, err)
		return false
	}

	if err := nctst.WriteUInt(h, nctst.NEW_CONNECTION_KEY); err != nil {
		finished(h, 0, err)
		return false
	}

	cmd := &nctst.CommandTestPing{}
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		finished(h, 0, err)
		return false
	}

	_, err := nctst.ReadLBuf(h)
	if err != nil {
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	cmd.SendTime = time.Now().UnixNano() / 1e6
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	buf, err := nctst.ReadLBuf(h)
	if err != nil {
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	command, err := nctst.ReadCommand(buf)
	if err != nil {
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	if command.Type != nctst.Cmd_testping {
		if finished != nil {
			finished(h, 0, errors.New("testping ret type error"))
		}
		return false
	}

	ret := command.Item.(*nctst.CommandTestPing)

	ping := uint32(time.Now().UnixNano()/1e6 - ret.SendTime)

	hh.TestPing = ping
	if finished != nil {
		finished(h, ping, nil)
	}

	return true
}
