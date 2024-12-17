package tg

import (
	"bot/concierge"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"strconv"
)

func (t Telegram) SessionStarted(u concierge.User) error {
	var err error
	t.Managers.MessageManager[u.ID], err = t.Tg.Send(tele.ChatID(t.Config.AdminWgChatID), "Создана сессия для: "+u.UserName, &tele.SendOptions{
		ThreadID: t.Config.AdminWgChatThread,
		ReplyMarkup: &tele.ReplyMarkup{
			OneTimeKeyboard: true,
			InlineKeyboard: [][]tele.InlineButton{{
				tele.InlineButton{
					Unique: "stop_session",
					Text:   "Stop",
					Data:   strconv.FormatInt(u.ID, 10)}}}}})
	if err != nil {
		return fmt.Errorf("tgmng: failed to send status message: %w", err)
	}
	return nil
}

func (t Telegram) SessionEnded(u concierge.User) error {
	_, err := t.Tg.Edit(t.Managers.MessageManager[u.ID], t.Managers.MessageManager[u.ID].Text+"Сессия завершена", &tele.SendOptions{ParseMode: "MarkdownV2"})
	if err != nil {
		_, err = t.Tg.Send(tele.ChatID(t.Config.AdminWgChatID), err.Error(), &tele.SendOptions{ReplyTo: t.Managers.MessageManager[u.ID], ThreadID: t.Managers.MessageManager[u.ID].ThreadID})
		if err != nil {
			return fmt.Errorf("tg: failed to edit message %w: %v \n", err, t.Managers.MessageManager[u.ID])
		}
		return fmt.Errorf("wg: tg: failed to edit message: %w", err)
	}
	_, err = t.Tg.Send(tele.ChatID(u.ID), "Сессия завершена")
	if err != nil {
		return fmt.Errorf("tg: failed to send session end message: %w", err)
	}
	return nil
}
