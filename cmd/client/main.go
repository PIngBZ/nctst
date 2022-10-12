package main

import (
	"errors"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/cmd/client/proxylist"
	"github.com/google/uuid"
	"github.com/xtaci/smux"
)

var (
	UUID     = uuid.NewString()
	ClientID uint

	authCode   int
	configFile string
	config     *Config

	kcp        *nctst.Kcp
	proxies    = []*Proxy{}
	tunnels    = make([]*nctst.OuterTunnel, 0)
	duplicater *nctst.Duplicater

	connKey string
)

func init() {
	rand.Seed(time.Now().Unix())
	nctst.OpenLog()

	flag.IntVar(&authCode, "d", 0, "auth code")
	flag.StringVar(&configFile, "c", "", "configure file")
	flag.Parse()

	if authCode == 0 {
		log.Println("Attention, no auth code. Only test environment can work.")
	}

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
}

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Listen)
	nctst.CheckError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	nctst.CheckError(err)

	WaittingLogin()

	kcp = nctst.NewKcp(ClientID)
	duplicater = nctst.NewDuplicater(config.Duplicate, kcp.OutputChan, func(v uint32) (uint32, []*nctst.OuterTunnel) { return 100, tunnels })

	var smuxClient *smux.Session
	if config.Compress {
		smuxClient, err = smux.Client(nctst.NewCompStream(kcp), nctst.SmuxConfig())
	} else {
		smuxClient, err = smux.Client(kcp, nctst.SmuxConfig())
	}
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
	var proxyList []string
	if len(config.Proxies) != 0 {
		proxyList = config.Proxies
	} else if len(config.ProxyFile) > 0 {
		proxyList = proxylist.GetProxyList(config.ProxyFile)
		if len(proxyList) == 0 {
			nctst.CheckError(errors.New("can not get proxy server list"))
		}
	} else {
		nctst.CheckError(errors.New("no proxy server"))
	}

	proxies = make([]*Proxy, len(proxyList))

	for i, proxyIP := range config.Proxies {
		tunnel := nctst.NewOuterTunnel(config.Key, uint(i), ClientID, kcp.InputChan, duplicater.Output)
		proxies[i] = NewProxy(uint(i), proxyIP, tunnel)
		tunnels = append(tunnels, tunnel)
	}
}
