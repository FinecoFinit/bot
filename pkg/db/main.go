package db

import "database/sql"

type User struct {
	ID               int64
	UserName         string
	Enabled          int
	TOTPSecret       string
	Session          int
	SessionTimeStamp string
	Peer             string
	PeerPre          string
	PeerPub          string
	AllowedIPs       string
	IP               int
}

type Admin struct {
	ID       int64
	UserName string
}

type QueueUser struct {
	ID         int64
	UserName   string
	TOTPSecret string
	Peer       string
	PeerPub    string
	PeerPre    string
	IP         int
}

type DataBase struct {
	DataBase *sql.DB
}
