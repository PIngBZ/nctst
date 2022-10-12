package main

import (
	"log"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxy/proxyclient"
)

type Proxy struct {
	ID         uint
	connectors map[uint]*ProxyConnector
}

func NewProxy(id uint, proxyIP string, tunnel *nctst.OuterTunnel) *Proxy {
	h := &Proxy{}
	h.ID = id

	h.connectors = make(map[uint]*ProxyConnector)
	for i := 0; i < config.Connperproxy; i++ {
		client := proxyclient.NewProxyClient(proxyIP, config.ServerIP, config.ServerPortI)
		if client == nil {
			return nil
		}

		h.connectors[id] = NewProxyConnector(uint(i), id, client, tunnel)
	}
	log.Printf("proxy created %d\n", id)
	return h
}
