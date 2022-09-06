package main

import (
	"log"

	"github.com/PIngBZ/nctst"
)

var (
	proxies = []*Proxy{}
)

type Proxy struct {
	ID   uint
	Addr string

	connectors map[uint]*ProxyConnector
}

func NewProxy(id uint, addr string, receiveChan chan *nctst.BufItem, tunnel *nctst.OuterTunnel) *Proxy {
	h := &Proxy{}
	h.ID = id

	h.connectors = make(map[uint]*ProxyConnector)
	for i := 0; i < config.Connperproxy; i++ {
		h.connectors[id] = NewProxyConnector(uint(i), id, addr, tunnel, receiveChan)
	}
	log.Printf("proxy created %d\n", id)
	return h
}
