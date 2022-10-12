package main

import (
	"database/sql"
	"time"

	"github.com/PIngBZ/nctst"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB *sql.DB
)

func init() {
	db, err := sql.Open("sqlite3", "./data.db")
	nctst.CheckError(err)
	DB = db

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
	_, err = db.Exec(cmd)
	nctst.CheckError(err)

}

func createAminUser() {
	time.Sleep(time.Second)
	cmd := "insert into userinfo(username,realname,password,admin) values(?,?,?,?)"
	DB.Exec(cmd, "admin", "Administrator", nctst.HashPassword("admin", config.AdminPassword), 1)

	cmd = "upadte userinfo set password=? where username=admin"
	DB.Exec(cmd, nctst.HashPassword("admin", config.AdminPassword))
}
