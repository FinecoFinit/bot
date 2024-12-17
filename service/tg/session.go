package tg

import (
	"bot/concierge"
	tele "gopkg.in/telebot.v3"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (t Telegram) Session(user *concierge.User, ti time.Time, statusMsg *tele.Message) {
	t.Managers.SessionManager[user.ID] = true
	for t.Managers.SessionManager[user.ID] {
		wgCommand := exec.Command("wg", "show", "wg0-server", "dump")
		out, err := wgCommand.Output()
		if err != nil {
			t.Logger.Err(err).Msg("wg: failed to run command for session")
		}

		if slices.IndexFunc(strings.Split(string(out), "\n"), func(c string) bool { return strings.Contains(c, user.PeerPub) }) == -1 {
			err := t.SessionEnded(*user)
			if err != nil {
				t.Logger.Err(err).Msg("wg: failed to find wg peer")
			}
			t.Managers.SessionManager[user.ID] = false
			return
		}

		outStr := strings.Fields(strings.Split(string(out), "\n")[slices.IndexFunc(strings.Split(string(out), "\n"), func(c string) bool { return strings.Contains(c, user.PeerPub) })])
		statusMsgText := "Создана сессия: \n" + " 👔: " + strings.ReplaceAll(user.UserName, ".", "\\.") + "\n" + " 🌍: ``" + strings.ReplaceAll(outStr[3], ".", "\\.") + "``\n" + " ⏬: " + outStr[5] + "\n" + " ⏫: " + outStr[6] + "\n"

		if statusMsg.Text != statusMsgText {
			_, err = t.Tg.Edit(statusMsg, strings.ReplaceAll(statusMsgText, "+", "\\+"), &tele.SendOptions{
				ParseMode: "MarkdownV2",
				ReplyMarkup: &tele.ReplyMarkup{
					OneTimeKeyboard: true,
					InlineKeyboard: [][]tele.InlineButton{{
						tele.InlineButton{
							Unique: "stop_session",
							Text:   "Stop",
							Data:   strconv.FormatInt(user.ID, 10)}}}}})
			if err != nil {
				t.Logger.Err(err).Msg("session: wg: tg: failed to edit status message")
			}
			statusMsg.Text = statusMsgText
		}

		if time.Now().Compare(ti.Add(time.Hour*11)) == +1 {
			err := t.Wireguard.WgStopSession(user)
			if err != nil {
				t.Logger.Err(err).Msg("session: wg: failed to stop session")
			}
			err = t.SessionEnded(*user)
			if err != nil {
				t.Logger.Err(err).Msg("session: wg: failed to end session")
			}
			t.Managers.SessionManager[user.ID] = false
		}
		time.Sleep(30 * time.Second)
	}
}
