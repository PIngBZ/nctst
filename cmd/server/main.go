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

	UserMgr = &UserManager{}

	clients            = make(map[string]*Client)
	clientsLocker      = sync.Mutex{}
	nextClientID  uint = uint(rand.Intn(89999) + 10000)
)

func init() {
	nctst.OpenLog()

	flag.StringVar(&configFile, "c", "", "configure file")
	flag.Parse()

	if configFile == "" {
		nctst.CheckError(errors.New("no config file"))
	}

	var err error
	config, err = parseConfig(configFile)
	nctst.CheckError(err)

	go nctst.CommandDaemon(config.Key)
}

func main() {
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

	buf, err := nctst.ReadLBuf(conn)
	if err != nil {
		conn.Close()
		log.Printf("onNewConnection ReadHeader err: %+v\n", err)
		return
	}

	if !nctst.IsCommand(buf) {
		conn.Close()
		buf.Release()
		log.Println("onNewConnection not command")
		return
	}

	command, err := nctst.ReadCommand(buf)
	buf.Release()
	if err != nil {
		log.Printf("onNewConnection ReadCommand err: %+v\n", err)
		conn.Close()
		return
	}

	if command.Type == nctst.Cmd_login {
		cmd := command.Item.(*nctst.CommandLogin)

		if cmd.Key != config.Key {
			conn.Close()
			log.Println("login error key: " + cmd.Key)
			return
		}

		if !UserMgr.CheckUserPassword(cmd.UserName, cmd.PassWord) {
			sendLoginReply(conn, cmd.ClientUUID, 0, "", nctst.LoginReply_errAuthority)
			return
		}

		clientsLocker.Lock()
		client, ok := clients[cmd.ClientUUID]
		if ok {
			clientsLocker.Unlock()
			conn.Close()
			log.Println("login uuid exist: " + cmd.ClientUUID)
			return
		}

		client = NewClient(cmd.ClientUUID, nextClientID, cmd.Compress, cmd.Duplicate, cmd.TarType)
		nextClientID++
		clients[cmd.ClientUUID] = client
		clientsLocker.Unlock()

		sendLoginReply(conn, client.UUID, client.ID, client.ConnKey, nctst.LoginReply_success)
		conn.Close()
		log.Printf("login success %s %s %d\n", client.UUID, cmd.UserName, client.ID)
	} else if command.Type == nctst.Cmd_handshake {
		cmd := command.Item.(*nctst.CommandHandshake)

		if cmd.Key != config.Key {
			log.Println("handshake error key: " + cmd.Key)
			conn.Close()
			return
		}

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
	} else {
		conn.Close()
		log.Printf("onNewConnection cmd type err: %d\n", command.Type)
	}
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
