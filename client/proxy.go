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

func NewProxy(id uint, proxy *proxyclient.ProxyInfo) *Proxy {
	h := &Proxy{}
	h.ID = id

	tunnel := nctst.NewOuterTunnel(config.Key, id, ClientID, kcp.InputChan, duplicater.Output)
	tunnels = append(tunnels, tunnel)

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
