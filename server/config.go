package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Listen        string `json:"listen"`
	Key           string `json:"key"`
	Localnetmask  string `json:"localnetmask"`
	AdminListen   string `json:"adminlisten"`
	AdminPassword string `json:"adminpwd"`
	Test          bool   `json:"test"`

	PingUrl string
}

func parseConfig(configFile string) (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}

	if err := json.NewDecoder(file).Decode(cfg); err != nil {
		return nil, err
	}

	var pingUrl string
	if cfg.AdminListen[0] == ':' {
		pingUrl = "http://127.0.0.1" + cfg.AdminListen
	} else {
		pingUrl = "http://" + cfg.AdminListen
	}
	cfg.PingUrl = pingUrl + "/ping"

	return cfg, nil
}

func GetConfigFromDB(key string) (value string, err error) {
	if err = DB.QueryRow("select value from config where key=?", key).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return
		}
		log.Printf("configFromDB key=%s, %+v\n", key, err)
	}
	return
}

func GetConfigIntFromDB(key string) (value int, err error) {
	s, err := GetConfigFromDB(key)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("GetConfigIntFromDB key=%s, value=%s, %+v\n", key, s, err)
		return 0, err
	}
	return n, nil
}

func GetConfigBoolFromDB(key string) (value bool, err error) {
	s, err := GetConfigFromDB(key)
	if err != nil {
		return false, err
	}
	return s == "true", nil
}

func SetDBConfig(key, value string) (err error) {
	var num int
	if DB.QueryRow("select count(*) from config where key=?", key).Scan(&num); num == 0 {
		if _, err := DB.Exec("insert into config(key,value) values(?,?)", key, value); err != nil {
			log.Printf("SetDBConfig insert key=%s, %+v\n", key, err)
		}
	} else {
		if _, err = DB.Exec("update config set value=? where key=?", value, key); err != nil {
			log.Printf("SetDBConfig update key=%s, %+v\n", key, err)
		}
	}
	return
}

func SetDBConfigInt(key string, value int) (err error) {
	return SetDBConfig(key, strconv.Itoa(value))
}

func SetDBConfigBool(key string, value bool) (err error) {
	v := "false"
	if value {
		v = "true"
	}
	return SetDBConfig(key, v)
}
