package nctst

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

var (
	CommandReceiveChan = make(chan *BufItem, 8)

	commandPublishObservers = make([]chan *Command, 0)
	commandPublishLocker    = sync.Mutex{}
)

type CommandType uint32

const (
	_ CommandType = iota

	Cmd_handshake
	Cmd_ping

	Cmd_max
)

func AttachCommandObserver(observer chan *Command) {
	commandPublishLocker.Lock()
	defer commandPublishLocker.Unlock()

	commandPublishObservers = append(commandPublishObservers, observer)
}

func RemoveCommandObserver(observer chan *Command) {
	commandPublishLocker.Lock()
	defer commandPublishLocker.Unlock()

	for idx, item := range commandPublishObservers {
		if item == observer {
			commandPublishObservers = append(commandPublishObservers[:idx], commandPublishObservers[idx+1:]...)
			return
		}
	}
}

func CommandDaemon() {
	for buf := range CommandReceiveChan {
		if cmd, err := CommandFromBuf(buf); err == nil {
			publishCommand(cmd)
		} else {
			log.Printf("CommandDaemon CommandFromBuf error: %+v %d", err, buf.Size())
		}
		buf.Release()
	}
}

func publishCommand(cmd *Command) {
	commandPublishLocker.Lock()
	defer commandPublishLocker.Unlock()

	for _, observer := range commandPublishObservers {
		select {
		case observer <- cmd:
		default:
		}
	}
}

func SendCommand(conn *net.TCPConn, command *Command) error {
	js, err := ToJson(command.Item)

	if err != nil {
		return err
	}
	data := []byte(js)

	if err := WriteUInt(conn, KCP_DATA_BUF_SIZE+1); err != nil {
		return err
	}

	if err := WriteUInt(conn, uint32(len(data))+4); err != nil {
		return err
	}

	if err := WriteUInt(conn, uint32(command.Type)); err != nil {
		return err
	}

	if _, err := WriteData(data, conn, len(data)); err != nil {
		return err
	}

	return nil
}

func IsCommand(n uint32) bool {
	return n == KCP_DATA_BUF_SIZE+1
}

func CommandFromBuf(buf *BufItem) (*Command, error) {
	t, _ := ReadUInt(buf)
	s := string(buf.Data())

	var obj interface{}
	switch CommandType(t) {
	case Cmd_handshake:
		obj = &CommandHandshake{}
	case Cmd_ping:
		obj = &CommandPing{}
	default:
		return nil, fmt.Errorf("CommandFromBuf error type: %d", t)
	}

	if err := json.Unmarshal([]byte(s), obj); err != nil {
		return nil, err
	}
	return &Command{Type: CommandType(t), Item: obj}, nil
}

func ReadCommand(reader io.Reader) (*Command, error) {
	l, err := ReadUInt(reader)
	if err != nil {
		return nil, fmt.Errorf("ReadCommand read len err: %+v", err)
	}

	if l > 1024 {
		return nil, fmt.Errorf("ReadCommand cmd len err: %d", l)
	}

	buf := DataBufPool.Get()
	if _, err = buf.ReadNFromReader(reader, int(l)); err != nil {
		buf.Release()
		return nil, fmt.Errorf("ReadCommand ReadNFromReader err: %+v", err)
	}

	command, err := CommandFromBuf(buf)
	buf.Release()

	return command, err
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

type CommandPing struct {
	TunnelID int
	ID       int
	Step     int
	SendTime int64
}
