package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/proxyclient"
	"github.com/hashicorp/go-retryablehttp"
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
	nctst.CommandXorKey = config.Key
}

func main() {
	fileInfo := &proxyclient.ProxyFile{Type: "file", Url: config.SrcFile}
	pingTarget := &proxyclient.PingTarget{Target: config.Target, PingThreads: config.PingThreadNum}

	proxyInfo := proxyclient.GetProxyListFromFile(fileInfo)

	for _, group := range proxyInfo.Groups {
		proxylist := proxyclient.PingSelectProxyFromList(group.List, config.SelectPerGroup, pingTarget, true)
		group.List = proxylist
	}
	proxyInfo.ClientTotalSelect = config.ClientTotalSelect

	buf, err := json.Marshal(proxyInfo)
	if err != nil {
		log.Printf("ToJson %+v\n", err)
		return
	}

	log.Println(string(buf))

	nctst.Xor(buf, []byte(config.Key))
	nctst.Xor(buf, []byte(config.UserName))

	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = time.Second * time.Duration(config.PublishTimeout)
	client.RetryMax = config.PublishRetry

	url := fmt.Sprintf("http://%s/updateProxylist", config.PublishServer.Address())
	buffer := bytes.NewBuffer([]byte{})
	nctst.WriteLData(buffer, buf)
	req, err := retryablehttp.NewRequest("POST", url, buffer)
	if err != nil {
		log.Printf("NewRequest %+v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(config.UserName, config.PassWord)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("http request %+v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Printf("Error, StatusCode = %d\n", response.StatusCode)
		return
	}

	ret, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("ReadAll %+v\n", err)
		return
	}

	apiResp := &nctst.APIResponse{}
	if err = json.Unmarshal(ret, apiResp); err != nil {
		log.Printf("Error, Response json Unmarshal %+v\n", err)
		return
	}

	if apiResp.Code != nctst.APIResponseCode_Success {
		log.Printf("Error, Response json code= %d\n", apiResp.Code)
		return
	}

	log.Println(string(ret))
}
