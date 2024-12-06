package tg

import (
	"bot/concierge"
	"bot/service/email"
	"bot/service/wg"
	"bot/storage"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v4"
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
	t.Tg.Handle("/enable", t.Enable)
	t.Tg.Handle("/disable", t.Disable)
	t.Tg.Handle("/get", t.Get)
	t.Tg.Handle("/edit", t.Edit)
	t.Tg.Handle(tele.OnText, t.Verification)
}
