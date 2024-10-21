package wgmng

import (
	"bot/pkg/dbmng"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v3"
)

type HighWay struct {
	Db                 *sql.DB
	Tg                 *tele.Bot
	SessionManager     map[int64]bool
	AdminChat          int64
	AdminLogChat       int64
	AdminLogChatThread int
	WgPreKeysDir       string
}

func (h HighWay) WgStartSession(user *dbmng.User) error {
	var (
		preK = path.Join(h.WgPreKeysDir, strconv.FormatInt(user.ID, 10))
	)

	_, err := h.Db.Exec(
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
	wgCommand := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.PeerPub,
		"preshared-key", preK,
		"allowed-ips", "192.168.88."+strconv.Itoa(user.IP)+"/32")
	err = wgCommand.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to start session: %w", err)
	}
	statusMsg, err := h.Tg.Send(tele.ChatID(h.AdminLogChat), "Создана сессия для: "+user.UserName, &tele.SendOptions{ThreadID: h.AdminLogChatThread})
	if err != nil {
		return fmt.Errorf("tgmng: failed to send status message: %w", err)
	}
	h.SessionManager[user.ID] = true
	go h.Session(user, time.Now(), statusMsg)
	return nil
}

func (h HighWay) Session(user *dbmng.User, t time.Time, statusMsg *tele.Message) {
	for h.SessionManager[user.ID] {
		_, err := h.Tg.Edit(statusMsg, "Status")
		if err != nil {
			fmt.Printf("tg: failed to edit message: %d, %v \n", statusMsg.ID, err)
		}
		if time.Now().Compare(t.Add(time.Hour*11)) == +1 {
			err := h.WgStopSession(user)
			if err != nil {
				_, err = h.Tg.Send(tele.ChatID(h.AdminChat), err.Error())
				if err != nil {
					fmt.Printf("tg: failed to stop session %d: %v \n", user.ID, err)
				}
			}
			_, err = h.Tg.Send(tele.ChatID(user.ID), "Сессия завершена")
			if err != nil {
				fmt.Printf("tg: failed to send message %d: %v \n", user.ID, err)
			}
			h.SessionManager[user.ID] = false
		}
		time.Sleep(30 * time.Second)
	}
}

func (h HighWay) WgStopSession(user *dbmng.User) error {
	_, err := h.Db.Exec(
		"UPDATE users SET Session = $1 WHERE id = $2",
		0,
		user.ID)
	if err != nil {
		return fmt.Errorf("db: failed to set stop session: %w", err)
	}
	wgCommand := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.PeerPub,
		"remove")
	err = wgCommand.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to stop session: %w", err)
	}
	return nil
}
