package main

import (
	"errors"
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
)

var (
	configFile string
	config     *Config

	clients                  = make(map[string]*Client)
	clientUserNameIndex      = make(map[string]*Client)
	clientsLocker            = sync.Mutex{}
	nextClientID        uint = uint(rand.Intn(89999) + 10000)
)

func init() {
	rand.Seed(time.Now().Unix())
	nctst.OpenLog()

	flag.StringVar(&configFile, "c", "", "configure file")
	flag.Parse()

	if configFile == "" {
		nctst.CheckError(errors.New("no config file"))
	}

	var err error
	config, err = parseConfig(configFile)
	nctst.CheckError(err)

	nctst.CommandXorKey = config.Key
}

func main() {
	createAminUser()

	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Listen)
	nctst.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	nctst.CheckError(err)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %+v\n", err)
			continue
		}

		go onNewConnection(conn)
	}
}

func onNewConnection(conn *net.TCPConn) {
	conn.SetDeadline(time.Now().Add(time.Second * 5))

	if k, err := nctst.ReadUInt(conn); err != nil {
		nctst.DelayClose(conn)
		log.Printf("onNewConnection ReadKey err: %+v\n", err)
		return
	} else if k != nctst.NEW_CONNECTION_KEY {
		nctst.DelayClose(conn)
		log.Printf("onNewConnection error key %d\n", k)
		return
	}

	buf, err := nctst.ReadLBuf(conn)
	if err != nil {
		nctst.DelayClose(conn)
		log.Printf("onNewConnection ReadHeader err: %+v\n", err)
		return
	}

	if !nctst.IsCommand(buf) {
		nctst.DelayClose(conn)
		buf.Release()
		log.Println("onNewConnection not command")
		return
	}

	command, err := nctst.ReadCommand(buf)
	buf.Release()
	if err != nil {
		log.Printf("onNewConnection ReadCommand err: %+v\n", err)
		nctst.DelayClose(conn)
		return
	}
	if command.Type == nctst.Cmd_idle {
		// do nothing
	} else if command.Type == nctst.Cmd_testping {
		nctst.SendCommand(conn, command)
		if buf, err := nctst.ReadLBuf(conn); err != nil {
			nctst.DelayClose(conn)
		} else if cmd, err := nctst.ReadCommand(buf); err != nil {
			nctst.DelayClose(conn)
		} else {
			nctst.SendCommand(conn, cmd)
		}
	} else if command.Type == nctst.Cmd_login {
		doLogin(conn, command)
	} else if command.Type == nctst.Cmd_handshake {
		doHandshake(conn, command)
	} else {
		nctst.DelayClose(conn)
		log.Printf("onNewConnection cmd type err: %d\n", command.Type)
	}
}

func doLogin(conn *net.TCPConn, command *nctst.Command) {
	defer conn.Close()

	cmd := command.Item.(*nctst.CommandLogin)

	if cmd.Key != config.Key {
		log.Println("login error key: " + cmd.Key)
		return
	}

	if !UserMgr.CheckAuthCode(cmd.UserName, cmd.AuthCode) {
		sendLoginReply(conn, cmd.ClientUUID, 0, "", nctst.LoginReply_errAuthCode)
		return
	}

	if !UserMgr.CheckUserPassword(cmd.UserName, cmd.PassWord) {
		sendLoginReply(conn, cmd.ClientUUID, 0, "", nctst.LoginReply_errAuthority)
		return
	}

	clientsLocker.Lock()

	if _, ok := clients[cmd.ClientUUID]; ok {
		clientsLocker.Unlock()
		log.Println("login uuid exist: " + cmd.ClientUUID)
		return
	}

	if client, ok := clientUserNameIndex[cmd.UserName]; ok {
		client.Close()
		delete(clients, client.UUID)
		delete(clientUserNameIndex, cmd.UserName)
	}

	user, _ := UserMgr.GetUser(cmd.UserName)

	client := NewClient(user, cmd.ClientUUID, nextClientID, cmd.Compress)
	nextClientID++
	clients[cmd.ClientUUID] = client
	clientUserNameIndex[cmd.UserName] = client

	clientsLocker.Unlock()

	sendLoginReply(conn, client.UUID, client.ID, client.ConnKey, nctst.LoginReply_success)

	log.Printf("login success %s %s %d\n", client.UUID, cmd.UserName, client.ID)
}

func doHandshake(conn *net.TCPConn, command *nctst.Command) {
	cmd := command.Item.(*nctst.CommandHandshake)

	clientsLocker.Lock()
	client, ok := clients[cmd.ClientUUID]
	clientsLocker.Unlock()

	if !ok {
		sendHandshakeReply(conn, cmd.ClientUUID, nctst.HandshakeReply_needlogin)
		conn.Close()
		log.Printf("handshake not login: %s %d %d %d\n", cmd.ClientUUID, cmd.ClientID, cmd.TunnelID, cmd.ConnID)
		return
	}

	if cmd.ConnectKey != client.ConnKey {
		conn.Close()
		log.Printf("handshake key error: %s %d %d %d\n", cmd.ClientUUID, cmd.ClientID, cmd.TunnelID, cmd.ConnID)
		return
	}

	sendHandshakeReply(conn, cmd.ClientUUID, nctst.HandshakeReply_success)

	conn.SetDeadline(time.Time{})

	client.AddConn(conn, cmd.TunnelID, cmd.ConnID)
}

func sendLoginReply(conn *net.TCPConn, uuid string, id uint, connKey string, code nctst.LoginReply_Code) {
	cmd := &nctst.CommandLoginReply{}
	cmd.ClientID = id
	cmd.ClientUUID = uuid
	cmd.ConnectKey = connKey
	cmd.Code = code
	nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_loginReply, Item: cmd})
}

func sendHandshakeReply(conn *net.TCPConn, uuid string, code nctst.HandshakeReply_Code) {
	cmd := &nctst.CommandHandshakeReply{}
	cmd.ClientUUID = uuid
	cmd.Code = code
	nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_handshakeReply, Item: cmd})
}
