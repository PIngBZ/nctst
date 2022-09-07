package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Listen       string   `json:"listen"`
	ServerIP     string   `json:"serverIP"`
	ServerPort   string   `json:"serverPort"`
	Proxies      []string `json:"proxies"`
	Connperproxy int      `json:"connperproxy"`
	Compress     bool     `json:"compress"`
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

	if cfg.Connperproxy == 0 {
		cfg.Connperproxy = 1
	}
	if cfg.Duplicate == 0 {
		cfg.Duplicate = 1
	}

	return cfg, nil
}
