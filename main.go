package main

import (
	"bot/pkg/dbmng"
	"bot/pkg/emailmng"
	"bot/pkg/tgutil"
	"bot/pkg/wgmng"
	"bot/pkg/worker"
	"database/sql"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wneessen/go-mail"

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
	tg, err := tele.NewBot(pref)
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

	dataVars := worker.DataVars{
		AdminLogChat:       adminLogChat,
		AdminLogChatThread: adminLogChatThread}

	db := worker.DbSet{
		DbVar:  database,
		DbUtil: dbmng.DB{Db: database}}
	res := worker.Resources{
		AdminDBIDs:     &aDBids,
		UserDBIDs:      &uDBids,
		QUserDBIDs:     &qDBids,
		SessionManager: sessionManager,
		MessageManager: messageManager,
		Logger:         logger,
	}

	wg := wgmng.HighWay{
		DbSet:              db,
		Tg:                 tg,
		Resources:          res,
		AdminLogChat:       adminLogChat,
		AdminLogChatThread: adminLogChatThread,
		WgPreKeysDir:       wgPreKeysDir}
	em := emailmng.HighWay{
		WgServerIP:  &wgSerIP,
		WgPublicKey: &wgPubKey,
		EmailClient: emailClient,
		EmailUser:   &emailUser,
		EmailPass:   &emailPass,
		EmailAddr:   &emailAddr}
	HWtg := tgutil.HighWay{
		DbSet:        db,
		Tg:           tg,
		Resources:    res,
		AllowedIPs:   wgAllowedIPs,
		DataVars:     dataVars,
		WGManager:    wg,
		EmailManager: em}

	err = db.DbUtil.GetAdminsIDs(&aDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get admin ids")
	}
	err = db.DbUtil.GetUsersIDs(&uDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get user ids")
	}
	err = db.DbUtil.GetQueueUsersIDs(&qDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get queue ids")
	}

	tg.Handle(&tele.Btn{Unique: "register_accept"}, HWtg.RegisterAccept)

	tg.Handle(&tele.Btn{Unique: "register_deny"}, HWtg.RegisterDeny)

	tg.Handle(&tele.Btn{Unique: "stop_session"}, HWtg.StopSession)

	tg.Handle("/register", HWtg.Register)

	tg.Handle("/accept", HWtg.Accept)

	tg.Handle("/adduser", HWtg.AddUser)

	tg.Handle("/deluser", HWtg.DelUser)

	tg.Handle("/sendcreds", HWtg.SendCreds)

	tg.Handle("/enable", HWtg.Enable)

	tg.Handle("/disable", HWtg.Disable)

	tg.Handle("/get", HWtg.Get)

	tg.Handle(tele.OnText, HWtg.Verification)

	tg.Start()
}
