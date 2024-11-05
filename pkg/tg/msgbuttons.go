package tg

import (
	"bot/pkg/db"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"
	tele "gopkg.in/telebot.v4"
)

func (h HighWay) RegisterAccept(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Error().Msg("accept: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if funk.ContainsInt64(*h.Resources.UserDBIDs, id) {
		return c.Respond(&tele.CallbackResponse{Text: "Пользователь уже зарегистрирован"})
	}

	qUser, err := h.DataBase.GetQueueUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: fmt.Errorf("accept: %w \n", err).Error()})
	}

	user := db.User{
		ID:               qUser.ID,
		UserName:         qUser.UserName,
		Enabled:          1,
		TOTPSecret:       qUser.TOTPSecret,
		Session:          0,
		SessionTimeStamp: "never",
		Peer:             qUser.Peer,
		PeerPre:          qUser.PeerPre,
		PeerPub:          qUser.PeerPub,
		AllowedIPs:       h.AllowedIPs,
		IP:               qUser.IP,
	}

	err = h.DataBase.RegisterUser(&user)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	h.Resources.Logger.Info().Msg("user: " + user.UserName + " with id: " + strconv.FormatInt(user.ID, 10) + " registered")

	_, err = h.Tg.Edit(c.Message(), c.Message().Text+"\nПользователь добавлен")
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	_, err = h.Tg.Send(tele.ChatID(user.ID), "Регистрация завершена, далее требуется только ввод двухфакторного кода")
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	err = h.DataBase.GetUsersIDs(h.Resources.UserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	err = h.DataBase.GetQueueUsersIDs(h.Resources.QUserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("accept")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Пользователь: " + user.UserName + " добавлен", ShowAlert: false})
}

func (h HighWay) RegisterDeny(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Error().Msg("deny: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if !funk.ContainsInt64(*h.Resources.QUserDBIDs, id) {
		return c.Respond(&tele.CallbackResponse{Text: "Пользователь не существует в списке на регистрацию"})
	}

	qUser, err := h.DataBase.GetQueueUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: fmt.Errorf("accept: %w \n", err).Error()})
	}

	err = h.DataBase.UnRegisterQUser(&qUser)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	h.Resources.Logger.Info().Msg("queue user: " + qUser.UserName + " with id: " + strconv.FormatInt(qUser.ID, 10) + " unregistered")

	_, err = h.Tg.Edit(c.Message(), c.Message().Text+"\nПользователь отклонен")
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	err = h.DataBase.GetQueueUsersIDs(h.Resources.QUserDBIDs)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("deny")
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}
	return c.Respond(&tele.CallbackResponse{Text: "Пользователь: " + qUser.UserName + " отклонен", ShowAlert: false})
}

func (h HighWay) StopSession(c tele.Context) error {
	if !slices.Contains(*h.Resources.AdminDBIDs, c.Sender().ID) {
		h.Resources.Logger.Error().Msg("stop_session: non admin user tried to use accept user")
		return c.Respond(&tele.CallbackResponse{Text: "Not admin"})
	}

	id, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неудалось обработать ID пользователя"})
	}

	if !h.Resources.SessionManager[id] {
		_, err := h.Tg.Edit(c.Message(), strings.ReplaceAll(c.Message().Text, ".", "\\.")+"\nСессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
		if err != nil {
			h.Resources.Logger.Error().Err(err).Msg("stop_session: failed to edit session message")
		}
		return c.Respond(&tele.CallbackResponse{Text: "Сессия не существует"})
	}

	user, err := h.DataBase.GetUser(&id)
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("stop_session: failed to get user")
		return c.Respond(&tele.CallbackResponse{Text: "db: Не удалось получить профиль пользователя"})
	}

	err = h.WGManager.WgStopSession(&user, h.Resources.MessageManager[id])
	if err != nil {
		h.Resources.Logger.Error().Err(err).Msg("stop_session: failed to stop session from button")
		return c.Respond(&tele.CallbackResponse{Text: "wg: Не удалось остановить сессию"})
	}

	return c.Respond(&tele.CallbackResponse{Text: "Сессия пользователя " + user.UserName + " остановлена", ShowAlert: false})
}
