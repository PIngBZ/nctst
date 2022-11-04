package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Key        string `json:"key"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
	SrcFile    string `json:"srcfile"`
	ServerHost string `json:"serverhost"`
	ServerPort int    `json:"serverport"`
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

	return cfg, nil
}
