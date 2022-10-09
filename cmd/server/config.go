package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Listen      string `json:"listen"`
	AdminListen string `json:"adminlisten"`
	Target      string `json:"target"`
	Key         string `json:"key"`
	Test        bool   `json:"test"`
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
