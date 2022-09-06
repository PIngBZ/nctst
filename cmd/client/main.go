package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/google/uuid"
	"github.com/xtaci/smux"
)

var (
	UUID     = uuid.NewString()
	ClientID uint

	configFile string
	config     *Config

	kcp        *nctst.Kcp
	duplicater *nctst.Duplicater
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

	go nctst.CommandDaemon()
}

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Listen)
	nctst.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	nctst.CheckError(err)

	if !Login() {
		return
	}

	kcp = nctst.NewKcp(ClientID)
	duplicater = nctst.NewDuplicater(config.Duplicate, kcp.OutputChan)

	smuxClient, err := smux.Client(nctst.NewCompStream(kcp), nctst.SmuxConfig())
	nctst.CheckError(err)

	startUpstreamProxies()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %+v\n", err)
			continue
		}

		log.Printf("AcceptTCP %s\n", conn.RemoteAddr().String())

		stream, err := smuxClient.OpenStream()
		if err != nil {
			conn.Close()
			log.Printf("AcceptTCP: %+v\n", err)
			continue
		}

		conn.SetDeadline(time.Time{})
		stream.SetDeadline(time.Time{})

		log.Printf("AcceptTCP transfer %s\n", conn.RemoteAddr().String())
		go nctst.Transfer(conn, stream)
	}
}

func startUpstreamProxies() {
	proxies = make([]*Proxy, len(config.Proxies))

	for i, serverIP := range config.Proxies {
		tunnel := nctst.NewOuterTunnel(uint(i), ClientID, duplicater.Output, kcp.InputChan)
		proxies[i] = NewProxy(uint(i), serverIP, kcp.InputChan, tunnel)
	}
}
