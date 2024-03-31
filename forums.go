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
	rows, err := s.db.Select(goqu.Star()).
		From(schema.ForumsTopicsTable).
		Join(
			schema.CommentTopicSubscribeTable,
			goqu.On(schema.ForumsTopicsTableColID.Eq(schema.CommentTopicSubscribeTableColItemID)),
		).
		Join(
			schema.CommentTopicTable,
			goqu.On(schema.ForumsTopicsTableColID.Eq(schema.CommentTopicTableColItemID)),
		).
		Where(
			schema.CommentTopicSubscribeTableColUserID.Eq(userID),
			schema.CommentTopicTableColTypeID.Eq(comments.TypeIDForums),
			schema.CommentTopicSubscribeTableColTypeID.Eq(comments.TypeIDForums),
		).Executor().QueryContext(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	defer util.Close(rows)

	result := 0
	if rows.Next() {
		err = rows.Scan(&result)
		if err != nil {
			return 0, err
		}
	}

	if err = rows.Err(); err != nil {
		return 0, err
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

	success, err := s.db.Select("disable_topics").From(schema.ForumsThemesTable).Where(goqu.C("id").Eq(themeID)).
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

	res, err := s.db.Insert(schema.ForumsTopicsTableName).
		Cols("theme_id", "name", "author_id", "author_ip", "add_datetime", "views", "status").
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
			goqu.On(schema.ForumsTopicsTableColThemeID.Eq(schema.ForumsThemeParentTableColForumThemeID)),
		).
		Where(
			schema.ForumsThemeParentTableColParentID.Eq(schema.ForumsThemesTableColID),
			schema.ForumsTopicsTableColStatus.In([]string{TopicStatusNormal, TopicStatusClosed}),
		)

	messagesSelect := topicsSelect.
		Join(
			schema.CommentMessageTable,
			goqu.On(schema.ForumsTopicsTableColID.Eq(schema.CommentMessageTableColItemID)),
		).
		Where(schema.CommentMessageTableColTypeID.Eq(comments.TypeIDForums))

	_, err := s.db.Update(schema.ForumsThemesTable).Set(goqu.Record{
		"topics":   topicsSelect,
		"messages": messagesSelect,
	}).
		Where(schema.ForumsThemesTableColID.Eq(themeID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Forums) setStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.Update(schema.ForumsTopicsTableName).
		Set(goqu.Record{"status": status}).
		Where(goqu.C("id").Eq(id)).
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

	err := s.db.QueryRowContext(
		ctx,
		`SELECT theme_id FROM `+schema.ForumsTopicsTableName+` WHERE id = ?`,
		id,
	).Scan(&themeID)
	if err != nil {
		return err
	}

	var needAttention bool

	err = s.db.QueryRowContext(
		ctx,
		`
			SELECT 1 FROM `+schema.CommentMessageTableName+` 
			WHERE item_id = ? AND type_id = ? AND `+schema.CommentMessageTableModeratorAttentionColName+` = ? LIMIT 1
		`,
		id, comments.TypeIDForums, comments.ModeratorAttentionRequired,
	).Scan(&needAttention)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		needAttention = false
	}

	if err != nil {
		return err
	}

	if needAttention {
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

	err := s.db.QueryRowContext(
		ctx,
		`SELECT theme_id FROM `+schema.ForumsTopicsTableName+` WHERE id = ?`,
		id,
	).Scan(&oldThemeID)
	if err != nil {
		return err
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
	sqSelect := s.db.Select("id", "name", "topics", "messages", "disable_topics", "description").
		From(schema.ForumsThemesTable).Where(goqu.I("id").Eq(themeID))

	if !isModerator {
		sqSelect = sqSelect.Where(goqu.L("NOT is_moderator"))
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
	sqSelect := s.db.Select("id", "name", "topics", "messages", "disable_topics", "description").
		From(schema.ForumsThemesTable).Order(goqu.I("position").Asc())

	if themeID > 0 {
		sqSelect = sqSelect.Where(goqu.I("parent_id").Eq(themeID))
	} else {
		sqSelect = sqSelect.Where(goqu.I("parent_id").IsNull())
	}

	if !isModerator {
		sqSelect = sqSelect.Where(goqu.L("NOT is_moderator"))
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
		schema.ForumsTopicsTableColID, schema.ForumsTopicsTableColName, schema.ForumsTopicsTableColStatus,
		schema.ForumsTopicsTableColAddDatetime, schema.ForumsTopicsTableColAuthorID, schema.ForumsTopicsTableColThemeID,
	).
		From(schema.ForumsTopicsTable).
		Join(schema.CommentTopicTable, goqu.On(
			schema.ForumsTopicsTableColID.Eq(schema.CommentTopicTableColItemID),
			schema.CommentTopicTableColTypeID.Eq(comments.TypeIDForums),
		)).
		Where(schema.ForumsTopicsTableColStatus.In([]string{TopicStatusNormal, TopicStatusClosed}))

	if !isModerator {
		sqSelect = sqSelect.
			Join(schema.ForumsThemesTable, goqu.On(schema.ForumsTopicsTableColThemeID.Eq(schema.ForumsThemesTableColID))).
			Where(goqu.L("NOT " + schema.ForumsThemesTableName + ".is_moderator"))
	}

	return sqSelect
}

func (s *Forums) Topic(ctx context.Context, topicID int64, userID int64, isModerator bool) (*ForumsTopic, error) {
	sqSelect := s.topicsSelect(isModerator).
		Where(schema.ForumsTopicsTableColID.Eq(topicID)).
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
			goqu.On(schema.ForumsTopicsTableColThemeID.Eq(schema.ForumsThemeParentTableColForumThemeID)),
		).
		Where(schema.ForumsThemeParentTableColParentID.Eq(themeID)).
		Order(schema.CommentTopicTableColLastUpdate.Desc()).
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
		schema.CommentMessageTableColID, schema.CommentMessageTableColDatetime, schema.CommentMessageTableColAuthorID,
	).
		From(schema.CommentMessageTable).
		Join(
			schema.ForumsTopicsTable,
			goqu.On(schema.CommentMessageTableColItemID.Eq(schema.ForumsTopicsTableColID)),
		).
		Join(
			schema.ForumsThemeParentTable,
			goqu.On(schema.ForumsTopicsTableColThemeID.Eq(schema.ForumsThemeParentTableColForumThemeID)),
		).
		Where(
			schema.ForumsTopicsTableColStatus.In([]string{TopicStatusNormal, TopicStatusClosed}),
			schema.ForumsTopicsTableColID.Eq(topicID),
			schema.CommentMessageTableColTypeID.Eq(comments.TypeIDForums),
		).
		Order(schema.CommentMessageTableColDatetime.Desc()).
		Limit(1)

	if !isModerator {
		sqSelect = sqSelect.
			Join(
				schema.ForumsThemesTable,
				goqu.On(schema.ForumsThemeParentTableColParentID.Eq(schema.ForumsThemesTableColID)),
			).
			Where(goqu.L("NOT " + schema.ForumsThemesTableName + ".is_moderator"))
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
		Order(schema.CommentTopicTableColLastUpdate.Desc())

	if themeID > 0 {
		sqSelect = sqSelect.Where(schema.ForumsTopicsTableColThemeID.Eq(themeID))
	}

	if subscription {
		sqSelect = sqSelect.Join(
			schema.CommentTopicSubscribeTable,
			goqu.On(
				schema.ForumsTopicsTableColID.Eq(schema.CommentTopicSubscribeTableColItemID),
				schema.CommentTopicSubscribeTableColTypeID.Eq(comments.TypeIDForums),
			),
		).
			Where(schema.CommentTopicSubscribeTableColUserID.Eq(userID))
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
