package proxyclient

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/PIngBZ/nctst"
)

type TrojanClient struct {
	proxyClient

	headerWritten bool
}

func NewTrojanClient(serverName string, serverIP string, serverPort int, targetHost string, targetPort int) ProxyClient {
	h := &TrojanClient{}
	h.ServerName = serverName
	h.ServerIP = serverIP
	h.ServerPort = serverPort
	h.TargetHost = targetHost
	h.TargetPort = targetPort
	return h
}

func (h *TrojanClient) Connect() error {
	h.Close()

	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         h.ServerName,
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", h.ServerIP, h.ServerPort), conf)
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

func (h *TrojanClient) Read(p []byte) (int, error) {
	if h.Conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, err := h.Conn.Read(p)
	return n, err
}

func (h *TrojanClient) Close() error {
	if h.Conn == nil {
		return nil
	}

	err := h.Conn.Close()
	h.Conn = nil
	h.headerWritten = false
	return err
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
	hash.Write([]byte(h.ServerName))
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

	buf.WriteByte(byte(len(h.TargetHost)))
	buf.Write([]byte(h.TargetHost))

	port := [2]byte{}
	binary.BigEndian.PutUint16(port[:], uint16(h.TargetPort))
	buf.Write(port[:])
}
