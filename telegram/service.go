package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/url"
	"strings"
)

type Service struct {
	accessToken  string
	db           *sql.DB
	hostsManager *hosts.Manager
	botAPI       *tgbotapi.BotAPI
}

func NewService(config config.TelegramConfig, db *sql.DB, hostsManager *hosts.Manager) *Service {
	return &Service{
		accessToken:  config.AccessToken,
		db:           db,
		hostsManager: hostsManager,
	}
}

func (s *Service) getBotAPI() (*tgbotapi.BotAPI, error) {
	if s.botAPI == nil {
		bot, err := tgbotapi.NewBotAPI(s.accessToken)
		if err != nil {
			return nil, err
		}
		s.botAPI = bot
	}

	return s.botAPI, nil
}

func (s *Service) NotifyMessage(ctx context.Context, fromId int64, userId int64, text string) error {
	fromName := "New personal message"

	if fromId > 0 {
		err := s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", fromId).Scan(&fromName)
		if err != nil {
			return err
		}
	}

	chatRows, err := s.db.QueryContext(ctx, "SELECT chat_id FROM telegram_chat WHERE user_id = ? AND messages", userId)
	if err != nil {
		return err
	}

	for chatRows.Next() {
		var chatID int64
		if err = chatRows.Scan(&chatID); err != nil {
			return err
		}

		uri, err := s.getUriByChatId(ctx, chatID)
		if err != nil {
			return err
		}
		uri.Path = "/account/messages"
		if fromId <= 0 {
			q := uri.Query()
			q.Add("folder", "system")
			uri.RawQuery = q.Encode()
		}

		telegramMessage := fmt.Sprintf(
			"%s: \n%s\n\n%s",
			fromName,
			text,
			uri.String(),
		)

		return s.sendMessage(ctx, telegramMessage, chatID)
	}

	return nil
}

func (s *Service) getUriByChatId(ctx context.Context, chatId int64) (*url.URL, error) {

	var userId int64
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM telegram_chat WHERE chat_id = ?", chatId).Scan(&userId)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if chatId > 0 && userId > 0 {
		language := ""
		err = s.db.QueryRowContext(ctx, "SELECT language FROM users WHERE id = ?", userId).Scan(&language)
		if err != nil {
			return nil, err
		}

		if len(language) > 0 {
			return s.hostsManager.GetUriByLanguage(language)
		}
	}

	return url.Parse("https://wheelsage.org")
}

func (s *Service) sendMessage(ctx context.Context, text string, chat int64) error {

	bot, err := s.getBotAPI()
	if err != nil {
		return err
	}

	if chat <= 0 {
		return errors.New("`chat_id` not provided")
	}

	mc := tgbotapi.NewMessage(chat, text)

	_, err = bot.Send(mc)

	if err != nil {

		if strings.Contains(err.Error(), "deactivated") {
			return s.unsubscribeChat(ctx, chat)
		}

		if strings.Contains(err.Error(), "blocked") {
			return s.unsubscribeChat(ctx, chat)
		}
	}

	return err
}

func (s *Service) unsubscribeChat(ctx context.Context, chatID int64) error {

	_, err := s.db.ExecContext(ctx, "DELETE FROM telegram_brand WHERE chat_id = ?", chatID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM telegram_chat WHERE chat_id = ?", chatID)

	return err
}
