package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	UserName     string `json:"username"`
	PassWord     string `json:"password"`
	Listen       string `json:"listen"`
	ServerIP     string `json:"serverIP"`
	ServerPort   string `json:"serverPort"`
	ServerPortI  int
	Proxies      []string `json:"proxies"`
	ProxyFile    string   `json:"proxyfile"`
	Connperproxy int      `json:"connperproxy"`
	Compress     bool     `json:"compress"`
	TarType      string   `json:"tartype"`
	Key          string   `json:"key"`
	Duplicate    int      `json:"duplicate"`
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

	cfg.ServerPortI, _ = strconv.Atoi(cfg.ServerPort)

	if cfg.Connperproxy == 0 {
		cfg.Connperproxy = 1
	}
	if cfg.Duplicate == 0 {
		cfg.Duplicate = 1
	}

	if cfg.TarType != "socks5" && cfg.TarType != "tcp" {
		return nil, fmt.Errorf("TarType can only be socks5/tcp")
	}

	return cfg, nil
}
