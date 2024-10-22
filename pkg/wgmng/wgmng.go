package wgmng

import (
	"bot/pkg/dbmng"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v3"
)

type HighWay struct {
	Db                 *sql.DB
	Tg                 *tele.Bot
	SessionManager     map[int64]bool
	MessageManager     map[int64]*tele.Message
	AdminLogChat       int64
	AdminLogChatThread int
	WgPreKeysDir       string
}

func (h HighWay) WgStartSession(user *dbmng.User) error {
	preK := path.Join(h.WgPreKeysDir, strconv.FormatInt(user.ID, 10))
	_, err := h.Db.Exec("UPDATE users SET Session = $1,SessionTimeStamp = $2 WHERE id = $3", 1, time.Now(), user.ID)
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

	h.MessageManager[user.ID], err = h.Tg.Send(tele.ChatID(h.AdminLogChat), "Создана сессия для: "+user.UserName, &tele.SendOptions{ThreadID: h.AdminLogChatThread})
	if err != nil {
		return fmt.Errorf("tgmng: failed to send status message: %w", err)
	}

	h.SessionManager[user.ID] = true
	go h.Session(user, time.Now(), h.MessageManager[user.ID])

	return nil
}

func (h HighWay) Session(user *dbmng.User, t time.Time, statusMsg *tele.Message) {
	for h.SessionManager[user.ID] {
		wgCommand := exec.Command("wg", "show", "wg0-server", "dump")
		out, err := wgCommand.Output()
		if err != nil {
			fmt.Printf("wgmng: failed to run command for session: %d", err)
		}

		outStr := strings.Fields(strings.Split(string(out), "\r\n")[slices.IndexFunc(strings.Split(string(out), "\r\n"), func(c string) bool { return strings.Contains(c, user.PeerPub) })])
		statusMsgText := "Создана сессия: \n" + " 👔: " + strings.ReplaceAll(user.UserName, ".", "\\.") + "\n" + " 🌍: ``" + strings.ReplaceAll(outStr[3], ".", "\\.") + "``\n" + " ⏬: " + outStr[5] + "\n" + " ⏫: " + outStr[6] + "\n"

		if statusMsg.Text != statusMsgText {
			_, err = h.Tg.Edit(statusMsg, statusMsgText, &tele.SendOptions{ParseMode: "MarkdownV2"})
			if err != nil {
				fmt.Printf("tg: failed to edit message: %d, %v \n", statusMsg.ID, err)
			}
			statusMsg.Text = statusMsgText
		}

		if time.Now().Compare(t.Add(time.Hour*11)) == +1 {
			err := h.WgStopSession(user, statusMsg)
			if err != nil {
				fmt.Printf("tg: failed to stop session: %d, %v \n", statusMsg.ID, err)
			}

			_, err = h.Tg.Send(tele.ChatID(user.ID), "Сессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
			if err != nil {
				fmt.Printf("tg: failed to send message %d: %v \n", user.ID, err)
			}

			h.SessionManager[user.ID] = false
		}
		time.Sleep(30 * time.Second)
	}
}

func (h HighWay) WgStopSession(user *dbmng.User, statusMsg *tele.Message) error {
	_, err := h.Db.Exec("UPDATE users SET Session = $1 WHERE id = $2", 0, user.ID)
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

	h.MessageManager[user.ID] = nil

	_, err = h.Tg.Edit(statusMsg, statusMsg.Text+"\nСессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
	if err != nil {
		_, err = h.Tg.Send(tele.ChatID(h.AdminLogChat), err.Error(), &tele.SendOptions{ReplyTo: statusMsg, ThreadID: h.AdminLogChatThread})
		if err != nil {
			return fmt.Errorf("tg: failed to edit message %d: %v \n", user.ID, err)
		}
	}
	return nil
}
