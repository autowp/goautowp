package messaging

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

type Options struct {
	AllMessagesLink bool
}

const (
	MaxText         = 2000
	MessagesPerPage = 20
)

type Repository struct {
	db              *goqu.Database
	telegramService *telegram.Service
}

type messageRow struct {
	ID          int64         `db:"id"`
	FromUserID  sql.NullInt64 `db:"from_user_id"`
	ToUserID    int64         `db:"to_user_id"`
	Readen      bool          `db:"readen"`
	Contents    string        `db:"contents"`
	AddDatetime time.Time     `db:"add_datetime"`
}

type Message struct {
	ID               int64
	AuthorID         *int64
	Text             string
	IsNew            bool
	CanDelete        bool
	Date             time.Time
	CanReply         bool
	DialogCount      int32
	AllMessagesLink  bool
	ToUserID         int64
	DialogWithUserID int64
}

func NewRepository(db *goqu.Database, telegramService *telegram.Service) *Repository {
	return &Repository{
		db:              db,
		telegramService: telegramService,
	}
}

func (s *Repository) GetUserNewMessagesCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getReceivedSelect(userID).Where(schema.PersonalMessagesTableReadenCol.IsNotTrue()),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetInboxCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getInboxSelect(userID),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetInboxNewCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getInboxSelect(userID).Where(schema.PersonalMessagesTableReadenCol.IsNotTrue()),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetSentCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getSentSelect(userID),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetSystemCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getSystemSelect(userID),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetSystemNewCount(ctx context.Context, userID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getSystemSelect(userID).Where(schema.PersonalMessagesTableReadenCol.IsNotTrue()),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) GetDialogCount(ctx context.Context, userID int64, withUserID int64) (int32, error) {
	paginator := util.Paginator{
		SQLSelect: s.getDialogSelect(userID, withUserID),
	}

	return paginator.GetTotalItemCount(ctx)
}

