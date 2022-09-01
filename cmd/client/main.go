package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"sync"
	"time"

	"github.com/dearzhp/nctst"
	"github.com/xtaci/smux"
)

var (
	configFile string
	config     *Config
	k          = nctst.NewKcp(10001)
	duplicater *nctst.Duplicater
	tunnels    = &sync.Map{}
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
}

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Listen)
	nctst.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	nctst.CheckError(err)

	smuxClient, err := smux.Client(nctst.NewCompStream(k), nctst.SmuxConfig())
	nctst.CheckError(err)

	duplicater = nctst.NewDuplicater(config.Duplicate, k.OutputChan, tunnels)
	duplicater.SetNum(config.Duplicate)

	go nctst.CommandDaemon()
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

	for i, server := range config.Proxies {
		tunnel := nctst.NewOuterTunnel(i)
		tunnels.Store(i, tunnel)
		proxies[i] = NewProxy(i, server, k.InputChan, tunnel)
	}
}
