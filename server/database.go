package main

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/PIngBZ/nctst"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB               *sql.DB
	CurrentDBVersion = 101
)

func init() {
	db, err := sql.Open("sqlite3", "./data.db")
	nctst.CheckError(err)
	DB = db

	createConfigTable(db)
	createUserTable(db)
	createDataCountTable(db)

	upgradeDatabase()
}

func createAminUser() {
	cmd := "insert into userinfo(username,realname,password,admin,proxy) values(?,?,?,?,?)"
	DB.Exec(cmd, "admin", "Administrator", nctst.HashPassword("admin", config.AdminPassword), 1, 1)

	cmd = "upadte userinfo set password=? where username=admin"
	DB.Exec(cmd, nctst.HashPassword("admin", config.AdminPassword))
}

func createConfigTable(db *sql.DB) {
	cmd := `
		CREATE TABLE IF NOT EXISTS config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key VARCHAR(64) UNIQUE,
			value VARCHAR(64)
		); 
	`
	_, err := db.Exec(cmd)
	nctst.CheckError(err)
}

func createUserTable(db *sql.DB) {
	cmd := `
		CREATE TABLE IF NOT EXISTS userinfo (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(64) UNIQUE,
			password VARCHAR(64),
			realname VARCHAR(64),
			admin INTEGER DEFAULT 0,
			session VARCHAR(64) DEFAULT "",
			lasttime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			createtime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status INTEGER DEFAULT 0
		); 
	`
	_, err := db.Exec(cmd)
	nctst.CheckError(err)
}

func createDataCountTable(db *sql.DB) {
	cmd := `
		CREATE TABLE IF NOT EXISTS datacount (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(64),
			send INTEGER DEFAULT 0,
			receive INTEGER DEFAULT 0,
			savetime TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		); 
	`
	_, err := db.Exec(cmd)
	nctst.CheckError(err)
}

func upgradeDatabase() {
	ver, _ := GetConfigIntFromDB("dbversion")

	if ver <= CurrentDBVersion {
		switch {
		case ver < 100:
			upgrade100()
			fallthrough
		case ver < 101:
			upgrade101()
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

func upgrade101() {
	_, err := DB.Exec("alter table userinfo add column nocodelogin INTEGER DEFAULT 0")
	nctst.CheckError(err)
}
