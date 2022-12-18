package nctst

import (
	"fmt"
	"time"

	"github.com/xtaci/smux"
)

const (
	DATA_BUF_SIZE = 1024 * 512

	KCP_UDP_RECEIVE_BUF_NUM = 1024
	KCP_UDP_SEND_BUF_NUM    = 1024

	NEW_CONNECTION_KEY uint32 = 0xFFEEFF
)

type AddrInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (t *AddrInfo) Address() string {
	return fmt.Sprintf("%s:%d", t.Host, t.Port)
}

func SmuxConfig() *smux.Config {
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxFrameSize = 1024 * 4
	smuxConfig.MaxReceiveBuffer = 1024 * 1024 * 8
	smuxConfig.KeepAliveDisabled = false
	smuxConfig.KeepAliveInterval = time.Second * 10
	smuxConfig.KeepAliveTimeout = time.Hour * 24 * 30

	err := smux.VerifyConfig(smuxConfig)
	CheckError(err)

	return smuxConfig
}
