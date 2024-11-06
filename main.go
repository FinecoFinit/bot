package main

import (
	"bot/pkg/concierge"
	"bot/pkg/db"
	"bot/pkg/email"
	"bot/pkg/tg"
	"bot/pkg/wg"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/wneessen/go-mail"

	_ "github.com/mattn/go-sqlite3"
	tele "gopkg.in/telebot.v4"
)

func main() {
	var (
		aDBids               []int64
		uDBids               []int64
		qDBids               []int64
		adminLogChat         int64
		sessionManager       = make(map[int64]bool)
		messageManager       = make(map[int64]*tele.Message)
		wgSerIP              = os.Getenv("WG_SER_IP")
		wgPubKey             = os.Getenv("WG_SER_PUBK")
		wgPreKeysDir         = os.Getenv("WG_PREKEYS_DIR")
		wgAllowedIPs         = os.Getenv("WG_ALLOWED_IPS")
		token                = os.Getenv("TOKEN")
		adminLogChatID       = os.Getenv("ADMIN_LOG_CHAT")
		adminLogChatThreadID = os.Getenv("ADMIN_LOG_CHAT_THREAD")
		dbPath               = os.Getenv("DB")
		emailUser            = os.Getenv("EMAIL_LOGIN")
		emailPass            = os.Getenv("EMAIL_PASS")
		emailAddr            = os.Getenv("EMAIL_ADDR")
		logFilePath          = os.Getenv("LOG_FILE")
	)

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	multiLog := zerolog.MultiLevelWriter(os.Stdout, logFile)
	if err != nil {
		panic(err)
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			panic(err)
		}
	}(logFile)
	logger := zerolog.New(multiLog).With().Timestamp().Logger()

	pref := tele.Settings{Token: token, Poller: &tele.LongPoller{Timeout: 10 * time.Second}}
	tgBot, err := tele.NewBot(pref)
	if err != nil {
		logger.Panic().Err(err).Msg("ENV: TOKEN parse error")
	}

	adminLogChat, err = strconv.ParseInt(adminLogChatID, 10, 64)
	if err != nil {
		logger.Panic().Err(err).Msg("ENV: ADMIN_LOG_CHAT parse error")
	}

	adminLogChatThread, err := strconv.Atoi(adminLogChatThreadID)
	if err != nil {
		logger.Panic().Err(err).Msg("ENV: ADMIN_LOG_CHAT_THREAD parse error")
	}

	location, err := filepath.Abs(dbPath)
	if err != nil {
		logger.Panic().Err(err).Msg("db file: file doesn't exists or corrupted")
	}

	database, err := sql.Open("sqlite3", location)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to open db")
	}

	emailClient, err := mail.NewClient(
		emailAddr,
		mail.WithPort(587),
		mail.WithSSL(),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(emailUser),
		mail.WithPassword(emailPass))
	if err != nil {
		logger.Panic().Err(err).Msg("failed to create mail client")
	}

	dataVars := concierge.DataVars{
		AdminLogChat:       adminLogChat,
		AdminLogChatThread: adminLogChatThread}

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
		WgPreKeysDir: wgPreKeysDir}
	em := email.HighWay{
		WgServerIP:  &wgSerIP,
		WgPublicKey: &wgPubKey,
		EmailClient: emailClient,
		EmailUser:   &emailUser,
		EmailPass:   &emailPass,
		EmailAddr:   &emailAddr}
	HWtg := tg.HighWay{
		DataBase:     &dbSet,
		Tg:           tgBot,
		Resources:    &res,
		AllowedIPs:   wgAllowedIPs,
		DataVars:     &dataVars,
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
