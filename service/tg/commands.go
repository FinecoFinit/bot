package tg

import (
	"bot/concierge"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thoas/go-funk"
	tele "gopkg.in/telebot.v3"
)

func (t Telegram) Start(c tele.Context) error {
	return c.Send("Для дальнейшего инструктажа обратитесь к документации в базе знаний")
}

func (t Telegram) Register(c tele.Context) error {

	if c.Args() == nil {
		return c.Send("```\n/register email```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	if len(c.Args()) != 1 {
		return c.Send("Ошибка введенных параметров")
	}

	if slices.Contains(*t.Managers.UserDBIDs, c.Sender().ID) {
		return c.Send("Пользователь существует")
	}
	if slices.Contains(*t.Managers.QUserDBIDs, c.Sender().ID) {
		return c.Send("Регистрация в процессе")
	}

	err := t.Storage.RegisterQueue(c.Sender().ID, c.Args()[0])
	if err != nil {
		t.Logger.Error().Err(err).Msg("registration: failed to register user")
		return c.Send("Ошибка, сообщите администратору")
	}

	_, err = t.Tg.Send(
		tele.ChatID(t.Config.AdminWgChatID),
		"В очередь добавлен новый пользователь:\n🆔: "+strconv.FormatInt(c.Sender().ID, 10)+
			"\n👔: @"+c.Sender().Username+
			"\n✉️: "+c.Args()[0], &tele.SendOptions{
			ThreadID: t.Config.AdminWgChatThreadHelm,
			ReplyMarkup: &tele.ReplyMarkup{
				OneTimeKeyboard: true,
				InlineKeyboard: [][]tele.InlineButton{{
					tele.InlineButton{
						Unique: "register_accept",
						Text:   "Accept",
						Data:   strconv.FormatInt(c.Sender().ID, 10)},
					tele.InlineButton{
						Unique: "register_deny",
						Text:   "Deny",
						Data:   strconv.FormatInt(c.Sender().ID, 10)}}}}})
	if err != nil {
		t.Logger.Error().Err(err).Msg("registration")
	}
	err = t.Storage.GetQueueUsersIDs(t.Managers.QUserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("registration: failed to update queue ids")
	}
	t.Logger.Info().Msg("new user registered in queue: " + strconv.FormatInt(c.Sender().ID, 10))
	return c.Send("Заявка на регистрацию принята")
}

func (t Telegram) Accept(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("accept: non admin user tried to use /accept")
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/accept id allowedips```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	if len(c.Args()) != 2 {
		return c.Send("Задано неверное количество параметров", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send("Неудалось обработать ID пользователя", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	qUser, err := t.Storage.GetQueueUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Send(fmt.Errorf("accept: %w \n", err).Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user := concierge.User{
		ID:               qUser.ID,
		UserName:         qUser.UserName,
		Enabled:          0,
		TOTPSecret:       qUser.TOTPSecret,
		Session:          0,
		SessionTimeStamp: "never",
		SessionMessageID: "never",
		Peer:             qUser.Peer,
		PeerPre:          qUser.PeerPre,
		PeerPub:          qUser.PeerPub,
		AllowedIPs:       c.Args()[1],
		IP:               qUser.IP,
	}

	err = t.Storage.RegisterUser(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = t.Storage.GetUsersIDs(t.Managers.UserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = t.Storage.GetQueueUsersIDs(t.Managers.QUserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь успешно добавлен", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) AddUser(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("adduser: non admin user tried to use /adduser" + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/adduser id email 0/1(disable/enabled) totp_secret wg_private wg_preshared wg_public allowedips ip```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	if len(c.Args()) != 8 {
		return c.Send("Ошибка введенных параметров", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	enabled, err := strconv.Atoi(c.Args()[2])
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	ip, err := strconv.Atoi(c.Args()[8])
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user := concierge.User{
		ID:               id,
		UserName:         c.Args()[1],
		Enabled:          enabled,
		TOTPSecret:       c.Args()[3],
		Session:          0,
		SessionTimeStamp: "never",
		SessionMessageID: "never",
		Peer:             c.Args()[4],
		PeerPre:          c.Args()[5],
		PeerPub:          c.Args()[6],
		AllowedIPs:       c.Args()[7],
		IP:               ip,
	}

	err = t.Storage.RegisterUser(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("adduser")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = t.Storage.GetUsersIDs(t.Managers.UserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("adduser")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь добавлен", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) DelUser(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("adduser: non admin user tried to use /adduser" + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/deluser id```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	if len(c.Args()) != 1 {
		return c.Send("Ошибка введенных параметров", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("failed to get user")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if t.Managers.SessionManager[user.ID] {
		err = t.Wireguard.WgStopSession(&user)
		if err != nil {
			t.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		err = t.NoticeSessionEnded(user)
		if err != nil {
			t.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		_, err = t.Tg.Edit(t.Managers.MessageManager[user.ID], t.Managers.MessageManager[user.ID].Text+"\nСессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
		t.Logger.Info().Msg("disable: forcefully stopped session of: " + user.UserName)
		delete(t.Managers.MessageManager, user.ID)
		delete(t.Managers.SessionManager, user.ID)
	}

	err = t.Storage.UnregisterUser(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("failed to unregister user")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = t.Storage.GetUsersIDs(t.Managers.UserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь: "+user.UserName+" удален", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) SendCreds(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("sendcreds: non admin user tried to use /sendcreds" + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/sendcreds id```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	if len(c.Args()) != 1 {
		return c.Send("Ошибка введенных параметров", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("sendcreds")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = t.Email.SendEmail(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("sendcreds")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	_, err = t.Tg.Send(tele.ChatID(user.ID), "Регистрация завершена, на почту отправлен QR-код двухфакторной аутентификации и конфигурационный файл, далее требуется только ввод двухфакторного кода")
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Креды отправлены", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) Get(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("enable: non admin user tried to use /get " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/get user/sessions email```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	switch c.Args()[0] {
	case "user":
		user, err := t.Storage.GetUserName(&c.Args()[1])
		if err != nil {
			t.Logger.Error().Err(err).Msg("get")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send(strconv.FormatInt(user.ID, 10)+" | "+user.UserName+" | "+user.AllowedIPs+" | "+t.Config.WgSubNet+strconv.Itoa(user.IP), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "sessions":
		var msg string
		for u := range t.Managers.SessionManager {
			user, err := t.Storage.GetUser(&u)
			if err != nil {
				t.Logger.Error().Err(err).Msg("get: sessions:")
			}
			msg = msg + strconv.FormatInt(u, 10) + " - " + user.UserName + "\n"
		}
		if msg == "" {
			return c.Send("No sessions", &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send(msg, &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "timed":
		var msg string
		for u := range t.Managers.TimedManager {
			user, err := t.Storage.GetUser(&u)
			if err != nil {
				t.Logger.Error().Err(err).Msg("get: timed:")
			}
			msg = msg + strconv.FormatInt(u, 10) + " - " + user.UserName + "\n"
		}
		if msg == "" {
			return c.Send("No timed users", &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send(msg, &tele.SendOptions{ThreadID: c.Message().ThreadID})
	default:
		return c.Send("Unknown argument", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
}

func (t Telegram) Set(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("enable: non admin user tried to use /enable " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/timedenable date id```", &tele.SendOptions{ParseMode: "MarkdownV2", ThreadID: c.Message().ThreadID})
	}

	id, err := strconv.ParseInt(c.Args()[1], 10, 64)
	if err != nil {
		t.Logger.Error().Err(err).Msg("set: failed to parse id")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("set: failed to find user")
	}

	switch c.Args()[0] {
	case "timed":
		st, err := time.Parse("2006-01-02 15:04:05 Z0700", c.Args()[2]+" 08:00:00 "+time.Now().Format("Z0700"))
		if err != nil {
			t.Logger.Error().Err(err).Msg("timedenenable failed to parse time")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		if time.Now().Before(st) {
			err = t.Storage.DisableUser(&id)
			if err != nil {
				t.Logger.Error().Err(err).Msg("timedenenable failed to disable user")
				return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
			}
		}
		err = t.Storage.AddTimedEnable(id, st)
		if err != nil {
			t.Logger.Error().Err(err).Msg("timedenenable failed")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		t.Managers.TimedManager[id] = true
		go t.Timed(id, st)
		return c.Send("Пользователь установлен в очередь", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "enable":
		err = t.Storage.EnableUser(&id)
		if err != nil {
			t.Logger.Error().Err(err).Msg("enable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send("Пользователь активирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "disable":
		if t.Managers.SessionManager[user.ID] {
			err = t.Wireguard.WgStopSession(&user)
			if err != nil {
				t.Logger.Error().Err(err).Msg("disable")
				return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
			}
			err = t.NoticeSessionEnded(user)
			if err != nil {
				t.Logger.Error().Err(err).Msg("disable")
				return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
			}
			err = t.Storage.SessionEnded(user.ID)
			if err != nil {
				t.Logger.Error().Err(err).Msg("disable")
			}
			t.Logger.Info().Msg("disable: forcefully stopped session for: " + user.UserName)
			delete(t.Managers.MessageManager, user.ID)
			delete(t.Managers.SessionManager, user.ID)
		}
		err = t.Storage.DisableUser(&id)
		if err != nil {
			t.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send("Пользователь деактивирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "ip":
		err = t.Storage.SetIp(id, c.Args()[2])
		if err != nil {
			t.Logger.Error().Err(err).Msg("ip")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send("IP адрес отредактирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	case "allowedips":
		err = t.Storage.SetAllowedIPs(id, c.Args()[2])
		if err != nil {
			t.Logger.Error().Err(err).Msg("allowedips")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		return c.Send("AllowedIPs отредактированы", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	default:
		return c.Send("Unknown argument", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
}

func (t Telegram) Verification(c tele.Context) error {
	if !funk.ContainsInt64(*t.Managers.UserDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("unregistered user sent message:" + strconv.FormatInt(c.Sender().ID, 10) + " " + c.Sender().Username)
		return c.Send("Error")
	}

	user, err := t.Storage.GetUser(&c.Sender().ID)
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
		_, err = t.Tg.Send(tele.ChatID(t.Config.AdminWgChatID), err.Error(), &tele.SendOptions{ThreadID: t.Config.AdminWgChatThread})
		if err != nil {
			t.Logger.Error().Err(err).Msg("failed to send message")
		}
		return c.Send("Произошла ошибка, обратитесь к администратору")
	}

	if user.Enabled == 0 {
		return c.Send("Аккаунт деактивирован")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      t.Config.TotpVendor,
		AccountName: user.UserName,
		Secret:      []byte(user.TOTPSecret)})
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
		_, err = t.Tg.Send(tele.ChatID(t.Config.AdminWgChatID), err.Error(), &tele.SendOptions{ThreadID: t.Config.AdminWgChatThread})
		if err != nil {
			t.Logger.Error().Err(err).Msg("failed to send message")
		}
		return c.Send("Произошла ошибка, обратитесь к администратору")
	}

	if !totp.Validate(c.Text(), key.Secret()) {
		t.Logger.Info().Msg(user.UserName + " failed validation")
		return c.Send("Неверный код")
	}

	if t.Managers.SessionManager[c.Sender().ID] {
		return c.Send("Сессия уже запущена")
	}

	err = t.Wireguard.WgStartSession(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
		return c.Send("Ошибка создания сессии, обратитесь к администратору")
	}
	msg, err := t.NoticeSessionStarted(user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
		return c.Send("Ошибка создания сессии, обратитесь к администратору")
	}
	err = t.Storage.SessionStarted(c.Sender().ID, time.Now(), msg)
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
	}
	t.Managers.MessageManager[user.ID] = msg
	t.Managers.SessionManager[user.ID] = true
	go t.Session(&user, time.Now(), t.Managers.MessageManager[user.ID])

	t.Logger.Info().Msg("session started for: " + user.UserName)

	return c.Send("Сессия создана")
}

func (t Telegram) Reload(c tele.Context) error {
	err := t.Storage.GetAdminsIDs(t.Managers.AdminDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("update:")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = t.Storage.GetUsersIDs(t.Managers.UserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("update:")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = t.Storage.GetQueueUsersIDs(t.Managers.QUserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("update:")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	return c.Send("Updated")
}
