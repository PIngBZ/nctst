package main

type ProxyServerType int

const (
	ProxyServerTypeSocks5 = iota
	ProxyServerTypeTrojan
)

type ProxyServerList struct {
}
