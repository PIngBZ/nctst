package proxylist

import "strings"

func GetProxyList(proxyFile string) []string {
	if strings.HasPrefix(proxyFile, "http") {
		return GetFromNet(proxyFile)
	} else {
		return GetFromFile(proxyFile)
	}
}

func GetFromNet(url string) []string {

	return []string{}
}

func GetFromFile(file string) []string {

	return []string{}
}
