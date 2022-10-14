package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/PIngBZ/nctst"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB               *sql.DB
	CurrentDBVersion = 100
)

func init() {
	db, err := sql.Open("sqlite3", "./data.db")
	nctst.CheckError(err)
	DB = db

	cmd := `
		CREATE TABLE IF NOT EXISTS config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key VARCHAR(64) UNIQUE,
			value VARCHAR(64),
		); 
	`
	_, err = db.Exec(cmd)
	nctst.CheckError(err)

	cmd = `
		CREATE TABLE IF NOT EXISTS userinfo (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(64) UNIQUE,
			password VARCHAR(64),
			realname VARCHAR(64),
			admin INTEGER DEFAULT 0,
			session VARCHAR(64) DEFAULT "",
			lasttime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			createtime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status INTEGER DEFAULT 0,
			proxy INTEGER DEFAULT 0
		); 
	`
	_, err = db.Exec(cmd)
	nctst.CheckError(err)

	upgradeDatabase()
}

func CreateAminUser() {
	time.Sleep(time.Second)
	cmd := "insert into userinfo(username,realname,password,admin) values(?,?,?,?)"
	DB.Exec(cmd, "admin", "Administrator", nctst.HashPassword("admin", config.AdminPassword), 1)

	cmd = "upadte userinfo set password=? where username=admin"
	DB.Exec(cmd, nctst.HashPassword("admin", config.AdminPassword))
}

func upgradeDatabase() {
	ver, _ := GetConfigIntFromDB("dbversion")

	if ver < CurrentDBVersion {
		switch {
		case ver < 100:
			upgrade100()
			fallthrough
		default:
		}

		SetDBConfig("dbversion", strconv.Itoa(CurrentDBVersion))
	} else {
		nctst.CheckError(fmt.Errorf("database version error, file: %d, current: %d", ver, CurrentDBVersion))
	}
}

func upgrade100() {
	_, err := DB.Exec("alter table userinfo add column proxy INTEGER DEFAULT 0")
	nctst.CheckError(err)
}
