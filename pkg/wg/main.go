package wg

import (
	"bot/pkg/concierge"
	"bot/pkg/db"
	tele "gopkg.in/telebot.v4"
)

type HighWay struct {
	DataBase     *db.DataBase
	DataVars     *concierge.DataVars
	Tg           *tele.Bot
	Resources    *concierge.Resources
	WgPreKeysDir string
}
