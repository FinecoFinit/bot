package main

import (
	"bot/pkg/dbmng"
	"bot/pkg/emailmng"
	"bot/pkg/wgmng"
	"bytes"
	"database/sql"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/thoas/go-funk"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pquerna/otp/totp"
	"github.com/wneessen/go-mail"

	tele "gopkg.in/telebot.v3"
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
	if err != nil {
		panic(err)
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			panic(err)
		}
	}(logFile)
	logger := zerolog.New(logFile).With().Timestamp().Logger()

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

	db, err := sql.Open("sqlite3", location)
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

	s := dbmng.DB{
		Db: db}
	wg := wgmng.HighWay{
		Db:                 db,
		Tg:                 tg,
		SessionManager:     sessionManager,
		MessageManager:     messageManager,
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

	err = s.GetAdminsIDs(&aDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get admin ids")
	}
	err = s.GetUsersIDs(&uDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get user ids")
	}
	err = s.GetQueueUsersIDs(&qDBids)
	if err != nil {
		logger.Panic().Err(err).Msg("db: failed to get queue ids")
	}

	tg.Handle("/register", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
		)

		if len(tga) != 1 {
			return c.Send("Ошибка введенных параметров")
		}

		if slices.Contains(uDBids, tgu.ID) {
			return c.Send("Пользователь существует")
		}
		if slices.Contains(qDBids, tgu.ID) {
			return c.Send("Регистрация в процессе")
		}
		err = s.RegisterQueue(tgu.ID, tga[0])
		if err != nil {
			logger.Error().Err(err).Msg("registration: failed to register user")
			return c.Send("Ошибка, сообщите администратору")
		}

		_, err = tg.Send(
			tele.ChatID(adminLogChat),
			"В очередь добавлен новый пользователь:\nID: ``"+strconv.FormatInt(tgu.ID, 10)+
				"``\nusername: @"+tgu.Username+
				"\nlogin: "+strings.Replace(tga[0], ".", "\\.", 1)+
				"\n`\n/accept "+strconv.FormatInt(tgu.ID, 10)+" [AllowedIP]`", &tele.SendOptions{
				ThreadID:  adminLogChatThread,
				ParseMode: "MarkdownV2"})
		if err != nil {
			logger.Error().Err(err).Msg("registration")
		}
		err = s.GetQueueUsersIDs(&qDBids)
		if err != nil {
			logger.Error().Err(err).Msg("registration: failed to update queue ids")
		}
		logger.Info().Msg("new user registered in queue: " + strconv.FormatInt(tgu.ID, 10))
		return c.Send("Заявка на регистрацию принята")
	})

	tg.Handle("/accept", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
		)

		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("accept: non admin user tried to use /accept")
			return c.Send("Unknown")
		}

		if len(tga) != 2 {
			return c.Send("Задано неверное количество параметров")
		}

		id, err := strconv.ParseInt(tga[0], 10, 64)
		if err != nil {
			return c.Send("Неудалось обработать ID пользователя")
		}

		qUser, err := s.GetQueueUser(&id)
		if err != nil {
			logger.Error().Err(err).Msg("accept")
			return c.Send(fmt.Errorf("accept: %w \n", err).Error())
		}

		user := dbmng.User{
			ID:               qUser.ID,
			UserName:         qUser.UserName,
			Enabled:          0,
			TOTPSecret:       qUser.TOTPSecret,
			Session:          0,
			SessionTimeStamp: "never",
			Peer:             qUser.Peer,
			PeerPre:          qUser.PeerPre,
			PeerPub:          qUser.PeerPub,
			AllowedIPs:       tga[1],
			IP:               qUser.IP,
		}

		err = s.RegisterUser(&user)
		if err != nil {
			logger.Error().Err(err).Msg("accept")
			return c.Send(err.Error())
		}

		err = s.GetUsersIDs(&uDBids)
		if err != nil {
			logger.Error().Err(err).Msg("accept")
			return c.Send(err.Error())
		}
		err = s.GetQueueUsersIDs(&qDBids)
		if err != nil {
			logger.Error().Err(err).Msg("accept")
			return c.Send(err.Error())
		}

		return c.Send("Пользователь успешно добавлен")
	})

	tg.Handle("/adduser", func(c tele.Context) error {
		// Get telegram user info
		var (
			tgu = c.Sender()
			tga = c.Args()
		)

		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("adduser: non admin user tried to use /adduser" + strconv.FormatInt(tgu.ID, 10))
			return c.Send("Unknown")
		}

		if len(tga) != 8 {
			return c.Send("Ошибка введенных параметров")
		}

		id, err := strconv.ParseInt(tga[0], 10, 64)
		if err != nil {
			return c.Send(err.Error())
		}
		enabled, err := strconv.Atoi(tga[2])
		if err != nil {
			return c.Send(err.Error())
		}
		ip, err := strconv.Atoi(tga[8])
		if err != nil {
			return c.Send(err.Error())
		}

		user := dbmng.User{
			ID:               id,
			UserName:         tga[1],
			Enabled:          enabled,
			TOTPSecret:       tga[3],
			Session:          0,
			SessionTimeStamp: "never",
			Peer:             tga[4],
			PeerPre:          tga[5],
			PeerPub:          tga[6],
			AllowedIPs:       tga[7],
			IP:               ip,
		}

		err = s.RegisterUser(&user)
		if err != nil {
			logger.Error().Err(err).Msg("adduser")
			return c.Send(err.Error())
		}
		err = s.GetUsersIDs(&uDBids)
		if err != nil {
			logger.Error().Err(err).Msg("adduser")
			return c.Send(err.Error())
		}

		return c.Send("Пользователь добавлен")
	})

	tg.Handle("/sendcreds", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
			buf bytes.Buffer
		)

		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("sendcreds: non admin user tried to use /sendcreds" + strconv.FormatInt(tgu.ID, 10))
			return c.Send("Unknown")
		}

		if len(tga) != 1 {
			return c.Send("Ошибка введенных параметров")
		}

		id, err := strconv.ParseInt(tga[0], 10, 64)
		if err != nil {
			return c.Send(err.Error())
		}

		user, err := s.GetUser(&id)
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		img, err := key.Image(256, 256)
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		err = png.Encode(&buf, img)
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		p := &tele.Photo{File: tele.FromReader(&buf)}

		_, err = tg.Send(tele.ChatID(id), p, &tele.SendOptions{Protected: true})
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		err = em.SendEmail(&user)
		if err != nil {
			logger.Error().Err(err).Msg("sendcreds")
			return c.Send(err.Error())
		}

		return c.Send("Креды отправлены")
	})

	tg.Handle("/enable", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
		)
		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("enable: non admin user tried to use /enable" + strconv.FormatInt(tgu.ID, 10))
			return c.Send("Unknown")
		}

		user, err := s.GetUser(&tgu.ID)
		if err != nil {
			logger.Error().Err(err).Msg("enable")
			return c.Send(err.Error())
		}

		err = s.EnableUser(&user.ID)
		if err != nil {
			logger.Error().Err(err).Msg("enable")
			return c.Send("Не удалось активировать пользователя")
		}

		return c.Send("Пользователь " + tga[0] + " активирован")
	})

	tg.Handle("/disable", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
		)

		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("disable: non admin user tried to use /disable" + strconv.FormatInt(tgu.ID, 10))
			return c.Send("Unknown")
		}

		user, err := s.GetUser(&tgu.ID)
		if err != nil {
			logger.Error().Err(err).Msg("enable")
			return c.Send(err.Error())
		}

		if sessionManager[user.ID] {
			err = wg.WgStopSession(&user, messageManager[user.ID])
			if err != nil {
				logger.Error().Err(err).Msg("disable")
				return c.Send(err.Error())
			}
			logger.Info().Msg("disable: forcefully stopped session of: " + user.UserName)
			sessionManager[user.ID] = false
		}

		err = s.DisableUser(&user.ID)
		if err != nil {
			logger.Error().Err(err).Msg("disable")
			return c.Send("Не удалось деактивировать пользователя")
		}

		return c.Send("Пользователь " + tga[0] + " деактивирован")
	})

	tg.Handle("/get", func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tga = c.Args()
		)

		if !slices.Contains(aDBids, tgu.ID) {
			logger.Error().Msg("get: non admin user tried to use /get" + strconv.FormatInt(tgu.ID, 10))
			return c.Send("Unknown")
		}

		user, err := s.GetUserName(&tga[0])
		if err != nil {
			logger.Error().Err(err).Msg("get")
			return c.Send(err.Error())
		}
		return c.Send(strconv.FormatInt(user.ID, 10)+" | "+user.UserName+" | "+"192.168.88."+strconv.Itoa(user.IP), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	})

	tg.Handle(tele.OnText, func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tgt = c.Text()
		)

		if !funk.ContainsInt64(uDBids, tgu.ID) {
			logger.Error().Msg("unregistered user sent message:" + strconv.FormatInt(tgu.ID, 10) + " " + tgu.Username)
			return c.Send("Error")
		}

		user, err := s.GetUser(&tgu.ID)
		if err != nil {
			logger.Error().Err(err).Msg("validation")
			_, err = tg.Send(tele.ChatID(adminLogChat), err.Error(), &tele.SendOptions{ThreadID: adminLogChatThread})
			if err != nil {
				logger.Error().Err(err).Msg("failed to send message")
			}
			return c.Send("Произошла ошибка, обратитесь к администратору")
		}

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			logger.Error().Err(err).Msg("validation")
			_, err = tg.Send(tele.ChatID(adminLogChat), err.Error(), &tele.SendOptions{ThreadID: adminLogChatThread})
			if err != nil {
				logger.Error().Err(err).Msg("failed to send message")
			}
			return c.Send("Произошла ошибка, обратитесь к администратору")
		}

		if !totp.Validate(tgt, key.Secret()) {
			logger.Info().Msg(user.UserName + " failed validation")
			return c.Send("Неверный код")
		}

		if sessionManager[tgu.ID] {
			return c.Send("Сессия уже запущена")
		}

		err = wg.WgStartSession(&user)
		if err != nil {
			logger.Error().Err(err).Msg("validation")
			return c.Send("Ошибка создания сессии, обратитесь к администратору")
		}

		logger.Info().Msg("session started for: " + user.UserName)

		return c.Send("Сессия создана")
	})

	tg.Start()
}
