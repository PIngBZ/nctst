package proxyclient

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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

func GetProxyList(proxyFile *ProxyFile, pingTarget *PingTarget) []*ProxyInfo {
	var proxyGroups *ProxyGroups
	if proxyFile.Type == "net" {
		proxyGroups = GetProxyListFromNet(proxyFile)
	} else if proxyFile.Type == "file" {
		proxyGroups = GetProxyListFromFile(proxyFile)
	} else {
		return nil
	}

	return SelectProxyFromGroupsInfo(proxyGroups, pingTarget)
}

func GetProxyListFromNet(proxyFile *ProxyFile) *ProxyGroups {
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
	for _, group := range proxyGroups.Groups {
		some := SelectProxyFromGroup(group, selectNum, pingTarget, false)
		result = append(result, some...)
		selectNum = proxyGroups.SelectPerGroup + (selectNum - len(some))
	}

	return result
}

func SelectProxyFromGroup(group *ProxyGroup, num int, pingTarget *PingTarget, printDetails bool) []*ProxyInfo {
	printf := func(format string, a ...any) {
		if printDetails {
			fmt.Printf(format, a...)
		}
	}

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
			work, ok := <-workChan
			if ok {
				client := NewProxyClient(work, pingTarget.Target)
				printf("NewProxyClient %s", pingTarget.Target.Address())
				if client.Ping(client, printDetails, nil) {
					pingResultChan <- nctst.Pair[uint32, *ProxyInfo]{First: client.LastPing(), Second: work}
				}
			}
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()

	close(pingResultChan)

	pingResult := make([]nctst.Pair[uint32, *ProxyInfo], 0)

	for p := range pingResultChan {
		pingResult = append(pingResult, p)
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
