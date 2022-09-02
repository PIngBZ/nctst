package main

import (
	"log"

	"github.com/PIngBZ/nctst"
)

var (
	proxies = []*Proxy{}
)

type Proxy struct {
	ID   int
	Addr string

	connectors map[int]*ProxyConnector
}

func NewProxy(id int, addr string, receiveChan chan *nctst.BufItem, tunnel *nctst.OuterTunnel) *Proxy {
	h := &Proxy{}
	h.ID = id

	h.connectors = make(map[int]*ProxyConnector)
	for i := 0; i < config.Connperproxy; i++ {
		h.connectors[id] = NewProxyConnector(i, id, addr, tunnel, receiveChan)
	}
	log.Printf("Proxy created %d", id)
	return h
}
