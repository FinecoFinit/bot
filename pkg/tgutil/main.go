package tgutil

import (
	"bot/pkg/emailmng"
	"bot/pkg/wgmng"
	"bot/pkg/worker"
	tele "gopkg.in/telebot.v4"
)

type HighWay struct {
	DbSet        worker.DbSet
	Tg           *tele.Bot
	Resources    worker.Resources
	AllowedIPs   string
	DataVars     worker.DataVars
	EmailManager emailmng.HighWay
	WGManager    wgmng.HighWay
}
