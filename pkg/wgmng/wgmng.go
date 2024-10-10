package wgmng

import (
	"bot/pkg/dbmng"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v3"
)

type HighWay struct {
	Db             *sql.DB
	Tg             *tele.Bot
	SessionManager map[int64]bool
	AdminChat      int64
}

func (d HighWay) WgStartSession(user *dbmng.User) error {
	var (
		preK = "/opt/wg/prekeys/" + strconv.FormatInt(user.ID, 10)
	)
	_, err := d.Db.Exec(
		"UPDATE users SET Session = $1,SessionTimeStamp = $2 WHERE id = $3",
		1,
		time.Now(),
		user.ID)
	if err != nil {
		return fmt.Errorf("db: failed to set start session: %w", err)
	}
	if _, err := os.Stat(preK); os.IsNotExist(err) {
		err = os.WriteFile(preK, []byte(user.PeerPre), 0644)
		if err != nil {
			return err
		}
	}
	wgcom := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.PeerPub,
		"preshared-key", preK,
		"allowed-ips", "192.168.88."+strconv.Itoa(user.IP)+"/32")
	err = wgcom.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to start session: %w", err)
	}
	d.SessionManager[user.ID] = true
	go d.Session(user, time.Now())
	return nil
}

func (d HighWay) Session(user *dbmng.User, t time.Time) {
	for d.SessionManager[user.ID] {
		if time.Now().Compare(t.Add(time.Hour*11)) == +1 {
			err := d.WgStopSession(user)
			if err != nil {
				d.Tg.Send(tele.ChatID(d.AdminChat), err)
			}
			d.Tg.Send(tele.ChatID(user.ID), "Сессия завершена")
			d.SessionManager[user.ID] = false
		}
		time.Sleep(30 * time.Second)
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
