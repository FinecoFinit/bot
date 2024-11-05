package tg

import (
	"bot/pkg/concierge"
	"bot/pkg/db"
	"bot/pkg/email"
	"bot/pkg/wg"

	tele "gopkg.in/telebot.v4"
)

type HighWay struct {
	DataBase     db.DataBase
	Tg           *tele.Bot
	Resources    concierge.Resources
	AllowedIPs   string
	DataVars     concierge.DataVars
	EmailManager email.HighWay
	WGManager    wg.HighWay
}
