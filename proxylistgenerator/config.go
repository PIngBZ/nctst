package main

import (
	"encoding/json"
	"os"

	"github.com/PIngBZ/nctst"
)

type Config struct {
	Key               string          `json:"key"`
	UserName          string          `json:"username"`
	PassWord          string          `json:"password"`
	SrcFile           string          `json:"srcfile"`
	Target            *nctst.AddrInfo `json:"target"`
	SelectPerGroup    int             `json:"selectpergroup"`
	ClientTotalSelect int             `json:"clienttotalselect"`
	PingThreadNum     int             `json:"pingthreadnum"`
	PublishServer     *nctst.AddrInfo `json:"publishserver"`
	PublishTimeout    int             `json:"publishtimeout"`
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
