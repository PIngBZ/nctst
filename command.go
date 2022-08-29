package nctst

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

var (
	CommandReceiveChan = make(chan *BufItem, 32)

	commandPublishObservers = make([]chan Command, 0)
	commandPublishLocker    = sync.Mutex{}
)

type CommandType uint32

const (
	_ CommandType = iota

	Cmd_handshake

	Cmd_max
)

func AttachObserver(observer chan Command) {
	commandPublishLocker.Lock()
	defer commandPublishLocker.Unlock()

	commandPublishObservers = append(commandPublishObservers, observer)
}

func CommandDaemon() {
	for {
		select {
		case <-CommandReceiveChan:
		}
	}
}

func SendCommand[T any](conn *net.TCPConn, t CommandType, command *T) error {
	js, err := ToJson(command)
	if err != nil {
		return err
	}
	data := []byte(js)

	if err := WriteUInt(conn, KCP_DATA_BUF_SIZE+uint32(t)); err != nil {
		return err
	}

	if err := WriteUInt(conn, uint32(len(data))); err != nil {
		return err
	}

	if _, err := WriteData(data, conn, len(data)); err != nil {
		return err
	}

	return nil
}

func ReadCommandType(conn *net.TCPConn) (CommandType, error) {
	n, err := ReadUInt(conn)
	if err != nil {
		return 0, err
	}

	if n <= KCP_DATA_BUF_SIZE || n > uint32(KCP_DATA_BUF_SIZE+Cmd_max) {
		return 0, fmt.Errorf("err type: %d", n)
	}

	return CommandType(n - KCP_DATA_BUF_SIZE), nil
}

func ReadCommand[T any](conn *net.TCPConn) (*T, error) {
	s, err := ReadLString(conn)
	if err != nil {
		return nil, err
	}

	var obj T
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func CommandFromBuf[T any](buf *BufItem) (*T, error) {
	s := string(buf.Data())

	var obj T
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

type Command struct {
	Type CommandType
	Item interface{}
}

type CommandHandshake struct {
	TunnelID  int
	ConnID    int
	Duplicate int
	Key       string
}
