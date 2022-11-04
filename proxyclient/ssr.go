package proxyclient

import (
	"io"
	"net/url"

	"github.com/PIngBZ/nctst"
	shadowsocksr "github.com/sun8911879/shadowsocksR"
	"github.com/sun8911879/shadowsocksR/tools/socks"
)

type SSRClient struct {
	proxyClient

	EncryptMethod string
	Obfs          string
	ObfsParam     string
	Protocol      string
	ProtocolParam string
}

func NewSSRClient(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	h := &SSRClient{}
	h.Server = server
	h.Target = target

	for k, v := range server.Params {
		switch k {
		case "cipher":
			h.EncryptMethod = v
		case "obfs":
			h.Obfs = v
		case "obfs-param":
			h.ObfsParam = v
		case "protocol":
			h.Protocol = v
		case "protocol-param":
			h.ProtocolParam = v
		}
	}

	return h
}

func (h *SSRClient) Connect() error {
	if h.Conn != nil {
		h.Conn.Close()
		h.Conn = nil
	}

	u := &url.URL{
		Scheme: "ssr",
		Host:   h.Server.Address(),
	}
	v := u.Query()
	v.Set("encrypt-method", h.EncryptMethod)
	v.Set("encrypt-key", h.Server.Password)
	v.Set("obfs", h.Obfs)
	v.Set("obfs-param", h.ObfsParam)
	v.Set("protocol", h.Protocol)
	v.Set("protocol-param", h.ProtocolParam)
	u.RawQuery = v.Encode()

	ssrconn, err := shadowsocksr.NewSSRClient(u)
	if err != nil {
		return err
	}

	ssrconn.IObfs.SetData(ssrconn.IObfs.GetData())
	ssrconn.IProtocol.SetData(ssrconn.IProtocol.GetData())

	rawaddr := socks.ParseAddr(h.Target.Address())
	if _, err := ssrconn.Write(rawaddr); err != nil {
		ssrconn.Close()
		return err
	}

	h.Conn = ssrconn
	return nil
}

func (h *SSRClient) Write(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := h.Conn.Write(p)
	return n, err
}

func (h *SSRClient) Read(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := h.Conn.Read(p)
	return n, err
}

func (h *SSRClient) Close() error {
	if h.Conn == nil {
		return nil
	}

	err := h.Conn.Close()
	h.Conn = nil
	return err
}
