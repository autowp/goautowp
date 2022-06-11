package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	"github.com/doug-martin/goqu/v9"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/url"
	"strings"
)

type Service struct {
	accessToken  string
	db           *goqu.Database
	hostsManager *hosts.Manager
	botAPI       *tgbotapi.BotAPI
}

func NewService(config config.TelegramConfig, db *goqu.Database, hostsManager *hosts.Manager) *Service {
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

func (s *Service) NotifyMessage(ctx context.Context, fromID int64, userID int64, text string) error {
	fromName := "New personal message"

	if fromID > 0 {
		err := s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", fromID).Scan(&fromName)
		if err != nil {
			return err
		}
	}

	chatRows, err := s.db.QueryContext(ctx, "SELECT chat_id FROM telegram_chat WHERE user_id = ? AND messages", userID)
	if err != nil {
		return err
	}

	for chatRows.Next() {
		var chatID int64
		if err = chatRows.Scan(&chatID); err != nil {
			return err
		}

		uri, err := s.getURIByChatID(ctx, chatID)
		if err != nil {
			return err
		}

		uri.Path = "/account/messages"

		if fromID <= 0 {
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

		err = s.sendMessage(ctx, telegramMessage, chatID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) getURIByChatID(ctx context.Context, chatID int64) (*url.URL, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM telegram_chat WHERE chat_id = ?", chatID).Scan(&userID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if chatID > 0 && userID > 0 {
		language := ""
		err = s.db.QueryRowContext(ctx, "SELECT language FROM users WHERE id = ?", userID).Scan(&language)

		if err != nil {
			return nil, err
		}

		if len(language) > 0 {
			return s.hostsManager.GetURIByLanguage(language)
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
