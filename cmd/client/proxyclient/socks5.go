package proxyclient

import (
	"strconv"
	"time"

	"github.com/PIngBZ/socks5"
)

type Socks5Client struct {
	proxyClient
}

func NewSocks5Client(serverName string, serverIP string, serverPort int, targetHost string, targetPort int) ProxyClient {
	h := &Socks5Client{}
	h.ServerName = serverName
	h.ServerIP = serverIP
	h.ServerPort = serverPort
	h.TargetHost = targetHost
	h.TargetPort = targetPort

	return h
}

func (h *Socks5Client) Connect() error {
	if h.Conn != nil {
		h.Conn.Close()
		h.Conn = nil
	}

	client := socks5.Client{
		ProxyAddr:        h.ServerIP + ":" + strconv.Itoa(h.ServerPort),
		DialTimeout:      time.Second * 5,
		HandshakeTimeout: time.Second * 5,
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: &socks5.NoAuth{},
		},
	}

	conn, err := client.Connect(socks5.Version5, h.TargetHost+":"+strconv.Itoa(h.TargetPort))
	if err != nil {
		return err
	}

	h.Conn = conn
	return nil
}

func (h *Socks5Client) Write(p []byte) (int, error) {
	n, err := h.Conn.Write(p)
	return n, err
}

func (h *Socks5Client) Read(p []byte) (int, error) {
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
