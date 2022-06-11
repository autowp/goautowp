package messaging

import (
	"context"
	"database/sql"
	"errors"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"strings"
	"time"
)

type Options struct {
	AllMessagesLink bool
}

const MaxText = 2000
const MessagesPerPage = 20

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
	DialogWithUserID *int64
}

func NewRepository(db *goqu.Database, telegramService *telegram.Service) *Repository {
	return &Repository{
		db:              db,
		telegramService: telegramService,
	}
}

func (s *Repository) GetUserNewMessagesCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getReceivedSelect(userID).Where(goqu.I("readen").IsNotTrue()),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetInboxCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getInboxSelect(userID),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetInboxNewCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getInboxSelect(userID).Where(goqu.I("readen").IsNotTrue()),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetSentCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getSentSelect(userID),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetSystemCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getSystemSelect(userID),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetSystemNewCount(userID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getSystemSelect(userID).Where(goqu.I("readen").IsNotTrue()),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) GetDialogCount(userID int64, withUserID int64) (int32, error) {
	paginator := util.Paginator{
		SqlSelect: s.getDialogSelect(userID, withUserID),
	}

	return paginator.GetTotalItemCount()
}

func (s *Repository) DeleteMessage(ctx context.Context, userID int64, messageID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_from = 1 WHERE from_user_id = ? AND id = ?
    `, userID, messageID)

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_to = 1 WHERE to_user_id = ? AND id = ?
    `, userID, messageID)

	return err
}

func (s *Repository) ClearSent(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_from = 1 WHERE from_user_id = ?
    `, userID)

	return err
}

func (s *Repository) ClearSystem(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM personal_messages WHERE to_user_id = ? AND from_user_id IS NULL
    `, userID)

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

	_, err := s.db.Insert("personal_messages").Rows(
		goqu.Record{
			"from_user_id": fromUserID,
			"to_user_id":   toUserID,
			"contents":     text,
			"add_datetime": goqu.L("NOW()"),
			"readen":       false,
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
		_, err = s.db.Update("personal_messages").
			Set(goqu.Record{"readen": true}).
			Where(
				goqu.I("id").In(ids),
			).
			Executor().Exec()
	}

	return err
}

func (s *Repository) markReadenRows(rows []messageRow, userId int64) error {
	ids := make([]int64, 0)

	for _, msg := range rows {
		if (!msg.Readen) && (msg.ToUserID == userId) {
			ids = append(ids, msg.ID)
		}
	}

	return s.markReaden(ids)
}

func (s *Repository) GetInbox(userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SqlSelect:         s.getInboxSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	ds, err := paginator.GetCurrentItems()
	if err != nil {
		return nil, nil, err
	}

	var msgs []messageRow
	err = ds.ScanStructs(&msgs)

	if err != nil {
		return nil, nil, err
	}

	err = s.markReadenRows(msgs, userID)
	if err != nil {
		return nil, nil, err
	}

	pages, err := paginator.GetPages()
	if err != nil {
		return nil, nil, err
	}

	list, err := s.prepareList(userID, msgs, Options{AllMessagesLink: true})
	if err != nil {
		return nil, nil, err
	}

	return list, pages, nil
}

func (s *Repository) GetSentbox(userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SqlSelect:         s.getSentSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	ds, err := paginator.GetCurrentItems()
	if err != nil {
		return nil, nil, err
	}

	var msgs []messageRow
	err = ds.ScanStructs(&msgs)

	if err != nil {
		return nil, nil, err
	}

	pages, err := paginator.GetPages()
	if err != nil {
		return nil, nil, err
	}

	list, err := s.prepareList(userID, msgs, Options{AllMessagesLink: true})
	if err != nil {
		return nil, nil, err
	}

	return list, pages, nil
}

