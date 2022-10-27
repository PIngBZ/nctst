package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

type ProxyType int

const (
	ProxyTypeSocks5 = iota
	ProxyTypeTrojan
)

type ProxyFileInfo struct {
	Type     string `json:"type"`
	Url      string `json:"url"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

func GetProxyList(proxyFile *ProxyFileInfo) []*proxyclient.ProxyInfo {
	var proxyListInfo *proxyclient.ProxyFileInfo
	if proxyFile.Type == "net" {
		proxyListInfo = getProxyListFromNet(proxyFile)
	} else if proxyFile.Type == "file" {
		proxyListInfo = getProxyListFromFile(proxyFile)
	} else {
		return nil
	}

	return selectProxyFromFileInfo(proxyListInfo)
}

func getProxyListFromNet(proxyFile *ProxyFileInfo) *proxyclient.ProxyFileInfo {
	log.Printf("**Request proxy list from %s ...", proxyFile.Url)
	conn, err := net.DialTimeout("tcp", proxyFile.Url, time.Second*10)
	if err != nil {
		log.Printf("getProxyListFromNet %+v\n", err)
		return nil
	}

	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 10))
	if _, err := conn.Write([]byte(proxyFile.Key)); err != nil {
		log.Printf("getProxyListFromNet %+v\n", err)
		return nil
	}

	buf, err := nctst.ReadLBuf(conn)
	if err != nil {
		log.Printf("getProxyListFromNet %+v\n", err)
		return nil
	}

	return loadProxyListFromData(buf.Data(), proxyFile.Password)
}

func getProxyListFromFile(proxyFile *ProxyFileInfo) *proxyclient.ProxyFileInfo {
	file, err := os.Open(proxyFile.Url)
	if err != nil {
		log.Printf("getProxyListFromFile %+v\n", err)
		return nil
	}

	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("getProxyListFromFile %+v\n", err)
		return nil
	}

	return loadProxyListFromData(content, proxyFile.Password)
}

func loadProxyListFromData(data []byte, password string) *proxyclient.ProxyFileInfo {
	nctst.Xor(data, []byte(password))

	var proxyFileInfo *proxyclient.ProxyFileInfo
	if err := json.Unmarshal(data, &proxyFileInfo); err != nil {
		log.Printf("loadProxyListFromData %+v\n", err)
		return nil
	}

	n := 0
	for _, group := range proxyFileInfo.ProxyGroups {
		n += len(group.ProxyList)
	}

	if n == 0 {
		log.Println("loadProxyListFromData no item")
		return nil
	}

	return proxyFileInfo
}

func selectProxyFromFileInfo(proxyListInfo *proxyclient.ProxyFileInfo) []*proxyclient.ProxyInfo {
	result := make([]*proxyclient.ProxyInfo, 0)

	selectNum := proxyListInfo.SelectPerGroup
	for _, group := range proxyListInfo.ProxyGroups {
		some := selectProxyFromGroup(group, selectNum)
		result = append(result, some...)
		selectNum = proxyListInfo.SelectPerGroup + (selectNum - len(some))
	}

	return result
}

func selectProxyFromGroup(group *proxyclient.ProxyListGroup, num int) []*proxyclient.ProxyInfo {
	workChan := make(chan *proxyclient.ProxyInfo, len(group.ProxyList))
	pingResultChan := make(chan nctst.Pair[uint32, *proxyclient.ProxyInfo], len(group.ProxyList))

	for _, proxy := range group.ProxyList {
		workChan <- proxy
	}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(group.ProxyList))

	for i := 0; i < 5; i++ {
		go func() {
			work, ok := <-workChan
			if ok {
				client := proxyclient.NewProxyClient(work, config.Server)
				if client.Ping(nil) {
					pingResultChan <- nctst.Pair[uint32, *proxyclient.ProxyInfo]{First: client.LastPing(), Second: work}
				}
			}
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()

	close(pingResultChan)

	pingResult := make([]nctst.Pair[uint32, *proxyclient.ProxyInfo], 0)

	for p := range pingResultChan {
		pingResult = append(pingResult, p)
	}

	sort.SliceStable(pingResult, func(i, j int) bool {
		return pingResult[i].First < pingResult[j].First
	})

	result := make([]*proxyclient.ProxyInfo, 0)
	for i, v := range pingResult {
		result = append(result, v.Second)

		log.Printf("*Ping proxy delay: %s %d\n", v.Second.Address(), v.First)

		if i >= num {
			break
		}
	}

	return result
}
