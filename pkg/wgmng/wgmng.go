package wgmng

import (
	dbmodule "bot/pkg/db"
	"database/sql"
	"os/exec"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Db *sql.DB
}

func (d DB) WgStartSession(user *dbmodule.User) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Session = $1,SessionTimeStamp = $2 WHERE id = $3",
		1,
		time.Stamp,
		user.ID)
	if err != nil {
		return err
	}
	wgcom := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.Peer,
		"allowed-ips", user.Allowedips,
		"ip", "-4", "route", "add",
		"192.168.88."+strconv.Itoa(user.IP)+"/32",
		"dev",
		"wg0-server")
	err = wgcom.Run()
	if err != nil {
		return err
	}
	print(time.Stamp)
	return nil
}
