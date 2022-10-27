package nctst

import "fmt"

const (
	MAX_TCP_DATA_INTERNET_LEN = 1024 * 1024
	DATA_BUF_SIZE             = 1024 * 64

	KCP_UDP_RECEIVE_BUF_NUM = 128
	KCP_UDP_SEND_BUF_NUM    = 8

	NEW_CONNECTION_KEY uint32 = 0xFFEEFF
)

type AddrInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (t *AddrInfo) Address() string {
	return fmt.Sprintf("%s:%d", t.Host, t.Port)
}
