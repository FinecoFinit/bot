package tg

import (
	"bot/concierge"
	"bot/service/email"
	"bot/service/wg"
	"bot/storage"
	"encoding/json"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
	"time"
)

type Telegram struct {
	Storage   *storage.MySql
	Tg        *tele.Bot
	Managers  *concierge.Managers
	Config    concierge.Config
	Wireguard *wg.WireGuard
	Logger    *zerolog.Logger
	Email     *email.Email
}

func (t Telegram) InitTelegram() {
	t.Tg.Handle(&tele.Btn{Unique: "register_accept"}, t.RegisterAccept)
	t.Tg.Handle(&tele.Btn{Unique: "register_deny"}, t.RegisterDeny)
	t.Tg.Handle(&tele.Btn{Unique: "stop_session"}, t.StopSession)
	t.Tg.Handle(&tele.Btn{Unique: "send_creds"}, t.SendCredsBtn)
	t.Tg.Handle("/start", t.Start)
	t.Tg.Handle("/register", t.Register)
	t.Tg.Handle("/accept", t.Accept)
	t.Tg.Handle("/adduser", t.AddUser)
	t.Tg.Handle("/deluser", t.DelUser)
	t.Tg.Handle("/sendcreds", t.SendCreds)
	t.Tg.Handle("/get", t.Get)
	t.Tg.Handle("/set", t.Set)
	t.Tg.Handle("/reload", t.Reload)
	t.Tg.Handle(tele.OnText, t.Verification)
}

func (t Telegram) RestoreSessions() {
	time.Sleep(5 * time.Second)
	users, err := t.Storage.GetUsers()
	if err != nil {
		t.Logger.Error().Err(err).Msg("Failed to restore sessions")
	}
	for _, u := range users {
		if u.Session == 0 {
			continue
		}
		go func(goU concierge.User) {
			st, err := time.Parse(time.DateTime, goU.SessionTimeStamp)
			if err != nil {
				t.Logger.Error().Err(err).Msg("Failed to restore sessions: time")
			}
			if time.Now().Compare(st.Add(11*time.Hour)) == -1 {
				var msg tele.Message
				err = json.Unmarshal([]byte(goU.SessionMessageID), &msg)
				if err != nil {
					t.Logger.Error().Err(err).Msg("Failed to restore sessions: message unmarshal")
				}
				t.Managers.SessionManager[goU.ID] = true
				t.Managers.MessageManager[goU.ID] = &msg
				go t.Session(&goU, st, t.Managers.MessageManager[goU.ID])
				t.Logger.Info().Msg("Session Restored: " + goU.UserName)
			}
		}(u)
	}
}

func (t Telegram) RestoreTimedEnable() {
	time.Sleep(5 * time.Second)
	users, err := t.Storage.GetTimedEnable()
	if err != nil {
		t.Logger.Error().Err(err).Msg("Failed to restore timed enable")
	}
	for _, u := range users {
		go func(goU concierge.TimedEnable) {
			st, err := time.Parse("2006-01-02 15:04:05 Z0700", u.Date)
			if err != nil {
				t.Logger.Error().Err(err).Msg("timedenenable failed to parse time")
			}
			t.Managers.TimedManager[goU.ID] = true
			go t.Timed(u.ID, st)
		}(u)
	}
}
