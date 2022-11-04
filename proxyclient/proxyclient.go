package proxyclient

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

type ProxyClient interface {
	net.Conn

	Connect() error
	Ping(self ProxyClient, printDetails bool, finished func(ProxyClient, uint32, error)) bool
	LastPing() uint32
}

func NewProxyClient(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	if server.Type == "socks5" {
		return NewSocks5Client(server, target)
	} else if server.Type == "trojan" {
		return NewTrojanClient(server, target)
	} else if server.Type == "ssr" {
		return NewSSRClient(server, target)
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

func (hh *proxyClient) Ping(self ProxyClient, printDetails bool, finished func(ProxyClient, uint32, error)) bool {
	printf := func(format string, a ...any) {
		if printDetails {
			fmt.Printf(format, a...)
		}
	}

	var h = self
	hh.TestPing = 100000

	defer h.Close()

	printf("Connecting %s\n", hh.Server.Address())
	if err := h.Connect(); err != nil {
		printf("Connect Failed %s %+v\n", hh.Server.Address(), err)
		finished(h, 0, err)
		return false
	}

	h.SetDeadline(time.Now().Add(time.Second * 8))

	printf("WriteUInt1 %s\n", hh.Server.Address())
	if err := nctst.WriteUInt(h, nctst.NEW_CONNECTION_KEY); err != nil {
		printf("WriteUInt1 Failed %s %+v\n", hh.Server.Address(), err)
		finished(h, 0, err)
		return false
	}

	printf("SendCommand1 %s\n", hh.Server.Address())
	cmd := &nctst.CommandTestPing{}
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		printf("SendCommand1 Failed %s %+v\n", hh.Server.Address(), err)
		finished(h, 0, err)
		return false
	}

	printf("Receive1 %s\n", hh.Server.Address())
	_, err := nctst.ReadLBuf(h)
	if err != nil {
		printf("ReadLBuf1 Failed %s %+v\n", hh.Server.Address(), err)
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	printf("SendCommand2 %s\n", hh.Server.Address())
	cmd.SendTime = time.Now().UnixNano() / 1e6
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		printf("SendCommand2 Failed %s %+v\n", hh.Server.Address(), err)
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	printf("Receive2 %s\n", hh.Server.Address())
	buf, err := nctst.ReadLBuf(h)
	if err != nil {
		printf("ReadLBuf2 Failed %s %+v\n", hh.Server.Address(), err)
		if finished != nil {
			finished(h, 0, err)
		}
		return false
	}

	command, err := nctst.ReadCommand(buf)
	if err != nil {
		printf("ReadCommand2 Failed %s %+v\n", hh.Server.Address(), err)
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
	printf("PingResult %d\n", ping)
	if finished != nil {
		finished(h, ping, nil)
	}

	return true
}