func (s *Repository) GetSystembox(userID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SqlSelect:         s.getSystemSelect(userID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	ds, err := paginator.GetCurrentItems()
	if err != nil {
		return nil, nil, err
	}

	var msgs []messageRow
	err = ds.ScanStructs(&msgs)

	if err != nil {
		return nil, nil, err
	}

	err = s.markReadenRows(msgs, userID)
	if err != nil {
		return nil, nil, err
	}

	pages, err := paginator.GetPages()
	if err != nil {
		return nil, nil, err
	}

	list, err := s.prepareList(userID, msgs, Options{AllMessagesLink: true})
	if err != nil {
		return nil, nil, err
	}

	return list, pages, nil
}

func (s *Repository) GetDialogbox(userID int64, withUserID int64, page int32) ([]Message, *util.Pages, error) {
	paginator := util.Paginator{
		SqlSelect:         s.getDialogSelect(userID, withUserID),
		ItemCountPerPage:  MessagesPerPage,
		CurrentPageNumber: page,
	}

	ds, err := paginator.GetCurrentItems()
	if err != nil {
		return nil, nil, err
	}

	var msgs []messageRow
	err = ds.ScanStructs(&msgs)

	if err != nil {
		return nil, nil, err
	}

	err = s.markReadenRows(msgs, userID)
	if err != nil {
		return nil, nil, err
	}

	pages, err := paginator.GetPages()
	if err != nil {
		return nil, nil, err
	}

	list, err := s.prepareList(userID, msgs, Options{AllMessagesLink: false})
	if err != nil {
		return nil, nil, err
	}

	return list, pages, nil
}

func (s *Repository) getReceivedSelect(userID int64) *goqu.SelectDataset {
	return s.db.From("personal_messages").
		Where(
			goqu.I("to_user_id").Eq(userID),
			goqu.I("deleted_by_to").IsFalse(),
		).
		Order(goqu.I("add_datetime").Desc())
}

func (s *Repository) getSystemSelect(userID int64) *goqu.SelectDataset {
	return s.getReceivedSelect(userID).Where(goqu.I("from_user_id").IsNull())
}

func (s *Repository) getInboxSelect(userID int64) *goqu.SelectDataset {
	return s.getReceivedSelect(userID).Where(goqu.I("from_user_id").IsNotNull())
}

func (s *Repository) getSentSelect(userID int64) *goqu.SelectDataset {
	return s.db.From("personal_messages").
		Where(
			goqu.I("from_user_id").Eq(userID),
			goqu.I("deleted_by_from").IsNotTrue(),
		).
		Order(goqu.I("add_datetime").Desc())
}

func (s *Repository) getDialogSelect(userID int64, withUserID int64) *goqu.SelectDataset {

	return s.db.From("personal_messages").
		Where(
			goqu.Or(
				goqu.And(
					goqu.I("from_user_id").Eq(userID),
					goqu.I("to_user_id").Eq(withUserID),
					goqu.I("deleted_by_from").IsNotTrue(),
				),
				goqu.And(
					goqu.I("from_user_id").Eq(withUserID),
					goqu.I("to_user_id").Eq(userID),
					goqu.I("deleted_by_to").IsNotTrue(),
				),
			),
		).
		Order(goqu.I("add_datetime").Desc())
}

func (s *Repository) prepareList(userID int64, rows []messageRow, options Options) ([]Message, error) {
	var err error

	cache := make(map[int64]int32)

	messages := make([]Message, len(rows))

	for idx, msg := range rows {
		isNew := msg.ToUserID == userID && !msg.Readen
		canDelete := msg.FromUserID.Valid && msg.FromUserID.Int64 == userID || msg.ToUserID == userID
		authorIsMe := msg.FromUserID.Valid && msg.FromUserID.Int64 == userID
		canReply := msg.FromUserID.Valid && !authorIsMe //  && ! $author['deleted']

		var dialogCount int32

		var ok bool

		if options.AllMessagesLink && msg.FromUserID.Valid {
			dialogWith := msg.FromUserID.Int64
			if msg.FromUserID.Valid && msg.FromUserID.Int64 == userID {
				dialogWith = msg.ToUserID
			}

			if dialogCount, ok = cache[dialogWith]; !ok {
				dialogCount, err = s.GetDialogCount(userID, dialogWith)
				if err != nil {
					return messages, err
				}

				cache[dialogWith] = dialogCount
			}
		}

		var dialogWithUserID *int64

		if msg.ToUserID == userID {
			if msg.FromUserID.Valid {
				dialogWithUserID = &msg.FromUserID.Int64
			}
		} else {
			dialogWithUserID = &userID
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
