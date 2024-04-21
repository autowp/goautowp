package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

const (
	TopicStatusNormal  = "normal"
	TopicStatusClosed  = "closed"
	TopicStatusDeleted = "deleted"
)

type ForumsTheme struct {
	ID            int64  `db:"id"`
	Name          string `db:"name"`
	TopicsCount   int32  `db:"topics"`
	MessagesCount int32  `db:"messages"`
	DisableTopics bool   `db:"disable_topics"`
	Description   string `db:"description"`
}

type ForumsTopic struct {
	ID           int64  `db:"id"`
	Name         string `db:"name"`
	Status       string `db:"status"`
	Messages     int32
	NewMessages  int32
	CreatedAt    time.Time `db:"add_datetime"`
	UserID       int64     `db:"author_id"`
	ThemeID      int64     `db:"theme_id"`
	Subscription bool
}

type CommentMessage struct {
	ID       int64         `db:"id"`
	Datetime time.Time     `db:"datetime"`
	UserID   sql.NullInt64 `db:"author_id"`
}

// Forums Main Object.
type Forums struct {
	db                 *goqu.Database
	commentsRepository *comments.Repository
}

func NewForums(db *goqu.Database, commentsRepository *comments.Repository) *Forums {
	return &Forums{
		db:                 db,
		commentsRepository: commentsRepository,
	}
}

func (s *Forums) GetUserSummary(ctx context.Context, userID int64) (int, error) {
	result := 0

	success, err := s.db.Select(goqu.COUNT(goqu.Star())).
		From(schema.ForumsTopicsTable).
		Join(
			schema.CommentTopicSubscribeTable,
			goqu.On(schema.ForumsTopicsTableIDCol.Eq(schema.CommentTopicSubscribeTableItemIDCol)),
		).
		Join(
			schema.CommentTopicTable,
			goqu.On(schema.ForumsTopicsTableIDCol.Eq(schema.CommentTopicTableItemIDCol)),
		).
		Where(
			schema.CommentTopicSubscribeTableUserIDCol.Eq(userID),
			schema.CommentTopicTableTypeIDCol.Eq(comments.TypeIDForums),
			schema.CommentTopicSubscribeTableTypeIDCol.Eq(comments.TypeIDForums),
		).ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
}

func (s *Forums) AddTopic(
	ctx context.Context,
	themeID int64,
	name string,
	userID int64,
	remoteAddr string,
) (int64, error) {
	var disableTopics bool

	success, err := s.db.Select(schema.ForumsThemesTableDisableTopicsCol).
		From(schema.ForumsThemesTable).
		Where(schema.ForumsThemesTableIDCol.Eq(themeID)).
		ScanValContext(ctx, &disableTopics)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	if disableTopics {
		return 0, errors.New("topics in this theme is disabled")
	}

	res, err := s.db.Insert(schema.ForumsTopicsTable).
		Cols(schema.ForumsTopicsTableThemeIDCol, schema.ForumsTopicsTableNameCol, schema.ForumsTopicsTableAuthorIDCol,
			schema.ForumsTopicsTableAuthorIPCol, schema.ForumsTopicsTableAddDatetimeCol,
			schema.ForumsTopicsTableViewsCol, schema.ForumsTopicsTableStatusCol).
		Vals(goqu.Vals{
			themeID,
			name,
			userID,
			goqu.Func("INET6_ATON", remoteAddr),
			goqu.Func("NOW"),
			0,
			TopicStatusNormal,
		}).Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	topicID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	err = s.updateThemeStat(ctx, themeID)
	if err != nil {
		return 0, err
	}

	return topicID, nil
}

