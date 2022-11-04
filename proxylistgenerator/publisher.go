package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
)

var (
	configFile string
	config     *Config
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
	fileInfo := &proxyclient.ProxyFile{Type: "file", Url: config.SrcFile}
	server := &nctst.AddrInfo{Host: config.ServerHost, Port: config.ServerPort}
	pingTarget := &proxyclient.PingTarget{Target: server, PingThreads: 1}

	proxyInfo := proxyclient.GetProxyListFromFile(fileInfo)

	for _, group := range proxyInfo.Groups {
		proxylist := proxyclient.SelectProxyFromGroup(group, 15, pingTarget, true)
		group.List = proxylist
	}

	buf, err := json.Marshal(proxyInfo)
	if err != nil {
		log.Printf("ToJson %+v\n", err)
		return
	}

	log.Println(string(buf))

	nctst.Xor(buf, []byte(config.Key))
	nctst.Xor(buf, []byte(config.UserName))

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/updateProxylist", config.ServerHost, config.ServerPort), nil)
	if err != nil {
		log.Printf("NewRequest %+v\n", err)
		return
	}

	req.SetBasicAuth(config.UserName, nctst.HashPassword(config.UserName, config.Password))
	response, err := client.Do(req)
	if err != nil {
		log.Printf("http request %+v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Printf("Error, StatusCode = %d", response.StatusCode)
		return
	}

	ret, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("ReadAll %+v\n", err)
		return
	}

	log.Println(string(ret))
}
