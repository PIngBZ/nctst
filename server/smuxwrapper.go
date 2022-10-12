package main

import (
	"net"

	"github.com/xtaci/smux"
)

type SmuxWrapper struct {
	session *smux.Session
}

func NewSmuxWrapper(session *smux.Session) *SmuxWrapper {
	h := &SmuxWrapper{}
	h.session = session
	return h
}

func (h *SmuxWrapper) Accept() (net.Conn, error) {
	return h.session.AcceptStream()
}

func (h *SmuxWrapper) Close() error {
	return h.session.Close()
}

func (h *SmuxWrapper) Addr() net.Addr {
	return h.session.LocalAddr()
}
