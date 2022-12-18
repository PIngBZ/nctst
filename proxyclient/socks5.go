package proxyclient

import (
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
		HandshakeTimeout: time.Second * 5,
		Auth:             auth,
	}

	conn, err := client.Connect(socks5.Version5, h.Target.Address())
	if err != nil {
		return err
	}

	h.Conn = conn
	return nil
}
