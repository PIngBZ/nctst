package main

import (
	"errors"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/client/core"
)

var (
	authCode   int
	configFile string
	config     *core.Config
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
	config, err = core.ParseConfig(configFile)
	nctst.CheckError(err)

	nctst.CommandXorKey = config.Key
}

func main() {
	nctst.CheckError(core.Start(config, authCode))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