func (s *Repository) DeleteMessage(ctx context.Context, userID int64, messageID int64) error {
	_, err := s.db.Update(schema.PersonalMessagesTable).
		Set(goqu.Record{schema.PersonalMessagesTableDeletedByFromColName: 1}).
		Where(
			schema.PersonalMessagesTableFromUserIDCol.Eq(userID),
			schema.PersonalMessagesTableIDCol.Eq(messageID),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	_, err = s.db.Update(schema.PersonalMessagesTable).
		Set(goqu.Record{schema.PersonalMessagesTableDeletedByToColName: 1}).
		Where(
			schema.PersonalMessagesTableToUserIDCol.Eq(userID),
			schema.PersonalMessagesTableIDCol.Eq(messageID),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) ClearSent(ctx context.Context, userID int64) error {
	_, err := s.db.Update(schema.PersonalMessagesTable).
		Set(goqu.Record{schema.PersonalMessagesTableDeletedByFromColName: 1}).
		Where(schema.PersonalMessagesTableFromUserIDCol.Eq(userID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) ClearSystem(ctx context.Context, userID int64) error {
	_, err := s.db.Delete(schema.PersonalMessagesTable).
		Where(
			schema.PersonalMessagesTableToUserIDCol.Eq(userID),
			schema.PersonalMessagesTableFromUserIDCol.IsNull(),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) CreateMessage(ctx context.Context, fromUserID int64, toUserID int64, text string) error {
	text = strings.TrimSpace(text)
	msgLength := len(text)

	if msgLength <= 0 {
		return errors.New("message is empty")
	}

	if msgLength > MaxText {
		return errors.New("too long message")
	}

	nullableFromUserID := sql.NullInt64{Int64: fromUserID, Valid: fromUserID != 0}

	_, err := s.db.Insert(schema.PersonalMessagesTable).Rows(
		goqu.Record{
			schema.PersonalMessagesTableFromUserIDColName:  nullableFromUserID,
			schema.PersonalMessagesTableToUserIDColName:    toUserID,
			schema.PersonalMessagesTableContentsColName:    text,
			schema.PersonalMessagesTableAddDatetimeColName: goqu.Func("NOW"),
			schema.PersonalMessagesTableReadenColName:      false,
		},
	).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	err = s.telegramService.NotifyMessage(ctx, fromUserID, toUserID, text)
	if err != nil {
		return err
	}

	return nil
}

func (s *Repository) markReaden(ids []int64) error {
	var err error
	if len(ids) > 0 {
		_, err = s.db.Update(schema.PersonalMessagesTable).
			Set(goqu.Record{schema.PersonalMessagesTableReadenColName: true}).
			Where(schema.PersonalMessagesTableIDCol.In(ids)).
			Executor().Exec()
	}

	return err
}

func (s *Repository) markReadenRows(rows []messageRow, userID int64) error {
	ids := make([]int64, 0)

	for _, msg := range rows {
		if (!msg.Readen) && (msg.ToUserID == userID) {
			ids = append(ids, msg.ID)
		}
	}

	return s.markReaden(ids)
}

func (s *Repository) getBox(
	ctx context.Context,
	userID int64,
	paginator util.Paginator,
	options Options,
) ([]Message, *util.Pages, error) {
	ds, err := paginator.GetCurrentItems(ctx)
	if err != nil {
		return nil, nil, err
	}

	var msgs []messageRow
	err = ds.ScanStructsContext(ctx, &msgs)

	if err != nil {
		return nil, nil, err
	}

	if userID > 0 {
		err = s.markReadenRows(msgs, userID)
		if err != nil {
			return nil, nil, err
		}
	}

	pages, err := paginator.GetPages(ctx)
	if err != nil {
		return nil, nil, err
	}

	list, err := s.prepareList(ctx, userID, msgs, options)
	if err != nil {
		return nil, nil, err
	}

	return list, pages, nil
}

func (s *Repository) GetInbox(ctx context.Context, userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SQLSelect:         s.getInboxSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	return s.getBox(ctx, userID, paginator, Options{AllMessagesLink: true})
}

func (s *Repository) GetSentbox(ctx context.Context, userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SQLSelect:         s.getSentSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	return s.getBox(ctx, userID, paginator, Options{AllMessagesLink: true})
}

func (s *Repository) GetSystembox(ctx context.Context, userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SQLSelect:         s.getSystemSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	return s.getBox(ctx, userID, paginator, Options{AllMessagesLink: false})
}

func (s *Repository) GetDialogbox(
	ctx context.Context,
	userID int64,
	withUserID int64,
	page int32,
) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SQLSelect:         s.getDialogSelect(userID, withUserID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	return s.getBox(ctx, userID, paginator, Options{AllMessagesLink: false})
}

func (s *Repository) getReceivedSelect(userID int64) *goqu.SelectDataset {
	return s.db.From(schema.PersonalMessagesTable).
		Where(
			schema.PersonalMessagesTableToUserIDCol.Eq(userID),
			schema.PersonalMessagesTableDeletedByToCol.IsFalse(),
		).
		Order(schema.PersonalMessagesTableAddDatetimeCol.Desc())
}

func (s *Repository) getSystemSelect(userID int64) *goqu.SelectDataset {
	return s.getReceivedSelect(userID).Where(schema.PersonalMessagesTableFromUserIDCol.IsNull())
}

func (s *Repository) getInboxSelect(userID int64) *goqu.SelectDataset {
	return s.getReceivedSelect(userID).Where(schema.PersonalMessagesTableFromUserIDCol.IsNotNull())
}

func (s *Repository) getSentSelect(userID int64) *goqu.SelectDataset {
	return s.db.From(schema.PersonalMessagesTable).
		Where(
			schema.PersonalMessagesTableFromUserIDCol.Eq(userID),
			schema.PersonalMessagesTableDeletedByFromCol.IsNotTrue(),
		).
		Order(schema.PersonalMessagesTableAddDatetimeCol.Desc())
}

func (s *Repository) getDialogSelect(userID int64, withUserID int64) *goqu.SelectDataset {
	return s.db.From(schema.PersonalMessagesTable).
		Where(
			goqu.Or(
				goqu.And(
					schema.PersonalMessagesTableFromUserIDCol.Eq(userID),
					schema.PersonalMessagesTableToUserIDCol.Eq(withUserID),
					schema.PersonalMessagesTableDeletedByFromCol.IsNotTrue(),
				),
				goqu.And(
					schema.PersonalMessagesTableFromUserIDCol.Eq(withUserID),
					schema.PersonalMessagesTableToUserIDCol.Eq(userID),
					schema.PersonalMessagesTableDeletedByToCol.IsNotTrue(),
				),
			),
		).
		Order(schema.PersonalMessagesTableAddDatetimeCol.Desc())
}

func (s *Repository) prepareList(
	ctx context.Context,
	userID int64,
	rows []messageRow,
	options Options,
) ([]Message, error) {
	var err error

	cache := make(map[int64]int32)

	messages := make([]Message, len(rows))

	for idx, msg := range rows {
		isNew := msg.ToUserID == userID && !msg.Readen
		canDelete := msg.FromUserID.Valid && msg.FromUserID.Int64 == userID || msg.ToUserID == userID
		authorIsMe := msg.FromUserID.Valid && msg.FromUserID.Int64 == userID
		canReply := msg.FromUserID.Valid && !authorIsMe //  && ! $author['deleted']

		var dialogWithUserID int64

		if msg.ToUserID == userID {
			if msg.FromUserID.Valid {
				dialogWithUserID = msg.FromUserID.Int64
			}
		} else {
			dialogWithUserID = msg.ToUserID
		}

		var dialogCount int32

		if options.AllMessagesLink && dialogWithUserID != 0 {
			var (
				ok bool
				id = dialogWithUserID
			)

			if dialogCount, ok = cache[id]; !ok {
				dialogCount, err = s.GetDialogCount(ctx, userID, id)
				if err != nil {
					return messages, err
				}

				cache[id] = dialogCount
			}
		}

		messages[idx] = Message{
			ID:               msg.ID,
			AuthorID:         util.SQLNullInt64ToPtr(msg.FromUserID),
			Text:             msg.Contents,
			IsNew:            isNew,
			CanDelete:        canDelete,
			Date:             msg.AddDatetime,
			CanReply:         canReply,
			DialogCount:      dialogCount,
			AllMessagesLink:  options.AllMessagesLink,
			ToUserID:         msg.ToUserID,
			DialogWithUserID: dialogWithUserID,
		}
	}

	return messages, nil
}
