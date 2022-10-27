package main

import (
	"log"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

type Proxy struct {
	ID         uint
	connectors map[uint]*ProxyConnector
}

func NewProxy(id uint, proxy *proxyclient.ProxyInfo, tunnel *nctst.OuterTunnel) *Proxy {
	h := &Proxy{}
	h.ID = id

	h.connectors = make(map[uint]*ProxyConnector)
	for i := 0; i < proxy.ConnNum; i++ {
		client := proxyclient.NewProxyClient(proxy, config.Server)
		if client == nil {
			return nil
		}

		h.connectors[id] = NewProxyConnector(uint(i), id, client, tunnel)
	}
	log.Printf("proxy created %d\n", id)
	return h
}
