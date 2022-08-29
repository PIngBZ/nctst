package main

import "github.com/dearzhp/nctst"

var (
	proxies = []*Proxy{}
)

type Proxy struct {
	ID   int
	Addr string

	tunnel     *nctst.OuterTunnel
	connectors map[int]*ProxyConnector
}

func NewProxy(id int, addr string, receiveChan chan *nctst.BufItem, sendChan chan *nctst.BufItem) *Proxy {
	h := &Proxy{}
	h.ID = id

	h.tunnel = nctst.NewOuterTunnel(id, sendChan)

	h.connectors = make(map[int]*ProxyConnector)
	for i := 0; i < config.Connperproxy; i++ {
		h.connectors[id] = NewProxyConnector(i, addr, h.tunnel, receiveChan, sendChan)
	}
	return h
}

func startUpstreamProxies() {
	proxies = make([]*Proxy, len(config.Proxies))

	for i, server := range config.Proxies {
		proxies[i] = NewProxy(i, server, k.InputChan, duplicater.Output)
	}
}