func (s *Forums) updateThemeStat(ctx context.Context, themeID int64) error {
	topicsSelect := s.db.Select(goqu.COUNT(goqu.Star())).
		From(schema.ForumsTopicsTable).
		Join(
			schema.ForumsThemeParentTable,
			goqu.On(schema.ForumsTopicsTableThemeIDCol.Eq(schema.ForumsThemeParentTableForumThemeIDCol)),
		).
		Where(
			schema.ForumsThemeParentTableParentIDCol.Eq(schema.ForumsThemesTableIDCol),
			schema.ForumsTopicsTableStatusCol.In([]string{TopicStatusNormal, TopicStatusClosed}),
		)

	messagesSelect := topicsSelect.
		Join(
			schema.CommentMessageTable,
			goqu.On(schema.ForumsTopicsTableIDCol.Eq(schema.CommentMessageTableItemIDCol)),
		).
		Where(schema.CommentMessageTableTypeIDCol.Eq(comments.TypeIDForums))

	_, err := s.db.Update(schema.ForumsThemesTable).Set(goqu.Record{
		schema.ForumsThemesTableTopicsColName:   topicsSelect,
		schema.ForumsThemesTableMessagesColName: messagesSelect,
	}).
		Where(schema.ForumsThemesTableIDCol.Eq(themeID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Forums) setStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.Update(schema.ForumsTopicsTable).
		Set(goqu.Record{schema.ForumsTopicsTableStatusColName: status}).
		Where(schema.ForumsTopicsTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Forums) Close(ctx context.Context, id int64) error {
	return s.setStatus(ctx, id, TopicStatusClosed)
}

func (s *Forums) Open(ctx context.Context, id int64) error {
	return s.setStatus(ctx, id, TopicStatusNormal)
}

func (s *Forums) Delete(ctx context.Context, id int64) error {
	var themeID int64

	success, err := s.db.Select(schema.ForumsTopicsTableThemeIDCol).
		From(schema.ForumsTopicsTable).
		Where(schema.ForumsTopicsTableIDCol.Eq(id)).
		ScanValContext(ctx, &themeID)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	var needAttention bool

	success, err = s.db.Select(goqu.L("1")).
		From(schema.CommentMessageTable).
		Where(
			schema.CommentMessageTableItemIDCol.Eq(id),
			schema.CommentMessageTableTypeIDCol.Eq(comments.TypeIDForums),
			schema.CommentMessageTableModeratorAttentionCol.Eq(comments.ModeratorAttentionRequired),
		).ScanValContext(ctx, &needAttention)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("cannot delete topic with moderator attention requirement")
	}

	err = s.setStatus(ctx, id, TopicStatusDeleted)
	if err != nil {
		return err
	}

	return s.updateThemeStat(ctx, themeID)
}

func (s *Forums) MoveTopic(ctx context.Context, id int64, themeID int64) error {
	var oldThemeID int64

	success, err := s.db.Select(schema.ForumsTopicsTableThemeIDCol).
		From(schema.ForumsTopicsTable).
		Where(schema.ForumsTopicsTableIDCol.Eq(id)).
		ScanValContext(ctx, &oldThemeID)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE "+schema.ForumsTopicsTableName+" SET theme_id = ? WHERE id = ?",
		themeID, id,
	)
	if err != nil {
		return err
	}

	err = s.updateThemeStat(ctx, themeID)
	if err != nil {
		return err
	}

	return s.updateThemeStat(ctx, oldThemeID)
}

func (s *Forums) Theme(ctx context.Context, themeID int64, isModerator bool) (*ForumsTheme, error) {
	sqSelect := s.db.Select(
		schema.ForumsThemesTableIDCol, schema.ForumsThemesTableNameCol, schema.ForumsThemesTableTopicsCol,
		schema.ForumsThemesTableMessagesCol, schema.ForumsThemesTableDisableTopicsCol,
		schema.ForumsThemesTableDescriptionCol).
		From(schema.ForumsThemesTable).
		Where(schema.ForumsThemesTableIDCol.Eq(themeID))

	if !isModerator {
		sqSelect = sqSelect.Where(schema.ForumsThemesTableIsModeratorCol.IsFalse())
	}

	var row ForumsTheme

	success, err := sqSelect.ScanStructContext(ctx, &row)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint:nilnil
	}

	return &row, nil
}

