package main

import (
	"bot/pkg/dbmng"
	"bot/pkg/emailmng"
	"bot/pkg/wgmng"
	"bytes"
	"database/sql"
	"fmt"
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
		aDBids         []int64
		sessionManager = make(map[int64]bool)
		adminChat      int64
		uDBids         []int64
		qDBids         []int64
		wgSerIP        = os.Getenv("WG_SER_IP")
		wgPubKey       = os.Getenv("WG_SER_PUBK")
		token          = os.Getenv("TOKEN")
		adminChatID    = os.Getenv("ADMIN_CHAT")
		dbPath         = os.Getenv("DB")
		emailUser      = os.Getenv("EMAIL_LOGIN")
		emailPass      = os.Getenv("EMAIL_PASS")
		emailAddr      = os.Getenv("EMAIL_ADDR")
	)
	pref := tele.Settings{Token: token, Poller: &tele.LongPoller{Timeout: 10 * time.Second}}

	tg, err := tele.NewBot(pref)
	if err != nil {
		panic(fmt.Errorf("ENV: TOKEN parse error: %w", err))
	}
	adminChat, err = strconv.ParseInt(adminChatID, 10, 64)
	if err != nil {
		panic(fmt.Errorf("ENV: ADMIN_CHAT parse error: %w", err))
	}

	// Locate DB
	location, err := filepath.Abs(dbPath)
	if err != nil {
		panic(fmt.Errorf("db file: file doesn't exists or corrupted: %w", err))
	}
	// Open SQL DB
	db, err := sql.Open("sqlite3", location)
	if err != nil {
		panic(fmt.Errorf("db: failed to open db %w", err))
	}

	emailc, err := mail.NewClient(emailAddr, mail.WithPort(587), mail.WithSSL(), mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(emailUser), mail.WithPassword(emailPass))
	if err != nil {
		panic(fmt.Errorf("failed to create mail client: %s", err))
	}

	s := dbmng.DB{Db: db}
	wg := wgmng.HighWay{Db: db, Tg: tg, SessionManager: sessionManager, AdminChat: adminChat}
	em := emailmng.HighWay{WgServerIP: &wgSerIP, WgPublicKey: &wgPubKey, EmailClient: emailc, EmailUser: &emailUser, EmailPass: &emailPass, EmailAddr: &emailAddr}

	admins, err := s.GetAdmins()
	if err != nil {
		panic(fmt.Errorf("db: failed to get admin ids: %w", err))
	}
	for _, u := range admins {
		aDBids = append(aDBids, u.ID)
	}

	err = s.GetUsersIDs(&uDBids)
	if err != nil {
		panic(fmt.Errorf("db: failed to get user ids: %w", err))
	}
	err = s.GetQueueUsersIDs(&qDBids)
	if err != nil {
		panic(fmt.Errorf("db: failed to get queue ids: %w", err))
	}

	tg.Handle("/register", func(c tele.Context) error {
		// Get telegram user info
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
			fmt.Println(err)
			return c.Send("Временная ошибка, сообщите администратору")
		}
		_, err = tg.Send(
			tele.ChatID(1254517365),
			"В очередь добавлен новый пользователь:\nID: ``"+strconv.FormatInt(tgu.ID, 10)+
				"``\nusername: @"+tgu.Username+
				"\nlogin: "+strings.Replace(tga[0], ".", "\\.", 1)+
				"\n`\n/accept "+strconv.FormatInt(tgu.ID, 10)+" [AllowedIP]`", &tele.SendOptions{
				ParseMode: "MarkdownV2"})
		if err != nil {
			return c.Send(err.Error())
		}
		return c.Send("Заявка на регистрацию принята")
	})

	tg.Handle("/accept", func(c tele.Context) error {
		// Get telegram user info
		var (
			tgu = c.Sender()
			tga = c.Args()
		)
		if !slices.Contains(aDBids, tgu.ID) {
			return c.Send("Unknown")
		}
		if len(tga) != 2 {
			return c.Send("Ошибка введенных параметров")
		}
		id, err := strconv.ParseInt(tga[0], 10, 64)
		if err != nil {
			return c.Send("Не удалось считать ID")
		}

		qUser, err := s.GetQueueUser(&id)
		if err != nil {
			return c.Send(err)
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
			return c.Send("Unknown")
		}
		if len(tga) != 8 {
			return c.Send("Ошибка введенных параметров")
		}
		id, err := strconv.ParseInt(tga[0], 10, 64)
		if err != nil {
			return c.Send("1")
		}
		enabled, err := strconv.Atoi(tga[2])
		if err != nil {
			return c.Send("2")
		}
		ip, err := strconv.Atoi(tga[8])
		if err != nil {
			return c.Send("3")
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
			panic(err)
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
			return c.Send(err.Error())
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			return c.Send(err.Error())
		}
		img, err := key.Image(256, 256)
		if err != nil {
			return c.Send(err.Error())
		}
		err = png.Encode(&buf, img)
		if err != nil {
			return c.Send(err.Error())
		}
		p := &tele.Photo{File: tele.FromReader(&buf)}
		_, err = tg.Send(tele.ChatID(id), p, &tele.SendOptions{})
		if err != nil {
			return c.Send(err.Error())
		}
		err = em.SendEmail(&user)
		if err != nil {
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
			return c.Send("Unknown")
		}
		err = s.EnableUser(&tga[0])
		if err != nil {
			fmt.Println(err)
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
			return c.Send("Unknown")
		}
		err = s.DisableUser(&tga[0])
		if err != nil {
			fmt.Println(err)
			return c.Send("Не удалось деактивировать пользователя")
		}
		return c.Send("Пользователь " + tga[0] + " деактивирован")
	})

	tg.Handle(tele.OnText, func(c tele.Context) error {
		var (
			tgu = c.Sender()
			tgt = c.Text()
		)
		if !funk.ContainsInt64(uDBids, tgu.ID) {
			return c.Send("Error")
		}
		user, err := s.GetUser(&tgu.ID)
		if err != nil {
			fmt.Println(fmt.Errorf("validation: failed to get user from db: %w", err))
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Secret: []byte(user.TOTPSecret)})
		if err != nil {
			fmt.Println(fmt.Errorf("validation: failed to get user totp key: %w", err))
		}
		if !totp.Validate(tgt, key.Secret()) {
			return c.Send("Неверный код")
		}
		err = wg.WgStartSession(&user)
		if err != nil {
			fmt.Println(err)
			return c.Send("Ошибка создания сессии, обратитесь к администратору")
		}
		return c.Send("Сессия создана")
	})
	tg.Start()
}
