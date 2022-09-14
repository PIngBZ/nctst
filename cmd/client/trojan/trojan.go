package trojan

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

type TrojanClient struct {
	ServerName string
	ServerIP   string
	ServerPort int

	TargetHost string
	TargetPort int

	Conn     net.Conn
	TestPing uint

	headerWritten bool
}

func NewTrojanClient(serverName string, serverIP string, serverPort int, targetHost string, targetPort int) *TrojanClient {
	h := &TrojanClient{}
	h.ServerName = serverName
	h.ServerIP = serverIP
	h.ServerPort = serverPort
	h.TargetHost = targetHost
	h.TargetPort = targetPort
	return h
}

func (h *TrojanClient) Connect() error {
	if h.Conn != nil {
		return errors.New("already connect")
	}
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
	if !h.headerWritten {
		n, err := h.writeWithHeader(p)
		if err != nil {
			return 0, fmt.Errorf("trojan failed to flush header with payload: %+v\n", err)
		}
		return n, nil
	}

	n, err := h.Conn.Write(p)
	return n, err
}

func (h *TrojanClient) Read(p []byte) (int, error) {
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

func (h *TrojanClient) writeMetadata(w io.Writer) (int64, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	buf.WriteByte(byte(1))

	_, err := buf.Write([]byte{byte(3)})
	buf.Write([]byte{byte(len(h.TargetHost))})
	_, err = buf.Write([]byte(h.TargetHost))
	if err != nil {
		return 0, err
	}
	port := [2]byte{}
	binary.BigEndian.PutUint16(port[:], uint16(h.TargetPort))
	n, err := buf.Write(port[:])
	if err != nil {
		return 0, err
	}

	n, err = w.Write(buf.Bytes())
	return int64(n), err
}

func (h *TrojanClient) Ping(finished func(*TrojanClient, uint, error)) {
	defer h.Close()

	if err := h.Connect(); err != nil {
		finished(h, 0, err)
		return
	}

	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_idle, Item: &nctst.CommandIdle{}}); err != nil {
		finished(h, 0, err)
		return
	}

	time.Sleep(time.Millisecond * 100)

	cmd := &nctst.CommandTestPing{}
	cmd.SendTime = time.Now().UnixNano() / 1e6
	if err := nctst.SendCommand(h, &nctst.Command{Type: nctst.Cmd_testping, Item: cmd}); err != nil {
		finished(h, 0, err)
		return
	}

	buf, err := nctst.ReadLBuf(h)
	if err != nil {
		finished(h, 0, err)
		return
	}

	command, err := nctst.ReadCommand(buf)
	if err != nil {
		finished(h, 0, err)
		return
	}

	if command.Type != nctst.Cmd_testping {
		finished(h, 0, errors.New("testping ret type error"))
		return
	}

	ret := command.Item.(*nctst.CommandTestPing)

	ping := uint(time.Now().UnixNano()/1e6 - ret.SendTime)

	h.TestPing = ping
	finished(h, ping, nil)
}
