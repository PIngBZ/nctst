package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/socks5"
	"github.com/xtaci/smux"
)

func startMapTargetsLoop(smuxClient *smux.Session, targets []*nctst.AddrInfo) {
	port := 2000 + rand.Intn(3000)
	log.Printf("\n++++++++++Preparing map local port to remote address++++++++++\n\n")
	defer log.Println("\n\n----------map local port end----------")

	for _, target := range targets {
		for {
			addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				log.Printf("startMapTargetsLoop ResolveTCPAddr %+v\n", err)
				break
			}

			listener, err := net.ListenTCP("tcp", addr)
			if err != nil {
				port++
				continue
			}

			log.Printf("**Local [:%d] <----------> remote %s\n", port, target.Address())
			go mapTargetLoop(smuxClient, target, listener)
			break
		}
	}
}

func mapTargetLoop(smuxClient *smux.Session, target *nctst.AddrInfo, listener *net.TCPListener) {
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("mapTargetLoop AcceptTCP: %+v\n", err)
			continue
		}

		log.Printf("mapTargetLoop AcceptTCP %s\n", conn.RemoteAddr().String())

		go mapTargetDoTransfer(conn, smuxClient, target)
	}
}

func mapTargetDoTransfer(conn *net.TCPConn, smuxClient *smux.Session, target *nctst.AddrInfo) {

	client := socks5.Client{
		HandshakeTimeout: time.Second * 5,
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: socks5.NoAuth{},
		},
		Dialer: func(client *socks5.Client, request *socks5.Request) (net.Conn, error) {
			return smuxClient.OpenStream()
		},
	}

	upConn, err := client.Connect(socks5.Version5, target.Address())
	if err != nil {
		conn.Close()
		log.Printf("mapTargetDoTransfer Connect: %+v\n", err)
		return
	}

	log.Printf("mapTargetDoTransfer Transfer %s\n", conn.RemoteAddr().String())
	go nctst.Transfer(conn, upConn)
}
