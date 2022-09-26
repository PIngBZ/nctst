package main

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/haochen233/socks5"
)

func WaittingLogin() {
	log.Println("login ...")
	for {
		for _, serverIP := range config.Proxies {
			if err := tryLogin(serverIP); err != nil {
				log.Printf("try login failed %s %+v\n", serverIP, err)
			} else {
				log.Printf("login success %d\n", ClientID)
				return
			}
		}
		log.Println("login failed, wait 5s to retry ...")
		time.Sleep(time.Second * 5)
	}
}

func tryLogin(addr string) error {
	client := socks5.Client{
		ProxyAddr: addr,
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: &socks5.NoAuth{},
		},
	}

	conn, err := client.Connect(socks5.Version5, config.ServerIP+":"+config.ServerPort)
	if err != nil {
		return err
	}

	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	if err = sendLogin(conn); err != nil {
		return err
	}

	if err = receiveLoginReply(conn); err != nil {
		return err
	}

	return nil
}

func sendLogin(conn *net.TCPConn) error {
	if err := nctst.WriteUInt(conn, nctst.NEW_CONNECTION_KEY); err != nil {
		return err
	}

	cmd := &nctst.CommandLogin{}
	cmd.ClientUUID = UUID
	cmd.Duplicate = config.Duplicate
	cmd.Compress = config.Compress
	cmd.Key = config.Key
	return nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_login, Item: cmd})
}

func receiveLoginReply(conn *net.TCPConn) error {
	buf, err := nctst.ReadLBuf(conn)
	if err != nil {
		return err
	}

	if nctst.GetCommandType(buf) != nctst.Cmd_loginReply {
		buf.Release()
		return errors.New("receiveLoginReply type error")
	}

	command, err := nctst.ReadCommand(buf)
	buf.Release()
	if err != nil {
		return err
	}
	cmd := command.Item.(*nctst.CommandLoginReply)

	ClientID = cmd.ClientID

	return nil
}
