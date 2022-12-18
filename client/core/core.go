package core

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/google/uuid"
	"github.com/xtaci/smux"
)

var (
	UUID     = uuid.NewString()
	ClientID uint

	authCode int
	config   *Config

	listener     *net.TCPListener
	proxyListMgr = NewProxyListManager()
	kcp          *nctst.Kcp
	smuxClient   *smux.Session
	duplicater   *nctst.Duplicater

	proxyServers = []*ProxyServer{}

	connKey string
)

func Start(cfg *Config, code int) error {
	Status.setStat(ClientStatusStep_Init)

	var success bool
	defer func() {
		if !success {
			Stop()
		}
	}()

	config = cfg
	authCode = code

	Status.setStat(ClientStatusStep_GetProxyList)
	proxyListMgr = NewProxyListManager()
	if err := proxyListMgr.Init(); err != nil {
		return err
	}

	Status.setStat(ClientStatusStep_Login)
	err := WaittingLogin()
	if err != nil {
		return err
	}

	Status.setStat(ClientStatusStep_StartUpstream)

	kcp = nctst.NewKcp(ClientID)

	duplicater = nctst.NewDuplicater(kcp.OutputChan, func(v uint32) (uint32, []*nctst.OuterTunnel) {
		tunnels := make([]*nctst.OuterTunnel, 0, len(proxyServers))
		for _, proxyServer := range proxyServers {
			tunnels = append(tunnels, proxyServer.tunnel)
		}
		return 100, tunnels
	})

	startUpstreamProxies()

	if config.Compress {
		smuxClient, err = smux.Client(nctst.NewCompStream(kcp), nctst.SmuxConfig())
	} else {
		smuxClient, err = smux.Client(kcp, nctst.SmuxConfig())
	}
	if err != nil {
		return err
	}

	Status.setStat(ClientStatusStep_StartMapLocal)
	startMapTargetsLoop(smuxClient, config.MapTargets)

	Status.setStat(ClientStatusStep_StartLocalService)
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Listen)
	if err != nil {
		return err
	}

	listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Printf("main AcceptTCP exit: %+v\n", err)
				return
			}

			log.Printf("main AcceptTCP %s\n", conn.RemoteAddr().String())

			go doTransfer(conn, smuxClient)
		}
	}()

	Status.setStat(ClientStatusStep_CheckingConnection)

	CheckConnection() // ignore first request
	time.Sleep(time.Second)

	err, delay := CheckConnection()
	if err != nil {
		return fmt.Errorf("CheckConnection %+v", err)
	}

	Status.setPing(delay)
	Status.setStat(ClientStatusStep_Running)
	success = true
	log.Printf("Start finished, socks5 listening: %s\n\n", config.Listen)
	return nil
}

func Stop() {
	for _, proxyServer := range proxyServers {
		if proxyServer != nil {
			proxyServer.SendLogout()
		}
	}

	if listener == nil {
		listener.Close()
		listener = nil
	}

	closeAllMapTargets()

	if duplicater != nil {
		duplicater.Close()
		duplicater = nil
	}

	stopUpstreamProxies()

	if smuxClient != nil {
		smuxClient.Close()
		smuxClient = nil
	}

	if kcp != nil {
		kcp.Close()
		kcp = nil
	}

	if proxyListMgr != nil {
		proxyListMgr.Release()
		proxyListMgr = nil
	}
}

func doTransfer(conn *net.TCPConn, smuxClient *smux.Session) {
	stream, err := smuxClient.OpenStream()
	if err != nil {
		conn.Close()
		log.Printf("main doTransfer OpenStream: %+v\n", err)
		return
	}

	conn.SetDeadline(time.Time{})
	stream.SetDeadline(time.Time{})

	log.Printf("main doTransfer Transfer %s\n", conn.RemoteAddr().String())
	go nctst.Transfer(conn, stream)
}

func startUpstreamProxies() {
	proxyServers = make([]*ProxyServer, proxyListMgr.SelectNum)
	for i := 0; i < proxyListMgr.SelectNum; i++ {
		proxyServers[i] = NewProxyServer(uint(i))
	}
}

func stopUpstreamProxies() {
	if proxyServers == nil {
		return
	}

	for _, proxyServer := range proxyServers {
		if proxyServer != nil {
			proxyServer.Close()
		}
	}
	proxyServers = nil
}

func CheckConnection() (error, int) {
	var proxy string
	if config.Listen[0] == ':' {
		proxy = "socks5://127.0.0.1" + config.Listen
	} else {
		proxy = "socks5://" + config.Listen
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: func(_ *http.Request) (*url.URL, error) {
				return url.Parse(proxy)
			},
		},
		Timeout: time.Second * 15,
	}

	start := time.Now().UnixNano()
	req, err := http.NewRequest("GET", PingURL+fmt.Sprintf("?t=%d", start), nil)
	if err != nil {
		return err, 0
	}

	req.SetBasicAuth(config.UserName, config.PassWord)

	response, err := httpClient.Do(req)
	if err != nil {
		return err, 0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("error, StatusCode = %d", response.StatusCode), 0
	}

	io.Copy(io.Discard, response.Body)

	delay := (time.Now().UnixNano() - start) / 1e6
	log.Printf("***All successed, ping = %d", delay)

	return nil, int(delay)
}