func (s *Forums) Themes(ctx context.Context, themeID int64, isModerator bool) ([]*ForumsTheme, error) {
	sqSelect := s.db.Select(
		schema.ForumsThemesTableIDCol, schema.ForumsThemesTableNameCol, schema.ForumsThemesTableTopicsCol,
		schema.ForumsThemesTableMessagesCol, schema.ForumsThemesTableDisableTopicsCol,
		schema.ForumsThemesTableDescriptionCol).
		From(schema.ForumsThemesTable).Order(schema.ForumsThemesTablePositionCol.Asc())

	if themeID > 0 {
		sqSelect = sqSelect.Where(schema.ForumsThemesTableParentIDCol.Eq(themeID))
	} else {
		sqSelect = sqSelect.Where(schema.ForumsThemesTableParentIDCol.IsNull())
	}

	if !isModerator {
		sqSelect = sqSelect.Where(schema.ForumsThemesTableIsModeratorCol.IsFalse())
	}

	rows := make([]*ForumsTheme, 0)

	err := sqSelect.ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (s *Forums) prepareTopic(ctx context.Context, topic *ForumsTopic, userID int64) error {
	if userID > 0 {
		messages, newMessages, err := s.commentsRepository.TopicStatForUser(ctx, comments.TypeIDForums, topic.ID, userID)
		if err != nil {
			return err
		}

		topic.Messages = messages
		topic.NewMessages = newMessages

		topic.Subscription, err = s.commentsRepository.IsSubscribed(ctx, userID, comments.TypeIDForums, topic.ID)
		if err != nil {
			return err
		}

		return nil
	}

	messages, err := s.commentsRepository.TopicStat(ctx, comments.TypeIDForums, topic.ID)
	if err != nil {
		return err
	}

	topic.Messages = messages

	return nil
}

func (s *Forums) topicsSelect(isModerator bool) *goqu.SelectDataset {
	sqSelect := s.db.Select(
		schema.ForumsTopicsTableIDCol, schema.ForumsTopicsTableNameCol, schema.ForumsTopicsTableStatusCol,
		schema.ForumsTopicsTableAddDatetimeCol, schema.ForumsTopicsTableAuthorIDCol, schema.ForumsTopicsTableThemeIDCol,
	).
		From(schema.ForumsTopicsTable).
		Join(schema.CommentTopicTable, goqu.On(
			schema.ForumsTopicsTableIDCol.Eq(schema.CommentTopicTableItemIDCol),
			schema.CommentTopicTableTypeIDCol.Eq(comments.TypeIDForums),
		)).
		Where(schema.ForumsTopicsTableStatusCol.In([]string{TopicStatusNormal, TopicStatusClosed}))

	if !isModerator {
		sqSelect = sqSelect.
			Join(schema.ForumsThemesTable, goqu.On(schema.ForumsTopicsTableThemeIDCol.Eq(schema.ForumsThemesTableIDCol))).
			Where(schema.ForumsThemesTableIsModeratorCol.IsFalse())
	}

	return sqSelect
}

func (s *Forums) Topic(ctx context.Context, topicID int64, userID int64, isModerator bool) (*ForumsTopic, error) {
	sqSelect := s.topicsSelect(isModerator).
		Where(schema.ForumsTopicsTableIDCol.Eq(topicID)).
		Limit(1)

	topic := ForumsTopic{}

	success, err := sqSelect.ScanStructContext(ctx, &topic)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint:nilnil
	}

	err = s.prepareTopic(ctx, &topic, userID)
	if err != nil {
		return nil, err
	}

	return &topic, nil
}

