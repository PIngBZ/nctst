package proxyclient

import (
	"io"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/socks5"
)

type Socks5Client struct {
	proxyClient
}

func NewSocks5Client(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	h := &Socks5Client{}
	h.Server = server
	h.Target = target
	return h
}

func (h *Socks5Client) Connect() error {
	if h.Conn != nil {
		h.Conn.Close()
		h.Conn = nil
	}

	auth := map[socks5.METHOD]socks5.Authenticator{
		socks5.NO_AUTHENTICATION_REQUIRED: socks5.NoAuth{},
	}

	if len(h.Server.LoginName) > 0 {
		auth[socks5.USERNAME_PASSWORD] = &socks5.UserPasswd{Username: h.Server.LoginName, Password: h.Server.Password}
	}

	client := socks5.Client{
		ProxyAddr:        h.Server.Address(),
		DialTimeout:      time.Second * 2,
		HandshakeTimeout: time.Second * 2,
		Auth:             auth,
	}

	conn, err := client.Connect(socks5.Version5, h.Target.Address())
	if err != nil {
		return err
	}

	h.Conn = conn
	return nil
}

func (h *Socks5Client) Write(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := h.Conn.Write(p)
	return n, err
}

func (h *Socks5Client) Read(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := h.Conn.Read(p)
	return n, err
}

func (h *Socks5Client) Close() error {
	if h.Conn == nil {
		return nil
	}

	err := h.Conn.Close()
	h.Conn = nil
	return err
}
