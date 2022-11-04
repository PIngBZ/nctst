package main

import (
	"encoding/json"
	"os"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

type Config struct {
	UserName   string                   `json:"username"`
	PassWord   string                   `json:"password"`
	Listen     string                   `json:"listen"`
	Server     *nctst.AddrInfo          `json:"server"`
	Proxies    []*proxyclient.ProxyInfo `json:"proxies"`
	ProxyFile  *proxyclient.ProxyFile   `json:"proxyfile"`
	MapTargets []*nctst.AddrInfo        `json:"maptargets"`
	Compress   bool                     `json:"compress"`
	Key        string                   `json:"key"`
}

func parseConfig(configFile string) (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}

	if err := json.NewDecoder(file).Decode(cfg); err != nil {
		return nil, err
	}

	for _, proxy := range cfg.Proxies {
		if proxy.ConnNum == 0 {
			proxy.ConnNum = 1
		}
	}

	return cfg, nil
}
