package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
)

var (
	configFile string
	config     *Config

	ProxyListData string
)

func init() {
	rand.Seed(time.Now().Unix())
	nctst.OpenLog()

	flag.StringVar(&configFile, "c", "", "configure file")
	flag.Parse()

	if configFile == "" {
		if exist, _ := nctst.PathExists("config.json"); !exist {
			nctst.CheckError(errors.New("no config file"))
		} else {
			configFile = "config.json"
		}
	}

	var err error
	config, err = parseConfig(configFile)
	nctst.CheckError(err)

	nctst.CommandXorKey = config.Password
}

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", config.Port))
	nctst.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	nctst.CheckError(err)

	go GenerateProxyList()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %+v\n", err)
			continue
		}

		log.Printf("AcceptTCP %s\n", conn.RemoteAddr().String())

		go doTransfer(conn, config.Key)
	}
}

func doTransfer(conn *net.TCPConn, key string) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 10))

	buf := make([]byte, len(key))
	l, err := conn.Read(buf)
	if err == nil {
		log.Printf("doTransfer: %+v\n", err)
		return
	}

	if l != len(key) {
		log.Println("read key len error")
		return
	}

	if string(buf) != key {
		log.Printf("key error: %s", string(buf))
		return
	}

	if err = nctst.WriteLString(conn, ProxyListData); err != nil {
		log.Printf("write data error: %+v\n", err)
		return
	}

	log.Println("return proxy list success")
}
