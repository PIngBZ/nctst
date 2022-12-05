package proxyclient

import (
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

type DirectClient struct {
	proxyClient
}

func NewDirectClient(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	h := &Socks5Client{}
	h.Server = server
	h.Target = target
	return h
}

func (h *DirectClient) Connect() error {
	if h.Conn != nil {
		h.Conn.Close()
		h.Conn = nil
	}

	conn, err := net.DialTimeout("tcp", h.Target.Address(), time.Second*5)
	if err != nil {
		return err
	}

	h.Conn = conn
	return nil
}
