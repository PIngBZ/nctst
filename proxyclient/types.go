package proxyclient

import (
	"time"

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
	Ping      uint32            `json:"-"`
	PingTime  time.Time         `json:"-"`
}

type ProxyGroup struct {
	Name string       `json:"name"`
	List []*ProxyInfo `json:"list"`
}

type ProxyGroups struct {
	Version           string        `json:"ver"`
	SelectPerGroup    int           `json:"selectpergroup"`
	ClientTotalSelect int           `json:"clienttotalselect"`
	Groups            []*ProxyGroup `json:"groups"`
}

type PingTarget struct {
	Target      *nctst.AddrInfo
	PingThreads int
}
