package proxyclient

import (
	"github.com/PIngBZ/nctst"
)

type ProxyFile struct {
	Type     string `json:"type"`
	Url      string `json:"url"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

type ProxyInfo struct {
	nctst.AddrInfo
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	LoginName string            `json:"loginname"`
	Password  string            `json:"password"`
	ConnNum   int               `json:"connnum"`
	Params    map[string]string `json:"params"`
}

type ProxyGroup struct {
	Name string       `json:"name"`
	List []*ProxyInfo `json:"list"`
}

type ProxyGroups struct {
	SelectPerGroup int           `json:"selectpergroup"`
	Groups         []*ProxyGroup `json:"groups"`
}

type PingTarget struct {
	Target      *nctst.AddrInfo
	PingThreads int
}
