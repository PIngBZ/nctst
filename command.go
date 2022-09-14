package nctst

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

var (
	CommandSignHeader  uint32 = 0xf1f121
	CommandReceiveChan        = make(chan *BufItem, 8)

	commandPublishObservers = make([]chan *Command, 0)
	commandPublishLocker    = sync.Mutex{}
	commandXorKey           string
)

type CommandType uint32

const (
	_ CommandType = iota

	Cmd_none

	Cmd_idle
	Cmd_testping
	Cmd_login
	Cmd_loginReply
	Cmd_handshake
	Cmd_handshakeReply
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

func CommandDaemon(key string) {
	commandXorKey = key
	for buf := range CommandReceiveChan {
		if cmd, err := ReadCommand(buf); err == nil {
			publishCommand(cmd)
		} else {
			log.Printf("CommandDaemon CommandFromBuf error: %+v %d\n", err, buf.Size())
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

func SendCommand(conn io.Writer, command *Command) error {
	js, err := ToJson(command.Item)

	if err != nil {
		return err
	}
	data := []byte(js)
	Xor(data, []byte(commandXorKey))

	if err := WriteUInt(conn, uint32(len(js)+8)); err != nil {
		return err
	}

	if err := WriteUInt(conn, CommandSignHeader); err != nil {
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

func IsCommand(buf *BufItem) bool {
	if buf.Size() < 16 {
		return false
	}
	if ToUint(buf.Data()[:4]) != CommandSignHeader {
		return false
	}
	if ToUint(buf.Data()[4:8]) >= uint32(Cmd_max) {
		return false
	}
	return true
}

func GetCommandType(buf *BufItem) CommandType {
	if !IsCommand(buf) {
		return Cmd_none
	}

	t := ToUint(buf.Data()[4:8])

	if t >= uint32(Cmd_max) {
		return Cmd_none
	}
	return CommandType(t)
}

func ReadCommand(buf *BufItem) (*Command, error) {
	if sign, _ := ReadUInt(buf); sign != CommandSignHeader {
		return nil, fmt.Errorf("CommandSignHeader error %d", sign)
	}

	t, _ := ReadUInt(buf)
	Xor(buf.Data(), []byte(commandXorKey))
	s := string(buf.Data())

	var obj interface{}
	switch CommandType(t) {
	case Cmd_idle:
		obj = &CommandIdle{}
	case Cmd_testping:
		obj = &CommandTestPing{}
	case Cmd_login:
		obj = &CommandLogin{}
	case Cmd_loginReply:
		obj = &CommandLoginReply{}
	case Cmd_handshake:
		obj = &CommandHandshake{}
	case Cmd_handshakeReply:
		obj = &CommandHandshakeReply{}
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

type Command struct {
	Type CommandType
	Item interface{}
}

type CommandIdle struct {
	Payload string
}

type CommandTestPing struct {
	SendTime int64
}

type CommandLogin struct {
	ClientUUID string
	UserName   string
	PassWord   string
	Duplicate  int
	Compress   bool
	TarType    string
	Key        string
}

type LoginReply_Code uint32

const (
	LoginReply_success LoginReply_Code = iota
	LoginReply_errAuthority
)

type CommandLoginReply struct {
	Code       LoginReply_Code
	ClientUUID string
	ClientID   uint
	ConnectKey string
}

type CommandHandshake struct {
	ClientUUID string
	ClientID   uint
	TunnelID   uint
	ConnID     uint
	ConnectKey string
	Key        string
}

type HandshakeReply_Code uint32

const (
	HandshakeReply_success HandshakeReply_Code = iota
	HandshakeReply_needlogin
)

type CommandHandshakeReply struct {
	ClientUUID string
	Code       HandshakeReply_Code
}

type CommandPing struct {
	ClientID uint
	TunnelID uint
	ID       uint
	Step     uint
	SendTime int64
}
