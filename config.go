package nctst

import "fmt"

const (
	DATA_BUF_SIZE = 1024 * 1024

	KCP_UDP_RECEIVE_BUF_NUM = 128
	KCP_UDP_SEND_BUF_NUM    = 32

	NEW_CONNECTION_KEY uint32 = 0xFFEEFF
)

type AddrInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (t *AddrInfo) Address() string {
	return fmt.Sprintf("%s:%d", t.Host, t.Port)
}
