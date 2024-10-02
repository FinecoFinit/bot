package main

import (
	dbmodule "bot/pkg/db"
	wgmng "bot/pkg/wgmng"
	"bytes"
	"database/sql"
	"image/png"
	"log"
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
		adminDBids []int64
	)
	pref := tele.Settings{Token: os.Getenv("TOKEN"), Poller: &tele.LongPoller{Timeout: 10 * time.Second}}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	// Locate DB
	location, err := filepath.Abs(os.Getenv("DB"))
	if err != nil {
		log.Fatal(err)
	}
	// Open SQL DB
	db, err := sql.Open("sqlite3", location)
	if err != nil {
		log.Fatal(err)
	}
	s := dbmodule.DB{Db: db}
	wg := wgmng.DB{Db: db}

	admins, err := s.GetAdmins()
	if err != nil {
		panic(err)
	}
	for _, u := range admins {
		adminDBids = append(adminDBids, u.ID)
	}

	b.Handle("/register", func(c tele.Context) error {
		// Get telegram user info
		var (
			tguser     = c.Sender()
			tgargs     = c.Args()
			userDBids  []int64
			queueDBids []int64
		)

		if len(tgargs) != 1 {
			return c.Send("Ошибка введенных параметров")
		}

		users, err := s.GetUsers()
		if err != nil {
			panic(err)
		}
		for _, u := range users {
			userDBids = append(userDBids, u.ID)
		}

		queue, err := s.GetQueueUsers()
		if err != nil {
			panic(err)
		}
		for _, u := range queue {
			queueDBids = append(queueDBids, u.ID)
		}

		// Registration logic
		if slices.Contains(userDBids, tguser.ID) {
			return c.Send("Пользователь существует")
		}
		if slices.Contains(queueDBids, tguser.ID) {
			return c.Send("Регистрация в процессе")
		} else {
			// Register user in queue
			err = s.RegisterQueue(tguser.ID, tgargs[0])
			if err != nil {
				return c.Send(err)
			}

			// Send request to chat
			b.Send(tele.ChatID(1254517365), "В очередь добавлен новый пользователь:\nID: "+strconv.FormatInt(tguser.ID, 10)+"\nusername: @"+tguser.Username+"\nlogin: "+tgargs[0]+"\n`\n/accept "+strconv.FormatInt(tguser.ID, 10)+" [AllowedIP]`", &tele.SendOptions{
				ParseMode: "MarkdownV2",
			})

			return c.Send("Заявка на регистрацию принята")
		}
	})

	b.Handle("/accept", func(c tele.Context) error {
		// Get telegram user info
		var (
			tguser = c.Sender()
			tgargs = c.Args()
		)
		if !slices.Contains(adminDBids, tguser.ID) {
			return c.Send("Unknown")
		}
		if len(tgargs) != 2 {
			return c.Send("Ошибка введенных параметров")
		}
		id, err := strconv.ParseInt(tgargs[0], 10, 64)
		if err != nil {
			return c.Send(err)
		}

		queueuser, err := s.GetQueueUser(&id)
		if err != nil {
			return c.Send(err)
		}

		user := dbmodule.User{
			ID:               queueuser.ID,
			UserName:         queueuser.UserName,
			Enabled:          0,
			TOTPSecret:       queueuser.TOTPSecret,
			Session:          0,
			SessionTimeStamp: "never",
			Peer:             queueuser.Peer,
			Allowedips:       tgargs[1],
			IP:               queueuser.IP,
		}

		err = s.RegisterUser(&user)
		if err != nil {
			return (err)
		}
		return c.Send("Пользователь успешно добавлен")
	})

	b.Handle("/adduser", func(c tele.Context) error {
		// Get telegram user info
		var (
			tguser = c.Sender()
			tgargs = c.Args()
		)
		if !slices.Contains(adminDBids, tguser.ID) {
			return c.Send("Unknown")
		}
		if len(tgargs) != 7 {
			return c.Send("Ошибка введенных параметров")
		}
		id, err := strconv.ParseInt(tgargs[0], 10, 64)
		if err != nil {
			return c.Send("1")
		}
		enabled, err := strconv.Atoi(tgargs[2])
		if err != nil {
			return c.Send("2")
		}
		ip, err := strconv.Atoi(tgargs[6])
		if err != nil {
			return c.Send("3")
		}
		user := dbmodule.User{
			ID:               id,
			UserName:         tgargs[1],
			Enabled:          enabled,
			TOTPSecret:       tgargs[3],
			Session:          0,
			SessionTimeStamp: "never",
			Peer:             tgargs[4],
			Allowedips:       tgargs[5],
			IP:               ip,
		}
		err = s.RegisterUser(&user)
		if err != nil {
			panic(err)
		}

		return c.Send("Пользователь добавлен")
	})

	b.Handle("/sendinfo", func(c tele.Context) error {
		var (
			tguser = c.Sender()
			tgargs = c.Args()
		)
		if !slices.Contains(adminDBids, tguser.ID) {
			return c.Send("Unknown")
		}
		if len(tgargs) != 1 {
			return c.Send("Ошибка введенных параметров")
		}
		id, err := strconv.ParseInt(tgargs[0], 10, 64)
		if err != nil {
			return c.Send("1")
		}
		user, err := s.GetUser(&id)
		if err != nil {
			return err
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			return err
		}
		img, err := key.Image(256, 256)
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		png.Encode(&buf, img)
		p := &tele.Photo{File: tele.FromReader(&buf)}
		_, err = b.Send(tele.ChatID(id), p, &tele.SendOptions{})
		if err != nil {
			return err
		}
		return c.Send("err")
	})

	b.Handle("/enable", func(c tele.Context) error {
		var (
			tguser = c.Sender()
			tgargs = c.Args()
		)
		if !slices.Contains(adminDBids, tguser.ID) {
			return c.Send("Unknown")
		}
		err = s.EnableUser(&tgargs[0])
		if err != nil {
			return c.Send("Не удалось активировать пользователя")
		}
		return c.Send("Пользователь " + tgargs[0] + " активирован")
	})

	b.Handle("/disable", func(c tele.Context) error {
		var (
			tguser = c.Sender()
			tgargs = c.Args()
		)
		if !slices.Contains(adminDBids, tguser.ID) {
			return c.Send("Unknown")
		}
		err = s.DisableUser(&tgargs[0])
		if err != nil {
			return c.Send("Не удалось деактивировать пользователя")
		}
		return c.Send("Пользователь " + tgargs[0] + " деактивирован")
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		var (
			tguser = c.Sender()
			tgtext = c.Text()
		)
		user, err := s.GetUser(&tguser.ID)
		if err != nil {
			return err
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "test",
			AccountName: user.UserName,
			Secret:      []byte(user.TOTPSecret)})
		if err != nil {
			return err
		}
		if totp.Validate(tgtext, key.Secret()) {
			err := wg.WgStartSession(&user)
			if err != nil {
				return err
			}
			return c.Send("good")
		}

		return c.Send(tgtext)
	})

	b.Start()
}
