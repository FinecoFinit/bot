package wg

import (
	"bot/pkg/db"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v4"
)

func (h HighWay) WgStartSession(user *db.User) error {
	preK := path.Join(h.WgPreKeysDir, strconv.FormatInt(user.ID, 10))
	_, err := h.DataBase.DataBase.Exec("UPDATE users SET Session = $1,SessionTimeStamp = $2 WHERE id = $3", 1, time.Now(), user.ID)
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
		"allowed-ips", h.DataVars.WgSubNet+strconv.Itoa(user.IP)+"/32")
	err = wgCommand.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to start session: %w", err)
	}

	h.Resources.MessageManager[user.ID], err = h.Tg.Send(tele.ChatID(h.DataVars.AdminLogChat), "Создана сессия для: "+user.UserName, &tele.SendOptions{
		ThreadID: h.DataVars.AdminLogChatThread,
		ReplyMarkup: &tele.ReplyMarkup{
			OneTimeKeyboard: true,
			InlineKeyboard: [][]tele.InlineButton{{
				tele.InlineButton{
					Unique: "stop_session",
					Text:   "Stop",
					Data:   strconv.FormatInt(user.ID, 10)}}}}})
	if err != nil {
		return fmt.Errorf("tgmng: failed to send status message: %w", err)
	}

	h.Resources.SessionManager[user.ID] = true
	go h.Session(user, time.Now(), h.Resources.MessageManager[user.ID])

	err = os.Remove(preK)
	if err != nil {
		return fmt.Errorf("wgmng: failed to delete pre-shared key from directory: %w", err)
	}

	return nil
}

func (h HighWay) Session(user *db.User, t time.Time, statusMsg *tele.Message) {
	for h.Resources.SessionManager[user.ID] {
		wgCommand := exec.Command("wg", "show", "wg0-server", "dump")
		out, err := wgCommand.Output()
		if err != nil {
			h.Resources.Logger.Err(err).Msg("wg: failed to run command for session")
		}

		if slices.IndexFunc(strings.Split(string(out), "\r\n"), func(c string) bool { return strings.Contains(c, user.PeerPub) }) == -1 {
			err := h.WgStopSession(user, statusMsg)
			if err != nil {
				h.Resources.Logger.Err(err).Msg("wg: failed to find wg peer")
			}
			return
		}

		outStr := strings.Fields(strings.Split(string(out), "\r\n")[slices.IndexFunc(strings.Split(string(out), "\r\n"), func(c string) bool { return strings.Contains(c, user.PeerPub) })])
		statusMsgText := "Создана сессия: \n" + " 👔: " + strings.ReplaceAll(user.UserName, ".", "\\.") + "\n" + " 🌍: ``" + strings.ReplaceAll(outStr[3], ".", "\\.") + "``\n" + " ⏬: " + outStr[5] + "\n" + " ⏫: " + outStr[6] + "\n"

		if statusMsg.Text != statusMsgText {
			_, err = h.Tg.Edit(statusMsg, strings.ReplaceAll(statusMsgText, "+", "\\+"), &tele.SendOptions{
				ParseMode: "MarkdownV2",
				ReplyMarkup: &tele.ReplyMarkup{
					OneTimeKeyboard: true,
					InlineKeyboard: [][]tele.InlineButton{{
						tele.InlineButton{
							Unique: "stop_session",
							Text:   "Stop",
							Data:   strconv.FormatInt(user.ID, 10)}}}}})
			if err != nil {
				h.Resources.Logger.Err(err).Msg("session: wg: tg: failed to edit status message")
			}
			statusMsg.Text = statusMsgText
		}

		if time.Now().Compare(t.Add(time.Hour*11)) == +1 {
			err := h.WgStopSession(user, statusMsg)
			if err != nil {
				h.Resources.Logger.Err(err).Msg("session: wg: failed to stop session")
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func (h HighWay) WgStopSession(user *db.User, statusMsg *tele.Message) error {
	_, err := h.DataBase.DataBase.Exec("UPDATE users SET Session = $1 WHERE id = $2", 0, user.ID)
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

	_, err = h.Tg.Edit(statusMsg, statusMsg.Text+"Сессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
	if err != nil {
		_, err = h.Tg.Send(tele.ChatID(h.DataVars.AdminLogChat), err.Error(), &tele.SendOptions{ReplyTo: statusMsg, ThreadID: statusMsg.ThreadID})
		if err != nil {
			return fmt.Errorf("tg: failed to edit message %d: %v \n", user.ID, err)
		}
		return fmt.Errorf("wg: tg: failed to edit message: %w", err)
	}
	_, err = h.Tg.Send(tele.ChatID(user.ID), "Сессия завершена")
	if err != nil {
		return fmt.Errorf("tg: failed to send session end message: %w", err)
	}

	h.Resources.SessionManager[user.ID] = false
	h.Resources.MessageManager[user.ID] = nil

	return nil
}
