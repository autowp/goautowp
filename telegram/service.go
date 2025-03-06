package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/frontend"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var errChatIDNotProvide = errors.New("`chat_id` not provided")

type Service struct {
	accessToken  string
	db           *goqu.Database
	hostsManager *hosts.Manager
	botAPI       *tgbotapi.BotAPI
}

func NewService(
	config config.TelegramConfig, db *goqu.Database, hostsManager *hosts.Manager,
) *Service {
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
		success, err := s.db.Select(schema.UserTableNameCol).
			From(schema.UserTable).
			Where(schema.UserTableIDCol.Eq(fromID)).
			ScanValContext(ctx, &fromName)
		if err != nil {
			return err
		}

		if !success {
			return sql.ErrNoRows
		}
	}

	var chatIDs []int64

	err := s.db.Select(schema.TelegramChatTableChatIDCol).
		From(schema.TelegramChatTable).
		Where(
			schema.TelegramChatTableUserIDCol.Eq(userID),
			schema.TelegramChatTableMessagesCol.IsTrue(),
		).
		ScanValsContext(ctx, &chatIDs)
	if err != nil {
		return err
	}

	ctx = context.WithoutCancel(ctx)

	for _, chatID := range chatIDs {
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

func (s *Service) NotifyPicture(
	ctx context.Context, picture *schema.PictureRow, itemRepository *items.Repository,
) error {
	itemIDsSelect, err := itemRepository.IDsSelect(query.ItemListOptions{
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		ItemParentCacheDescendant: &query.ItemParentCacheListOptions{
			PictureItemsByItemID: &query.PictureItemListOptions{
				PictureID: picture.ID,
			},
		},
	})
	if err != nil {
		return err
	}

	var chatIDs []int64

	err = s.db.Select(schema.TelegramBrandTableChatIDCol).From(schema.TelegramBrandTable).Where(
		schema.TelegramBrandTableItemIDCol.In(itemIDsSelect),
		schema.TelegramBrandTableNewCol.IsTrue(),
		schema.TelegramBrandTableChatIDCol.NotIn(
			s.db.Select(schema.TelegramChatTableChatIDCol).
				From(schema.TelegramChatTable).
				Join(schema.PictureTable, goqu.On(schema.TelegramChatTableUserIDCol.Eq(schema.PictureTableOwnerIDCol))).
				Where(schema.PictureTableIDCol.Eq(picture.ID)),
		),
		schema.TelegramBrandTableChatIDCol.IsNotNull(),
	).ScanValsContext(ctx, &chatIDs)
	if err != nil {
		return err
	}

	ctx = context.WithoutCancel(ctx)

	for _, chatID := range chatIDs {
		uri, err := s.getURIByChatID(ctx, chatID)
		if err != nil {
			return err
		}

		err = s.sendMessage(ctx, frontend.PictureURL(uri, picture.Identity), chatID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) getURIByChatID(ctx context.Context, chatID int64) (*url.URL, error) {
	if chatID > 0 {
		language := ""

		success, err := s.db.Select(schema.UserTableLanguageCol).
			From(schema.UserTable).
			Join(schema.TelegramChatTable, goqu.On(schema.UserTableIDCol.Eq(schema.TelegramChatTableUserIDCol))).
			Where(schema.TelegramChatTableChatIDCol.Eq(chatID)).
			ScanValContext(ctx, &language)
		if err != nil {
			return nil, err
		}

		if success && len(language) > 0 {
			return s.hostsManager.URIByLanguage(language)
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
		return errChatIDNotProvide
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
	_, err := s.db.Delete(schema.TelegramBrandTable).
		Where(schema.TelegramBrandTableChatIDCol.Eq(chatID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	_, err = s.db.Delete(schema.TelegramChatTable).
		Where(schema.TelegramChatTableChatIDCol.Eq(chatID)).
		Executor().ExecContext(ctx)

	return err
}