func (s *Forums) LastTopic(ctx context.Context, themeID int64, userID int64, isModerator bool) (*ForumsTopic, error) {
	sqSelect := s.topicsSelect(isModerator).
		Join(
			schema.ForumsThemeParentTable,
			goqu.On(schema.ForumsTopicsTableThemeIDCol.Eq(schema.ForumsThemeParentTableForumThemeIDCol)),
		).
		Where(schema.ForumsThemeParentTableParentIDCol.Eq(themeID)).
		Order(schema.CommentTopicTableLastUpdateCol.Desc()).
		Limit(1)

	topic := ForumsTopic{}

	success, err := sqSelect.ScanStructContext(ctx, &topic)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint:nilnil
	}

	err = s.prepareTopic(ctx, &topic, userID)
	if err != nil {
		return nil, err
	}

	return &topic, nil
}

func (s *Forums) LastMessage(ctx context.Context, topicID int64, isModerator bool) (*CommentMessage, error) {
	sqSelect := s.db.Select(
		schema.CommentMessageTableIDCol, schema.CommentMessageTableDatetimeCol, schema.CommentMessageTableAuthorIDCol,
	).
		From(schema.CommentMessageTable).
		Join(
			schema.ForumsTopicsTable,
			goqu.On(schema.CommentMessageTableItemIDCol.Eq(schema.ForumsTopicsTableIDCol)),
		).
		Join(
			schema.ForumsThemeParentTable,
			goqu.On(schema.ForumsTopicsTableThemeIDCol.Eq(schema.ForumsThemeParentTableForumThemeIDCol)),
		).
		Where(
			schema.ForumsTopicsTableStatusCol.In([]string{TopicStatusNormal, TopicStatusClosed}),
			schema.ForumsTopicsTableIDCol.Eq(topicID),
			schema.CommentMessageTableTypeIDCol.Eq(comments.TypeIDForums),
		).
		Order(schema.CommentMessageTableDatetimeCol.Desc()).
		Limit(1)

	if !isModerator {
		sqSelect = sqSelect.
			Join(
				schema.ForumsThemesTable,
				goqu.On(schema.ForumsThemeParentTableParentIDCol.Eq(schema.ForumsThemesTableIDCol)),
			).
			Where(schema.ForumsThemesTableIsModeratorCol.IsFalse())
	}

	cm := CommentMessage{}

	success, err := sqSelect.ScanStructContext(ctx, &cm)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint:nilnil
	}

	return &cm, nil
}

func (s *Forums) Topics(
	ctx context.Context,
	themeID int64, userID int64, isModerator bool, subscription bool, page int32,
) ([]*ForumsTopic, *util.Pages, error) {
	sqSelect := s.topicsSelect(isModerator).
		Order(schema.CommentTopicTableLastUpdateCol.Desc())

	if themeID > 0 {
		sqSelect = sqSelect.Where(schema.ForumsTopicsTableThemeIDCol.Eq(themeID))
	}

	if subscription {
		sqSelect = sqSelect.Join(
			schema.CommentTopicSubscribeTable,
			goqu.On(
				schema.ForumsTopicsTableIDCol.Eq(schema.CommentTopicSubscribeTableItemIDCol),
				schema.CommentTopicSubscribeTableTypeIDCol.Eq(comments.TypeIDForums),
			),
		).
			Where(schema.CommentTopicSubscribeTableUserIDCol.Eq(userID))
	}

	rows := make([]*ForumsTopic, 0)

	paginator := util.Paginator{
		SQLSelect:         sqSelect,
		CurrentPageNumber: page,
	}

	sqSelect, err := paginator.GetCurrentItems(ctx)
	if err != nil {
		return nil, nil, err
	}

	err = sqSelect.ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, nil, err
	}

	for _, row := range rows {
		err = s.prepareTopic(ctx, row, userID)
		if err != nil {
			return nil, nil, err
		}
	}

	pages, err := paginator.GetPages(ctx)
	if err != nil {
		return nil, nil, err
	}

	return rows, pages, nil
}
