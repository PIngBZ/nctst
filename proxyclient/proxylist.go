package proxyclient

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
)

type ProxyType int

const (
	ProxyTypeSocks5 = iota
	ProxyTypeTrojan
)

func GetProxyList(proxyFile *ProxyFile, pingTarget *PingTarget, key, userName, password string) []*ProxyInfo {
	var proxyGroups *ProxyGroups
	if proxyFile.Type == "net" {
		proxyGroups = GetProxyListFromNet(proxyFile, key, userName, password)
	} else if proxyFile.Type == "file" {
		proxyGroups = GetProxyListFromFile(proxyFile)
	} else {
		return nil
	}

	return SelectProxyFromGroupsInfo(proxyGroups, pingTarget)
}

func GetProxyListFromNet(proxyFile *ProxyFile, key, userName, password string) *ProxyGroups {
	log.Printf("**Request proxy list from %s ...", proxyFile.Url)

	req, err := http.NewRequest("GET", proxyFile.Url, nil)
	if err != nil {
		log.Printf("GetProxyListFromNet create request %s %+v\n", proxyFile.Url, err)
		return nil
	}
	req.SetBasicAuth(userName, password)

	client := &http.Client{
		Timeout: time.Second * 15,
	}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("GetProxyListFromNet do request %s %+v\n", proxyFile.Url, err)
		return nil
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Printf("GetProxyListFromNet response statuscode %s %d\n", proxyFile.Url, response.StatusCode)
		return nil
	}

	buf := nctst.DataBufPool.Get()
	if _, err = buf.SetAllFromReader(response.Body); err != nil {
		log.Printf("GetProxyListFromNet read body %s %+v\n", proxyFile.Url, err)
		return nil
	}

	nctst.Xor(buf.Data()[4:], []byte(key))
	nctst.Xor(buf.Data()[4:], []byte(userName))

	return LoadProxyListFromData(buf.Data(), proxyFile.Password)
}

func GetProxyListFromFile(proxyFile *ProxyFile) *ProxyGroups {
	file, err := os.Open(proxyFile.Url)
	if err != nil {
		log.Printf("getProxyListFromFile %+v\n", err)
		return nil
	}

	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("getProxyListFromFile %+v\n", err)
		return nil
	}
	return LoadProxyListFromData(content, proxyFile.Password)
}

func LoadProxyListFromData(data []byte, password string) *ProxyGroups {
	nctst.Xor(data, []byte(password))

	var proxyGroups *ProxyGroups
	if err := json.Unmarshal(data, &proxyGroups); err != nil {
		log.Printf("loadProxyListFromData %+v\n", err)
		return nil
	}

	n := 0
	for _, group := range proxyGroups.Groups {
		n += len(group.List)
	}

	if n == 0 {
		log.Println("loadProxyListFromData no item")
		return nil
	}

	return proxyGroups
}

func SelectProxyFromGroupsInfo(proxyGroups *ProxyGroups, pingTarget *PingTarget) []*ProxyInfo {
	result := make([]*ProxyInfo, 0)

	selectNum := proxyGroups.SelectPerGroup
	for i, group := range proxyGroups.Groups {
		log.Printf("**Ping group: %s, %d/%d\n", group.Name, i+1, len(proxyGroups.Groups))
		some := SelectProxyFromGroup(group, selectNum, pingTarget, false)
		result = append(result, some...)
		selectNum = proxyGroups.SelectPerGroup + (selectNum - len(some))
	}

	return result
}

func SelectProxyFromGroup(group *ProxyGroup, num int, pingTarget *PingTarget, printDetails bool) []*ProxyInfo {
	workChan := make(chan *ProxyInfo, len(group.List))
	pingResultChan := make(chan nctst.Pair[uint32, *ProxyInfo], len(group.List))

	for _, proxy := range group.List {
		workChan <- proxy
	}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(group.List))

	if pingTarget.PingThreads == 0 {
		pingTarget.PingThreads = 1
	}
	for i := 0; i < pingTarget.PingThreads; i++ {
		go func() {
			for {
				work, ok := <-workChan
				if ok {
					client := NewProxyClient(work, pingTarget.Target)
					if client.Ping(client, printDetails, nil) {
						pingResultChan <- nctst.Pair[uint32, *ProxyInfo]{First: client.LastPing(), Second: work}
					}
				}
				waitGroup.Done()
			}
		}()
	}

	pingResult := make([]nctst.Pair[uint32, *ProxyInfo], 0)

	go func() {
		waitGroup.Wait()
		close(pingResultChan)
	}()

	n, total := 1, len(group.List)
	for p := range pingResultChan {
		pingResult = append(pingResult, p)
		log.Printf("%d/%d %dms %s\n", n, total, p.First, p.Second.Name)
		n++
	}
	sort.SliceStable(pingResult, func(i, j int) bool {
		return pingResult[i].First < pingResult[j].First
	})

	result := make([]*ProxyInfo, 0)
	for i, v := range pingResult {
		result = append(result, v.Second)

		log.Printf("*Ping proxy delay: %s %d\n", v.Second.Address(), v.First)

		if i >= num {
			break
		}
	}

	return result
}
