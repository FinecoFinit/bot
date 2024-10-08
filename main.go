package main

import (
	"bot/pkg/dbmng"
	"bot/pkg/wgmng"
	"bytes"
	"database/sql"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pquerna/otp/totp"

	tele "gopkg.in/telebot.v3"
)

func main() {
	var (
		aDBids         []int64
		sessionManager = make(map[int64]bool)
		adminChat      int64
	)
	pref := tele.Settings{Token: os.Getenv("TOKEN"), Poller: &tele.LongPoller{Timeout: 10 * time.Second}}

	tg, err := tele.NewBot(pref)
	if err != nil {
		panic(fmt.Errorf("ENV: TOKEN parse error: %w", err))
	}
	adminChat, err = strconv.ParseInt(os.Getenv("ADMIN-CHAT"), 10, 64)
	if err != nil {
		panic(fmt.Errorf("ENV: ADMINCHAT parse error: %w", err))
	}

	// Locate DB
	location, err := filepath.Abs(os.Getenv("DB"))
	if err != nil {
		panic(fmt.Errorf("db file: file doesn't exists or corrupted: %w", err))
	}
	// Open SQL DB
	db, err := sql.Open("sqlite3", location)
	if err != nil {
		panic(fmt.Errorf("db: failed to open db %w", err))
	}
	s := dbmng.DB{Db: db}
	wg := wgmng.HighWay{Db: db, Tg: tg, SessionManager: sessionManager, AdminChat: adminChat}

	admins, err := s.GetAdmins()
	if err != nil {
		panic(fmt.Errorf("db: failed to get admin ids: %w", err))
	}
	for _, u := range admins {
		aDBids = append(aDBids, u.ID)
	}

	tg.Handle("/register", func(c tele.Context) error {
		// Get telegram user info
		var (
			tgu       = c.Sender()
			tga       = c.Args()
			userDBids []int64
			qDBids    []int64
		)

		if len(tga) != 1 {
			return c.Send("Ошибка введенных параметров")
		}

		users, err := s.GetUsers()
		if err != nil {
			fmt.Println(fmt.Errorf("db: failed to get users ids: %w", err))
			return c.Send("Временная ошибка, сообщите администратору")
		}
		for _, u := range users {
			userDBids = append(userDBids, u.ID)
		}

		queue, err := s.GetQueueUsers()
		if err != nil {
			fmt.Println(fmt.Errorf("db: failed to get registration queue ids: %w", err))
			return c.Send("Временная ошибка, сообщите администратору")
		}
		for _, u := range queue {
			qDBids = append(qDBids, u.ID)
		}

		if slices.Contains(userDBids, tgu.ID) {
			return c.Send("Пользователь существует")
		}
		if slices.Contains(qDBids, tgu.ID) {
			return c.Send("Регистрация в процессе")
		} else {
			err = s.RegisterQueue(tgu.ID, tga[0])
			if err != nil {
				fmt.Println(err)
				return c.Send("Временная ошибка, сообщите администратору")
			}
			_, err = tg.Send(
				tele.ChatID(1254517365),
				"В очередь добавлен новый пользователь:\nID: "+strconv.FormatInt(tgu.ID, 10)+"\nusername: @"+tgu.Username+"\nlogin: "+tga[0]+"\n`\n/accept "+strconv.FormatInt(tgu.ID, 10)+" [AllowedIP]`", &tele.SendOptions{
					ParseMode: "MarkdownV2",
				})
			if err != nil {
				return c.Send(err.Error())
			}
			return c.Send("Заявка на регистрацию принята")
		}
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
			AllowedIPs:       tga[1],
			IP:               qUser.IP,
		}

		err = s.RegisterUser(&user)
		if err != nil {
			return c.Send(err)
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
		ip, err := strconv.Atoi(tga[7])
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
			PeerPub:          tga[5],
			AllowedIPs:       tga[6],
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
		user, err := s.GetUser(&tgu.ID)
		if err != nil {
			fmt.Println(err)
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			fmt.Println(err)
		}
		if totp.Validate(tgt, key.Secret()) {
			err := wg.WgStartSession(&user)
			if err != nil {
				return c.Send("Ошибка создания сессии, обратитесь к администратору")
			}
			return c.Send("Сессия создана")
		}

		return c.Send(tgt)
	})

	tg.Start()
}
