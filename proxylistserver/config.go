package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port       int              `json:"port"`
	Key        string           `json:"key"`
	Password   string           `json:"password"`
	Generators []*GeneratorItem `json:"generators"`
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
