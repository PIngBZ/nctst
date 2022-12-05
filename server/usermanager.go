package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
)

var (
	UserMgr = &UserManager{}
)

type UserStatus int

const (
	UserStatus_Active UserStatus = iota
	UserStatus_Blocked
)

type TrafficCountInfo struct {
	Send    uint64
	Receive uint64
}

func (h *TrafficCountInfo) FormatSend() string {
	return humanize.Bytes(h.Send)
}

func (h *TrafficCountInfo) FormatSReceive() string {
	return humanize.Bytes(h.Receive)
}

type UserInfo struct {
	ID          string
	UserName    string
	RealName    string
	Hash        string
	Admin       bool
	Session     string
	LastTime    time.Time
	CreateTime  time.Time
	Status      UserStatus
	CodeInfo    *CodeInfo
	Proxy       bool
	NoCodeLogin bool

	TrafficHour  TrafficCountInfo
	TrafficDay   TrafficCountInfo
	TrafficWeek  TrafficCountInfo
	TrafficMonth TrafficCountInfo
}

type CodeInfo struct {
	Code    int
	Time    time.Time
	Seconds int
}

type UserManager struct {
	authCodes    sync.Map
	initCode     atomic.Uint32
	initCodeTime atomic.Value
}

func (h *UserManager) CheckUserPassword(username, hash string) bool {
	var count int
	err := DB.QueryRow("select count(*) from userinfo where username=? and password=?", username, hash).Scan(&count)
	if err != nil {
		log.Printf("db query user error %s %+v\n", username, err)
		return false
	}
	return count != 0
}

func (h *UserManager) CheckAuthCode(username string, code int) bool {
	if config.Test {
		return true
	}

	if user, err := h.GetUser(username); err != nil {
		return false
	} else if user.NoCodeLogin {
		return true
	} else if c, ok := h.authCodes.Load(username); ok {
		info := c.(*CodeInfo)
		if info.Time.Add(time.Second * 65).Before(time.Now()) {
			h.authCodes.Delete(username)
			return false
		}
		return info.Code == code
	}
	return false
}

func (h *UserManager) SaveCount(user *UserInfo, send, receive int64) {
	_, err := DB.Exec("insert into datacount(username,send,receive) values(?,?,?)", user.UserName, send, receive)
	if err != nil {
		log.Printf("SaveCount error: %+v\n", err)
	} else {
		log.Printf("SaveCount %s s: %d r: %d", user.UserName, send, receive)
	}
}
