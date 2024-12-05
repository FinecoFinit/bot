package tg

import (
	"bot/pkg/db"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pquerna/otp/totp"
	"github.com/thoas/go-funk"
	tele "gopkg.in/telebot.v4"
)

func (h HighWay) Start(c tele.Context) error {
	return c.Send("Hello")
}

func (h HighWay) Register(c tele.Context) error {

	if c.Args() == nil {
		return c.Send("```\n/register email```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	if len(c.Args()) != 1 {
		return c.Send("Ошибка введенных параметров")
	}

	if slices.Contains(*h.Resources.UserDBIDs, c.Sender().ID) {
		return c.Send("Пользователь существует")
	}
	if slices.Contains(*h.Resources.QUserDBIDs, c.Sender().ID) {
		return c.Send("Регистрация в процессе")
	}
	err := h.DataBase.RegisterQueue(c.Sender().ID, c.Args()[0])
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("registration: failed to register user")
		return c.Send("Ошибка, сообщите администратору")
	}

	_, err = h.Tg.Send(
		tele.ChatID(h.DataVars.AdminLogChat),
		"В очередь добавлен новый пользователь:\n🆔: ``"+strconv.FormatInt(c.Sender().ID, 10)+
			"``\n👔: @"+c.Sender().Username+
			"\n✉️: "+strings.Replace(c.Args()[0], ".", "\\.", 1), &tele.SendOptions{
			ThreadID:  h.DataVars.AdminLogChatThread,
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
		h.Resources.Logger.Error().Err(err).Msg("registration")
	}
	err = h.DataBase.GetQueueUsersIDs(h.Resources.QUserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("registration: failed to update queue ids")
	}
	h.Resources.Logger.Info().Msg("new user registered in queue: " + strconv.FormatInt(c.Sender().ID, 10))
	return c.Send("Заявка на регистрацию принята")
}

func (h HighWay) Accept(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("accept: non admin user tried to use /accept")
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

	qUser, err := h.DataBase.GetQueueUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Send(fmt.Errorf("accept: %w \n", err).Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user := db.User{
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

	err = h.DataBase.RegisterUser(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.DataBase.GetUsersIDs(h.Resources.UserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = h.DataBase.GetQueueUsersIDs(h.Resources.QUserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь успешно добавлен", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) AddUser(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("adduser: non admin user tried to use /adduser" + strconv.FormatInt(c.Sender().ID, 10))
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

	user := db.User{
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

	err = h.DataBase.RegisterUser(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("adduser")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	err = h.DataBase.GetUsersIDs(h.Resources.UserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("adduser")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь добавлен", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) DelUser(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("adduser: non admin user tried to use /adduser" + strconv.FormatInt(c.Sender().ID, 10))
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

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("failed to get user")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.DataBase.UnregisterUser(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("failed to unregister user")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.DataBase.GetUsersIDs(h.Resources.UserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь: "+user.UserName+" удален", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) SendCreds(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("sendcreds: non admin user tried to use /sendcreds" + strconv.FormatInt(c.Sender().ID, 10))
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

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("sendcreds")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.EmailManager.SendEmail(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("sendcreds")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	_, err = h.Tg.Send(tele.ChatID(user.ID), "Регистрация завершена, на почту отправлен QR-код двухфакторной аутентификации и конфигурационный файл, далее требуется только ввод двухфакторного кода")
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Креды отправлены", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) Enable(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("enable: non admin user tried to use /enable " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/enable id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("enable")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.DataBase.EnableUser(&user.ID)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("enable")
		return c.Send("Не удалось активировать пользователя", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь "+c.Args()[0]+" активирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) Disable(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("enable: non admin user tried to use /disable " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/disable id```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("enable")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if h.Resources.SessionManager[user.ID] {
		err = h.WGManager.WgStopSession(&user, h.Resources.MessageManager[user.ID])
		if err != nil {
			h.Resources.Logger.Error().Err(err).Msg("disable")
			return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
		}
		h.Resources.Logger.Info().Msg("disable: forcefully stopped session of: " + user.UserName)
		h.Resources.SessionManager[user.ID] = false
	}

	err = h.DataBase.DisableUser(&user.ID)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("disable")
		return c.Send("Не удалось деактивировать пользователя", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Пользователь "+c.Args()[0]+" деактивирован", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) Get(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("enable: non admin user tried to use /get " + strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("Unknown", &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	if c.Args() == nil {
		return c.Send("```\n/get email```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	user, err := h.DataBase.GetUserName(&c.Args()[0])
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("get")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}
	return c.Send(strconv.FormatInt(user.ID, 10)+" | "+user.UserName+" | "+user.AllowedIPs+" | "+h.DataVars.WgSubNet+strconv.Itoa(user.IP), &tele.SendOptions{ThreadID: c.Message().ThreadID})
}

func (h HighWay) Verification(c tele.Context) error {
	if !funk.ContainsInt64(*h.Resources.UserDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("unregistered user sent message:" + strconv.FormatInt(c.Sender().ID, 10) + " " + c.Sender().Username)
		return c.Send("Error")
	}

	user, err := h.DataBase.GetUser(&c.Sender().ID)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("validation")
		_, err = h.Tg.Send(tele.ChatID(h.DataVars.AdminLogChat), err.Error(), &tele.SendOptions{ThreadID: h.DataVars.AdminLogChatThread})
		if err != nil {
			h.Resources.Logger.Error().Err(err).Msg("failed to send message")
		}
		return c.Send("Произошла ошибка, обратитесь к администратору")
	}

	if user.Enabled == 0 {
		return c.Send("Аккаунт деактивирован")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      h.DataVars.TotpVendor,
		AccountName: user.UserName,
		Secret:      []byte(user.TOTPSecret)})
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("validation")
		_, err = h.Tg.Send(tele.ChatID(h.DataVars.AdminLogChat), err.Error(), &tele.SendOptions{ThreadID: h.DataVars.AdminLogChatThread})
		if err != nil {
			h.Resources.Logger.Error().Err(err).Msg("failed to send message")
		}
		return c.Send("Произошла ошибка, обратитесь к администратору")
	}

	if !totp.Validate(c.Text(), key.Secret()) {
		h.Resources.Logger.Info().Msg(user.UserName + " failed validation")
		return c.Send("Неверный код")
	}

	if h.Resources.SessionManager[c.Sender().ID] {
		return c.Send("Сессия уже запущена")
	}

	err = h.WGManager.WgStartSession(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("validation")
		return c.Send("Ошибка создания сессии, обратитесь к администратору")
	}

	h.Resources.Logger.Info().Msg("session started for: " + user.UserName)

	return c.Send("Сессия создана")
}

func (h HighWay) Edit(c tele.Context) error {
	if !funk.ContainsInt64(*h.Resources.UserDBIDs, c.Sender().ID) {
		h.Resources.Logger.Warn().Msg("unregistered user sent message:" + strconv.FormatInt(c.Sender().ID, 10) + " " + c.Sender().Username)
		return c.Send("Error")
	}

	if c.Args() == nil {
		return c.Send("```\n/edit id param value```", &tele.SendOptions{ParseMode: "MarkdownV2"})
	}

	id, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("edit")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	err = h.DataBase.Edit(&user, c.Args()[1], c.Args()[2])
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("edit")
		return c.Send(err.Error(), &tele.SendOptions{ThreadID: c.Message().ThreadID})
	}

	return c.Send("Изменение успешно произведено", &tele.SendOptions{ThreadID: c.Message().ThreadID})
}
