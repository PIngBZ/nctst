package main

import (
	"errors"
	"flag"
	"fmt"
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
	smuxServer *smux.Session
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

	smux, err := smux.Server(nctst.NewCompStream(k), nctst.SmuxConfig())
	nctst.CheckError(err)
	smuxServer = smux

	duplicater = nctst.NewDuplicater(1, k.OutputChan, tunnels)

	go smuxLoop()
	go nctst.CommandDaemon()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %+v\n", err)
			continue
		}

		go newOuterConn(conn)
	}
}

func newOuterConn(conn *net.TCPConn) {
	tunnelID, connID, err := checkInitCommand(conn)
	if err != nil {
		log.Printf("checkHandshakeError %s %+v\n", conn.RemoteAddr().String(), err)
		conn.Close()
		return
	}

	log.Printf("new connection %s %d %d", conn.RemoteAddr().String(), tunnelID, connID)

	conn.SetDeadline(time.Time{})

	if tunnel_i, ok := tunnels.Load(int(tunnelID)); ok {
		tunnel := tunnel_i.(*nctst.OuterTunnel)
		outerconn := nctst.NewOuterConnection(int(tunnelID), int(connID), conn, k.InputChan, tunnel.OutputChan)
		tunnel.Add(int(connID), outerconn)
	} else {
		tunnel := nctst.NewOuterTunnel(int(tunnelID))
		if tunnel_i, ok := tunnels.LoadOrStore(int(tunnelID), tunnel); ok {
			tunnel = tunnel_i.(*nctst.OuterTunnel)
		} else {
			tunnel.Run()
		}

		outerconn := nctst.NewOuterConnection(int(tunnelID), int(connID), conn, k.InputChan, tunnel.OutputChan)
		tunnel.Add(int(connID), outerconn)
	}
}

func checkInitCommand(conn *net.TCPConn) (int, int, error) {
	if t, err := nctst.ReadUInt(conn); err != nil {
		return 0, 0, fmt.Errorf("checkInitCommand ReadHeader err: %+v", err)
	} else if !nctst.IsCommand(t) {
		return 0, 0, fmt.Errorf("checkInitCommand not command: %d", t)
	}

	command, err := nctst.ReadCommand(conn)
	if err != nil {
		return 0, 0, fmt.Errorf("checkInitCommand ReadCommand err: %+v", err)
	}

	if command.Type != nctst.Cmd_handshake {
		return 0, 0, fmt.Errorf("checkInitCommand cmd type err: %d", command.Type)
	}

	cmd := command.Item.(*nctst.CommandHandshake)

	log.Printf("key: %s", cmd.Key)
	if cmd.Key != config.Key {
		return 0, 0, errors.New("error key: " + cmd.Key)
	}

	duplicater.SetNum(cmd.Duplicate)

	return cmd.TunnelID, cmd.ConnID, nil
}

func smuxLoop() {
	for {
		stream, err := smuxServer.AcceptStream()
		if err != nil {
			log.Printf("accept stream: %+v\n", err)
			continue
		}

		log.Printf("AcceptStream %+v\n", stream.ID())
		go connectTarget(stream)
	}
}

func connectTarget(stream *smux.Stream) {
	target, err := net.DialTimeout("tcp", config.Target, time.Second*3)
	if err != nil {
		log.Printf("connectTarget: %+v\n", err)
		stream.Close()
		return
	}

	target.SetDeadline(time.Time{})
	stream.SetDeadline(time.Time{})

	log.Printf("transfer %+v\n", stream.ID())
	go nctst.Transfer(stream, target)
}
