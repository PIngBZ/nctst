package proxyclient

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

type TrojanClient struct {
	proxyClient

	headerWritten bool
}

func NewTrojanClient(server *ProxyInfo, target *nctst.AddrInfo) ProxyClient {
	h := &TrojanClient{}
	h.Server = server
	h.Target = target
	return h
}

func (h *TrojanClient) Connect() error {
	h.Close()
	h.headerWritten = false

	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         h.Server.LoginName,
	}

	dialer := &net.Dialer{Timeout: time.Second * 2}
	conn, err := tls.DialWithDialer(dialer, "tcp", h.Server.Address(), conf)
	if err != nil {
		return err
	}

	h.Conn = conn
	return nil
}

func (h *TrojanClient) Write(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	if !h.headerWritten {
		n, err := h.writeWithHeader(p)
		if err != nil {
			return 0, fmt.Errorf("trojan failed to flush header with payload: %+v", err)
		}
		return n, nil
	}

	n, err := h.Conn.Write(p)
	return n, err
}

func (h *TrojanClient) writeWithHeader(payload []byte) (int, error) {
	buf := bytes.NewBuffer(make([]byte, 0, nctst.DATA_BUF_SIZE+256))
	buf.Write(h.passwordHash())
	buf.Write([]byte{0x0d, 0x0a})
	h.writeMetadata(buf)
	buf.Write([]byte{0x0d, 0x0a})

	if payload != nil {
		buf.Write(payload)
	}
	_, err := h.Conn.Write(buf.Bytes())

	h.headerWritten = true
	return 0, err
}

func (h *TrojanClient) passwordHash() []byte {
	hash := sha256.New224()
	hash.Write([]byte(h.Server.Password))
	val := hash.Sum(nil)
	str := ""
	for _, v := range val {
		str += fmt.Sprintf("%02x", v)
	}
	return []byte(str)
}

func (h *TrojanClient) writeMetadata(buf *bytes.Buffer) {
	buf.WriteByte(byte(1))
	buf.WriteByte(byte(3))

	buf.WriteByte(byte(len(h.Target.Host)))
	buf.Write([]byte(h.Target.Host))

	port := [2]byte{}
	binary.BigEndian.PutUint16(port[:], uint16(h.Target.Port))
	buf.Write(port[:])
}
