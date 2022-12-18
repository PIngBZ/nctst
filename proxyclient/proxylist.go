package proxyclient

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/hashicorp/go-retryablehttp"
)

type ProxyType int

func GetProxyList(proxyFile *ProxyFile, pingTarget *PingTarget, key, userName, password string) (int, []*ProxyInfo, string) {
	var proxyGroups *ProxyGroups
	if proxyFile.Type == "net" {
		proxyGroups = GetProxyListFromNet(proxyFile, key, userName, password)
	} else if proxyFile.Type == "file" {
		proxyGroups = GetProxyListFromFile(proxyFile)
	} else {
		return 0, nil, ""
	}

	if proxyGroups == nil {
		return 0, nil, ""
	}

	return proxyGroups.ClientTotalSelect, SelectProxyFromGroupsInfo(proxyGroups, pingTarget), proxyGroups.Version
}

func GetProxyListFromNet(proxyFile *ProxyFile, key, userName, password string) *ProxyGroups {
	log.Printf("**Request proxy list from %s ...", proxyFile.Url)

	req, err := retryablehttp.NewRequest("GET", proxyFile.Url, nil)
	if err != nil {
		log.Printf("GetProxyListFromNet create request %s %+v\n", proxyFile.Url, err)
		return nil
	}
	req.SetBasicAuth(userName, password)

	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = time.Second * 15
	client.RetryMax = 3

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

	if proxyGroups.ClientTotalSelect == 0 {
		proxyGroups.ClientTotalSelect = 2
	}

	return proxyGroups
}

func SelectProxyFromGroupsInfo(proxyGroups *ProxyGroups, pingTarget *PingTarget) []*ProxyInfo {
	result := make([]*ProxyInfo, 0)

	selectNum := proxyGroups.SelectPerGroup
	for i, group := range proxyGroups.Groups {
		log.Printf("**Ping group: %s, %d/%d\n", group.Name, i+1, len(proxyGroups.Groups))
		some := PingSelectProxyFromList(group.List, selectNum, pingTarget, false)
		result = append(result, some...)
		selectNum = proxyGroups.SelectPerGroup + (selectNum - len(some))
	}

	return result
}

func PingSelectProxyFromList(input []*ProxyInfo, num int, pingTarget *PingTarget, printDetails bool) []*ProxyInfo {
	workChan := make(chan *ProxyInfo, len(input))
	pingResultChan := make(chan *ProxyInfo, len(input))
	for _, proxy := range input {
		workChan <- proxy
	}
	close(workChan)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(input))

	if pingTarget.PingThreads == 0 {
		pingTarget.PingThreads = 1
	}
	for i := 0; i < pingTarget.PingThreads; i++ {
		go func() {
			for work := range workChan {
				client := NewProxyClient(work, pingTarget.Target)
				if client.Ping(client, true, nil) {
					pingResultChan <- work
				}
				waitGroup.Done()
			}
		}()
	}

	pingResult := make([]*ProxyInfo, 0, len(input))

	go func() {
		waitGroup.Wait()
		close(pingResultChan)
	}()

	n, total := 1, len(input)
	for p := range pingResultChan {
		pingResult = append(pingResult, p)
		log.Printf("%d/%d %dms %s\n", n, total, p.Ping, p.Name)
		n++
	}
	sort.SliceStable(pingResult, func(i, j int) bool {
		return pingResult[i].Ping < pingResult[j].Ping
	})

	result := make([]*ProxyInfo, 0, len(pingResult))
	for i, v := range pingResult {
		result = append(result, v)

		log.Printf("*Ping proxy delay: %s %d\n", v.Address(), v.Ping)

		if i >= num {
			break
		}
	}

	return result
}
