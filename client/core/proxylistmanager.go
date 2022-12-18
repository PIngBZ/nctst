package core

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
	mapset "github.com/deckarep/golang-set/v2"
)

type ProxyListManager struct {
	All       []*proxyclient.ProxyInfo
	AllIdx    mapset.Set[string]
	UsingIdx  mapset.Set[string]
	SelectNum int
	version   string

	Locker sync.Mutex

	die     chan struct{}
	dieOnce sync.Once
}

func NewProxyListManager() *ProxyListManager {
	h := &ProxyListManager{}

	h.AllIdx = mapset.NewSet[string]()
	h.UsingIdx = mapset.NewSet[string]()

	h.die = make(chan struct{})
	return h
}

func (h *ProxyListManager) Init() error {
	if err, _ := h.requestProxyList(); err != nil {
		return err
	}

	if config.ProxyFile.Type == "net" {
		go h.daemon()
	}

	return nil
}

func (h *ProxyListManager) requestProxyList() (error, bool) {
	if config.ProxyFile == nil || len(config.ProxyFile.Url) == 0 {
		return errors.New("no proxy config"), false
	}

	SelectNum, All, version := proxyclient.GetProxyList(config.ProxyFile, &proxyclient.PingTarget{Target: config.Server, PingThreads: 5}, config.Key, config.UserName, config.PassWord)

	if len(All) == 0 {
		return errors.New("GetProxyList return empty"), false
	}

	if config.ProxyFile.Type == "net" && version == h.version {
		return errors.New("no new version proxy list"), false
	}

	log.Printf("**Found %d items from server %s\n", len(All), config.ProxyFile.Url)

	if len(All) == 0 || SelectNum == 0 {
		return errors.New("GetProxyList return too small"), false
	}

	h.Locker.Lock()
	defer h.Locker.Unlock()

	h.All = All
	h.SelectNum = nctst.Min(len(All), SelectNum)

	h.AllIdx.Clear()
	for _, v := range h.All {
		h.AllIdx.Add(v.Address())
	}

	return nil, true
}

func (h *ProxyListManager) Release() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.die)
		once = true
	})

	if !once {
		return
	}
}

func (h *ProxyListManager) Get() *proxyclient.ProxyInfo {
	h.Locker.Lock()
	defer h.Locker.Unlock()

	for _, v := range h.All {
		if !h.UsingIdx.Contains(v.Address()) {
			h.UsingIdx.Add(v.Address())
			return v
		}
	}
	return nil
}

func (h *ProxyListManager) Put(proxy *proxyclient.ProxyInfo) {
	h.Locker.Lock()
	defer h.Locker.Unlock()

	h.UsingIdx.Remove(proxy.Address())

	if h.AllIdx.Contains(proxy.Address()) && proxy.PingTime.Add(time.Hour).Before(time.Now()) {
		go func() {
			client := proxyclient.NewProxyClient(proxy, config.Server)
			client.Ping(client, false, nil)

			h.Locker.Lock()
			defer h.Locker.Unlock()

			sort.SliceStable(h.All, func(i, j int) bool {
				return h.All[i].Ping < h.All[j].Ping
			})
		}()
	}
}

func (h *ProxyListManager) daemon() {
	ticker := time.NewTicker(time.Hour * 6)
	for {
		select {
		case <-h.die:
			return
		case <-ticker.C:
			if err, updated := h.requestProxyList(); err == nil && updated {
				for _, proxyServer := range proxyServers {
					if proxyServer != nil {
						if !h.AllIdx.Contains(proxyServer.proxy.Address()) {
							proxyServer.ChangeProxy()
							time.Sleep(time.Second * 30)
						}
					}
				}
			}
		}
	}
}
