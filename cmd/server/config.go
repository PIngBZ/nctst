package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Listen        string `json:"listen"`
	Target        string `json:"target"`
	Key           string `json:"key"`
	AdminListen   string `json:"adminlisten"`
	AdminPassword string `json:"adminpwd"`
	Test          bool   `json:"test"`
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
