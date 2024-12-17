package tg

import (
	"bot/concierge"
	"fmt"
	"slices"
	"strconv"
	"strings"
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
		return c.Send("```\n/register email```", &tele.SendOptions{ParseMode: "MarkdownV2"})
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
		"В очередь добавлен новый пользователь:\n🆔: ``"+strconv.FormatInt(c.Sender().ID, 10)+
			"``\n👔: @"+c.Sender().Username+
			"\n✉️: "+strings.Replace(c.Args()[0], ".", "\\.", 1), &tele.SendOptions{
			ThreadID:  t.Config.AdminWgChatThread,
			ParseMode: "MarkdownV2",
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
		return c.Send("```\n/accept id allowedips```", &tele.SendOptions{ParseMode: "MarkdownV2"})
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
		return c.Send("```\n/adduser id email 0/1(disable/enabled) totp_secret wg_private wg_preshared wg_public allowedips ip```", &tele.SendOptions{ParseMode: "MarkdownV2"})
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
		return c.Send("```\n/deluser id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
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
		return c.Send("```\n/sendcreds id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
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

func (t Telegram) Enable(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("enable: non admin user tried to use /enable " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/enable id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("enable")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = t.Storage.EnableUser(&user.ID)
	if err != nil {
		t.Logger.Error().Err(err).Msg("enable")
		return c.Send("Не удалось активировать пользователя", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь "+c.Args()[0]+" активирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) Disable(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("enable: non admin user tried to use /disable " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/disable id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("enable")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if t.Managers.SessionManager[user.ID] {
		err = t.Wireguard.WgStopSession(&user)
		if err != nil {
			t.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		err = t.SessionEnded(user)
		if err != nil {
			t.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		t.Logger.Info().Msg("disable: forcefully stopped session of: " + user.UserName)
		t.Managers.SessionManager[user.ID] = false
	}

	err = t.Storage.DisableUser(&user.ID)
	if err != nil {
		t.Logger.Error().Err(err).Msg("disable")
		return c.Send("Не удалось деактивировать пользователя", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь "+c.Args()[0]+" деактивирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (t Telegram) Get(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("enable: non admin user tried to use /get " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/get email```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	user, err := t.Storage.GetUserName(&c.Args()[0])
	if err != nil {
		t.Logger.Error().Err(err).Msg("get")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	return c.Send(strconv.FormatInt(user.ID, 10)+" | "+user.UserName+" | "+user.AllowedIPs+" | "+t.Config.WgSubNet+strconv.Itoa(user.IP), &tele.SendOptions{ThreadID: c.Message().ThreadID})
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
	err = t.SessionStarted(user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("validation")
		return c.Send("Ошибка создания сессии, обратитесь к администратору")
	}
	t.Managers.SessionManager[user.ID] = true
	go t.Session(&user, time.Now(), t.Managers.MessageManager[user.ID])

	t.Logger.Info().Msg("session started for: " + user.UserName)

	return c.Send("Сессия создана")
}

func (t Telegram) Edit(c tele.Context) error {
	if !funk.ContainsInt64(*t.Managers.UserDBIDs, c.Sender().ID) {
		t.Logger.Warn().Msg("unregistered user sent message:" + strconv.FormatInt(c.Sender().ID, 10) + " " + c.Sender().Username)
		return c.Send("Error")
	}

	if c.Args() == nil {
		return c.Send("```\n/edit id param value```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("edit")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = t.Storage.Edit(&user, c.Args()[1], c.Args()[2])
	if err != nil {
		t.Logger.Error().Err(err).Msg("edit")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Изменение успешно произведено", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}
