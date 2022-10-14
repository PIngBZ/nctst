package proxylist

import (
	"strings"
)

type ProxyType int

const (
	ProxyTypeSocks5 = iota
	ProxyTypeTrojan
)

type ProxyListItem struct {
	Type    ProxyType
	URL     string
	ConnNum int
}

func InitProxyList() {

}

func GetProxyList(proxyFile string) []string {
	if strings.HasPrefix(proxyFile, "http") {
		return GetFromNet(proxyFile)
	} else {
		return GetFromFile(proxyFile)
	}
}

func GetFromNet(url string) []string {
	// data, err := nctst.HttpGetString(url)
	// if err != nil {

	// }
	return []string{}
}

func GetFromFile(file string) []string {

	return []string{}
}
