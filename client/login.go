package main

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

var (
	ErrLoginAuthority = errors.New("error username or password")
	ErrLoginAuthCode  = errors.New("error auth code")
)

func WaittingLogin(proxyList []*proxyclient.ProxyInfo) {
	log.Println("login ...")

	for _, p := range proxyList {
		client := proxyclient.NewProxyClient(p, config.Server)
		if client == nil {
			continue
		}

		if err := tryLogin(client); err == nil {
			log.Printf("login success %d\n", ClientID)
			return
		} else if err == ErrLoginAuthority || err == ErrLoginAuthCode {
			log.Printf("try login failed %s\n", p.Host)
			nctst.CheckError(err)
			return
		} else {
			log.Printf("try login failed %s %+v\n", p.Host, err)
		}

		log.Println("wait 5s to retry ...")
		time.Sleep(time.Second * 5)
	}
}

func tryLogin(client proxyclient.ProxyClient) error {
	err := client.Connect()
	if err != nil {
		return err
	}

	defer client.Close()

	client.SetDeadline(time.Now().Add(time.Second * 5))

	if err = sendLoginCommand(client); err != nil {
		return err
	}

	if err = receiveLoginReply(client); err != nil {
		return err
	}

	return nil
}

func sendLoginCommand(conn io.Writer) error {
	if err := nctst.WriteUInt(conn, nctst.NEW_CONNECTION_KEY); err != nil {
		return err
	}

	cmd := &nctst.CommandLogin{}
	cmd.AuthCode = authCode
	cmd.UserName = config.UserName
	cmd.PassWord = nctst.HashPassword(config.UserName, config.PassWord)
	cmd.ClientUUID = UUID
	cmd.Compress = config.Compress
	cmd.Key = config.Key
	log.Printf("**Do login code: %d, name: %s", cmd.AuthCode, cmd.UserName)
	return nctst.SendCommand(conn, &nctst.Command{Type: nctst.Cmd_login, Item: cmd})
}

func receiveLoginReply(conn io.Reader) error {
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

	if cmd.Code == nctst.LoginReply_errAuthCode {
		return ErrLoginAuthCode
	} else if cmd.Code == nctst.LoginReply_errAuthority {
		return ErrLoginAuthority
	}

	ClientID = cmd.ClientID
	connKey = cmd.ConnectKey

	return nil
}
