package tg

import (
	"bot/concierge"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"
	tele "gopkg.in/telebot.v3"
)

func (t Telegram) RegisterAccept(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Error().Msg("accept: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if funk.ContainsInt64(*t.Managers.UserDBIDs, id) {
		return c.Respond(&tele.CallbackResponse{Text: "Пользователь уже зарегистрирован"})
	}

	qUser, err := t.Storage.GetQueueUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: fmt.Errorf("accept: %w \n", err).Error()})
	}

	user := concierge.User{
		ID:               qUser.ID,
		UserName:         qUser.UserName,
		Enabled:          1,
		TOTPSecret:       qUser.TOTPSecret,
		Session:          0,
		SessionTimeStamp: "never",
		SessionMessageID: "never",
		Peer:             qUser.Peer,
		PeerPre:          qUser.PeerPre,
		PeerPub:          qUser.PeerPub,
		AllowedIPs:       t.Config.WgAllowedIps,
		IP:               qUser.IP,
	}

	err = t.Storage.RegisterUser(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	t.Logger.Info().Msg("user: " + user.UserName + " with id: " + strconv.FormatInt(user.ID, 10) + " registered")

	_, err = t.Tg.Edit(c.Message(), c.Message().Text+"\nПользователь добавлен", &tele.SendOptions{
		ReplyMarkup: &tele.ReplyMarkup{
			OneTimeKeyboard: true,
			InlineKeyboard: [][]tele.InlineButton{{
				tele.InlineButton{
					Unique: "send_creds",
					Text:   "Отправить креды",
					Data:   strconv.FormatInt(user.ID, 10)}}}},
	})
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	err = t.Storage.GetUsersIDs(t.Managers.UserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	err = t.Storage.GetQueueUsersIDs(t.Managers.QUserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Пользователь: " + user.UserName + " добавлен", ShowAlert: false})
}

func (t Telegram) RegisterDeny(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Error().Msg("deny: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if !funk.ContainsInt64(*t.Managers.QUserDBIDs, id) {
		return c.Respond(&tele.CallbackResponse{Text: "Пользователь не существует в списке на регистрацию"})
	}

	qUser, err := t.Storage.GetQueueUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: fmt.Errorf("accept: %w \n", err).Error()})
	}

	err = t.Storage.UnRegisterQUser(&qUser)
	if err != nil {
		t.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	t.Logger.Info().Msg("queue user: " + qUser.UserName + " with id: " + strconv.FormatInt(qUser.ID, 10) + " unregistered")

	_, err = t.Tg.Edit(c.Message(), c.Message().Text+"\nПользователь отклонен")
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	err = t.Storage.GetQueueUsersIDs(t.Managers.QUserDBIDs)
	if err != nil {
		t.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Пользователь: " + qUser.UserName + " отклонен", ShowAlert: false})
}

func (t Telegram) StopSession(c tele.Context) error {
	if !slices.Contains(*t.Managers.AdminDBIDs, c.Sender().ID) {
		t.Logger.Error().Msg("stop_session: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if !t.Managers.SessionManager[id] {
		_, err := t.Tg.Edit(c.Message(), strings.ReplaceAll(c.Message().Text, ".", "\\.")+"\nСессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
		if err != nil {
			t.Logger.Error().Err(err).Msg("stop_session: failed to edit session message")
		}
		return c.Respond(&tele.CallbackResponse{Text: "Сессия не существует"})
	}

	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("stop_session: failed to get user")
		return c.Respond(&tele.CallbackResponse{Text: "db: Не удалось получить профиль пользователя"})
	}

	err = t.Wireguard.WgStopSession(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("stop_session: failed to stop session from button")
		return c.Respond(&tele.CallbackResponse{Text: "wg: Не удалось остановить сессию"})
	}

	err = t.Storage.SessionEnded(id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("stop_session: failed to stop session from button")
		return c.Respond(&tele.CallbackResponse{Text: "wg: Не удалось остановить сессию"})
	}

	delete(t.Managers.MessageManager, user.ID)
	delete(t.Managers.SessionManager, user.ID)

	_, err = t.Tg.Edit(c.Message(), strings.ReplaceAll(c.Message().Text, ".", "\\.")+"\nСессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
	if err != nil {
		t.Logger.Error().Err(err).Msg("stop_session: failed to edit session message")
		return c.Respond(&tele.CallbackResponse{Text: "wg: Не удалось остановить сессию"})
	}

	return c.Respond(&tele.CallbackResponse{Text: "Сессия пользователя " + user.UserName + " остановлена", ShowAlert: false})
}

func (t Telegram) SendCredsBtn(c tele.Context) error {
	id, err := strconv.ParseInt(c.Data(), 10, 64)
	user, err := t.Storage.GetUser(&id)
	if err != nil {
		t.Logger.Error().Err(err).Msg("send_creds_btn")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	err = t.Email.SendEmail(&user)
	if err != nil {
		t.Logger.Error().Err(err).Msg("send_creds_btn")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	_, err = t.Tg.Send(tele.ChatID(user.ID), "На почту отправлен конфигурационный файл и QR код для двухфакторной аутентификации, далее требуется вводить 2FA код для запуска сессии")
	if err != nil {
		t.Logger.Error().Err(err).Msg("send_creds_btn")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Креды отправлены", ShowAlert: false})
}
