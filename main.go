package main

import (
	"bot/pkg/concierge"
	"bot/pkg/db"
	"bot/pkg/email"
	"bot/pkg/tg"
	"bot/pkg/wg"
	"database/sql"
	"gopkg.in/yaml.v3"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/wneessen/go-mail"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v4"
)

func main() {
	var (
		aDBids         []int64
		uDBids         []int64
		qDBids         []int64
		sessionManager = make(map[int64]bool)
		messageManager = make(map[int64]*tele.Message)
		configPath     = os.Getenv("CONFIG_PATH")
		config         concierge.Config
	)

	yamlData, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(config.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		panic(err)
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			panic(err)
		}
	}(logFile)
	logger := zerolog.New(zerolog.MultiLevelWriter(os.Stdout, logFile)).With().Timestamp().Logger()

	tgBot, err := tele.NewBot(tele.Settings{Token: config.TgToken, Poller: &tele.LongPoller{Timeout: 10 * time.Second}})
	if err != nil {
		logger.Panic().Err(err).Msg("ENV: TOKEN parse error")
	}

	database, err := sql.Open("sqlite3", config.DbPath)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to open db")
	}

	emailClient, err := mail.NewClient(
		config.EmailAddress,
		mail.WithPort(587),
		mail.WithSSL(),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(config.EmailUser),
		mail.WithPassword(config.EmailPassword))
	if err != nil {
		logger.Panic().Err(err).Msg("failed to create mail client")
	}

	dataVars := concierge.DataVars{
		AdminLogChat:       config.AdminWgChatID,
		AdminLogChatThread: config.AdminWgChatThread,
		WgDNS:              config.WgDNS,
		WgSubNet:           config.WgSubNet,
	}

	dbSet := db.DataBase{DataBase: database}
	res := concierge.Resources{
		AdminDBIDs:     &aDBids,
		UserDBIDs:      &uDBids,
		QUserDBIDs:     &qDBids,
		SessionManager: sessionManager,
		MessageManager: messageManager,
		Logger:         logger,
	}

	wireguard := wg.HighWay{
		DataBase:     &dbSet,
		DataVars:     &dataVars,
		Tg:           tgBot,
		Resources:    &res,
		WgPreKeysDir: config.WgPreKeysDir}
	em := email.HighWay{
		DataVars:    dataVars,
		WgServerIP:  &config.WgPublicIP,
		WgPublicKey: &config.WgPublicKey,
		EmailClient: emailClient,
		EmailUser:   &config.EmailUser,
		EmailPass:   &config.EmailPassword,
		EmailAddr:   &config.EmailAddress}
	HWtg := tg.HighWay{
		DataBase:     &dbSet,
		DataVars:     &dataVars,
		Tg:           tgBot,
		Resources:    &res,
		AllowedIPs:   config.WgAllowedIps,
		WGManager:    &wireguard,
		EmailManager: &em}

	err = dbSet.GetAdminsIDs(&aDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get admin ids")
	}
	err = dbSet.GetUsersIDs(&uDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get user ids")
	}
	err = dbSet.GetQueueUsersIDs(&qDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get queue ids")
	}

	tgBot.Handle(&tele.Btn{Unique: "register_accept"}, HWtg.RegisterAccept)

	tgBot.Handle(&tele.Btn{Unique: "register_deny"}, HWtg.RegisterDeny)

	tgBot.Handle(&tele.Btn{Unique: "stop_session"}, HWtg.StopSession)

	tgBot.Handle("/register", HWtg.Register)

	tgBot.Handle("/accept", HWtg.Accept)

	tgBot.Handle("/adduser", HWtg.AddUser)

	tgBot.Handle("/deluser", HWtg.DelUser)

	tgBot.Handle("/sendcreds", HWtg.SendCreds)

	tgBot.Handle("/enable", HWtg.Enable)

	tgBot.Handle("/disable", HWtg.Disable)

	tgBot.Handle("/get", HWtg.Get)

	tgBot.Handle("/edit", HWtg.Edit)

	tgBot.Handle(tele.OnText, HWtg.Verification)

	tgBot.Start()
}
