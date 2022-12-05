package core

import (
	"log"
	"sync"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

type ProxyServer struct {
	ID         uint
	tunnel     *nctst.OuterTunnel
	connectors []*ProxyConnector

	proxy *proxyclient.ProxyInfo

	die     chan struct{}
	dieOnce sync.Once
}

func NewProxyServer(id uint) *ProxyServer {
	h := &ProxyServer{}
	h.ID = id
	h.die = make(chan struct{})

	h.proxy = proxyListMgr.Get()
	if h.proxy == nil {
		log.Println("NewProxyServer no enougth proxy server")
		return nil
	}

	h.tunnel = nctst.NewOuterTunnel(config.Key, h.ID, ClientID, kcp.InputChan, duplicater.Output, nil)

	h.connectors = make([]*ProxyConnector, h.proxy.ConnNum)
	for i := 0; i < h.proxy.ConnNum; i++ {
		h.connectors[uint(i)] = NewProxyConnector(uint(i), h.ID, h.tunnel, func(pc *ProxyConnector) proxyclient.ProxyClient {
			return proxyclient.NewProxyClient(h.proxy, config.Server)
		})
	}

	log.Printf("proxyserver created %d\n", id)
	return h
}

func (h *ProxyServer) ChangeProxy() {
	newProxy := proxyListMgr.Get()
	if newProxy == nil {
		log.Println("ChangeProxy no enougth proxy server")
		return
	}

	proxyListMgr.Put(h.proxy)
	h.proxy = newProxy

	h.tunnel.RemoveAllConn()
}

func (h *ProxyServer) SendLogout() {
	if h.IsClosed() {
		return
	}

	if h.tunnel == nil {
		return
	}

	cmd := &nctst.CommandLogout{}
	cmd.UserName = config.UserName
	cmd.ClientUUID = UUID
	h.tunnel.SendCommand(&nctst.Command{Type: nctst.Cmd_logout, Item: cmd})
}

func (h *ProxyServer) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.die)
		once = true
	})

	if !once {
		return
	}

	for _, connector := range h.connectors {
		connector.Release()
	}
	h.connectors = nil

	h.tunnel.Close()
	h.tunnel = nil
}

func (c *ProxyServer) IsClosed() bool {
	select {
	case <-c.die:
		return true
	default:
		return false
	}
}
