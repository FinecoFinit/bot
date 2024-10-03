package wgmng

import (
	dbmng "bot/pkg/dbmng"
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v3"
)

type HighWay struct {
	Db             *sql.DB
	Tg             *tele.Bot
	Sessionmanager map[int64]bool
	Adminchat      int64
}

func (d HighWay) WgStartSession(user *dbmng.User) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Session = $1,SessionTimeStamp = $2 WHERE id = $3",
		1,
		time.Now(),
		user.ID)
	if err != nil {
		return fmt.Errorf("db: failed to set start session: %w", err)
	}
	wgcom := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.Peer,
		"allowed-ips", "192.168.88."+strconv.Itoa(user.IP)+"/32")
	err = wgcom.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to start session: %w", err)
	}
	d.Sessionmanager[user.ID] = true
	go d.Session(user, time.Now())
	return nil
}

func (d HighWay) Session(user *dbmng.User, t time.Time) {
	for d.Sessionmanager[user.ID] {
		if time.Now().Compare(t.Add(time.Hour*11)) == +1 {
			err := d.WgStopSession(user)
			if err != nil {
				d.Tg.Send(tele.ChatID(d.Adminchat), err)
			}
			d.Tg.Send(tele.ChatID(user.ID), "Сессия завершена")
			d.Sessionmanager[user.ID] = false
		}
	}
}

func (d HighWay) WgStopSession(user *dbmng.User) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Session = $1 WHERE id = $2",
		0,
		user.ID)
	if err != nil {
		return fmt.Errorf("db: failed to set stop session: %w", err)
	}
	wgcom := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.Peer,
		"remove")
	err = wgcom.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to stop session: %w", err)
	}
	return nil
}
