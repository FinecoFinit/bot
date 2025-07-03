package service

import (
	"bot/concierge"
	"bot/service/email"
	"bot/service/tg"
	"bot/service/wg"
	"bot/storage"
	"database/sql"
	"time"

	tele "gopkg.in/telebot.v3"
)

type Unity struct {
	Config    concierge.Config
	Storage   *storage.MySql
	Telegram  *tg.Telegram
	WireGuard *wg.WireGuard
	Managers  *concierge.Managers
}

func Initialize(c concierge.Config) (Unity, error) {
	var (
		aDBids         []int64
		uDBids         []int64
		qDBids         []int64
		sessionManager = make(map[int64]bool)
		timedManager   = make(map[int64]bool)
		messageManager = make(map[int64]*tele.Message)
		tgBot          *tele.Bot
	)

	l, err := InitLogger(c.LogFilePath)
	if err != nil {
		panic(err)
	}

	database, err := sql.Open("sqlite3", c.DbPath)
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to open db")
	}

	tgBot, err = tele.NewBot(tele.Settings{Token: c.TgToken, Poller: &tele.LongPoller{Timeout: 10 * time.Second}})
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to init telegram")
	}

	managers := concierge.Managers{
		AdminDBIDs:     &aDBids,
		UserDBIDs:      &uDBids,
		QUserDBIDs:     &qDBids,
		SessionManager: sessionManager,
		MessageManager: messageManager,
		TimedManager:   timedManager,
	}

	emailClient, err := InitEmail(c)
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to init email client")
	}

	wire := wg.WireGuard{Config: c}

	stor := storage.MySql{
		MySql:     database,
		Wireguard: &wire,
		Config:    c,
	}

	em := email.Email{
		Config:      c,
		EmailClient: emailClient,
	}

	teleg := tg.Telegram{
		Storage:   &stor,
		Tg:        tgBot,
		Managers:  &managers,
		Config:    c,
		Wireguard: &wire,
		Logger:    &l,
		Email:     &em,
	}

	app := Unity{
		Config:    c,
		Storage:   &stor,
		Telegram:  &teleg,
		WireGuard: &wire,
		Managers:  &managers,
	}
	err = app.Storage.GetAdminsIDs(app.Managers.AdminDBIDs)
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to get admins")
	}
	err = app.Storage.GetUsersIDs(app.Managers.UserDBIDs)
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to get users")
	}
	err = app.Storage.GetQueueUsersIDs(app.Managers.QUserDBIDs)
	if err != nil {
		l.Panic().Err(err).Msg("db: failed to get queue")
	}
	app.Telegram.InitTelegram()

	return app, nil
}
