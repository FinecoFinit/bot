package worker

import (
	"bot/pkg/dbmng"
	"database/sql"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v4"
)

type DataVars struct {
	AdminLogChat       int64
	AdminLogChatThread int
}

type DbSet struct {
	DbVar  *sql.DB
	DbUtil dbmng.DB
}

type Resources struct {
	AdminDBIDs     *[]int64
	UserDBIDs      *[]int64
	QUserDBIDs     *[]int64
	SessionManager map[int64]bool
	MessageManager map[int64]*tele.Message
	Logger         zerolog.Logger
}
