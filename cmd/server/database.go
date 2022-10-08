package main

import (
	"database/sql"

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
			uid INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(64),
			password VARCHAR(64),
			name VARCHAR(64),
			lastip VARCHAR(64),
			lasttime TIMESTAMP DEFAULT 0,
			createtime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status INTEGER DEFAULT 0,
		); 
	`
	db.Exec(cmd)
}
