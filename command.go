package nctst

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

const (
	commandSignHeader uint32 = 0xf1f121
)

var (
	CommandXorKey string
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

type CommandManager struct {
	CommandReceiveChan chan *BufItem

	commandPublishObservers []chan *Command
	commandPublishLocker    sync.Mutex

	Die chan struct{}

	dieOnce sync.Once
}

func NewCommandManager() *CommandManager {
	h := &CommandManager{}
	h.CommandReceiveChan = make(chan *BufItem, 8)
	h.commandPublishObservers = make([]chan *Command, 0)

	h.Die = make(chan struct{})

	go h.commandDaemon()
	return h
}

func (h *CommandManager) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.Die)
		once = true
	})

	if !once {
		return
	}

	h.commandPublishLocker.Lock()
	defer h.commandPublishLocker.Unlock()

	h.commandPublishObservers = nil
}

func (h *CommandManager) AttachCommandObserver(observer chan *Command) {
	h.commandPublishLocker.Lock()
	defer h.commandPublishLocker.Unlock()

	h.commandPublishObservers = append(h.commandPublishObservers, observer)
}

func (h *CommandManager) DetachCommandObserver(observer chan *Command) {
	h.commandPublishLocker.Lock()
	defer h.commandPublishLocker.Unlock()

	for idx, item := range h.commandPublishObservers {
		if item == observer {
			h.commandPublishObservers = append(h.commandPublishObservers[:idx], h.commandPublishObservers[idx+1:]...)
			return
		}
	}
}

func (h *CommandManager) commandDaemon() {
	for {
		select {
		case <-h.Die:
			return
		case buf := <-h.CommandReceiveChan:
			if cmd, err := ReadCommand(buf); err == nil {
				h.publishCommand(cmd)
			} else {
				log.Printf("CommandDaemon CommandFromBuf error: %+v %d\n", err, buf.Size())
			}
			buf.Release()
		}
	}
}

func (h *CommandManager) publishCommand(cmd *Command) {
	h.commandPublishLocker.Lock()
	defer h.commandPublishLocker.Unlock()

	for _, observer := range h.commandPublishObservers {
		select {
		case <-h.Die:
			return
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
	Xor(data, []byte(CommandXorKey))

	if err := WriteUInt(conn, uint32(len(js)+8)); err != nil {
		return err
	}

	if err := WriteUInt(conn, commandSignHeader); err != nil {
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
	if ToUint(buf.Data()[:4]) != commandSignHeader {
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
	if sign, _ := ReadUInt(buf); sign != commandSignHeader {
		return nil, fmt.Errorf("CommandSignHeader error %d", sign)
	}

	t, _ := ReadUInt(buf)
	Xor(buf.Data(), []byte(CommandXorKey))
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
	AuthCode   int
	ClientUUID string
	UserName   string
	PassWord   string
	Compress   bool
	Key        string
}

type LoginReply_Code uint32

const (
	LoginReply_success LoginReply_Code = iota
	LoginReply_errAuthCode
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
