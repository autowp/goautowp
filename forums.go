package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/autowp/goautowp/comments"
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
	rows, err := s.db.QueryContext(ctx, `
		SELECT count(1)
		FROM forums_topics
			JOIN comment_topic_subscribe ON forums_topics.id = comment_topic_subscribe.item_id
			JOIN comment_topic ON forums_topics.id = comment_topic.item_id
		WHERE comment_topic_subscribe.user_id = ?
		  	AND comment_topic.type_id = ?
			AND comment_topic_subscribe.type_id = ?
	`, userID, comments.TypeIDForums, comments.TypeIDForums)
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

	err := s.db.QueryRowContext(
		ctx,
		`SELECT disable_topics FROM forums_themes WHERE id = ?`,
		themeID,
	).Scan(&disableTopics)
	if err != nil {
		return 0, err
	}

	if disableTopics {
		return 0, errors.New("topics in this theme is disabled")
	}

	res, err := s.db.Insert("forums_topics").
		Cols("theme_id", "name", "author_id", "author_ip", "add_datetime", "views", "status").
		Vals(goqu.Vals{
			themeID,
			name,
			userID,
			goqu.L("INET6_ATON(?)", remoteAddr),
			goqu.L("NOW()"),
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
	_, err := s.db.ExecContext(
		ctx,
		`
			UPDATE forums_themes 
			SET topics = (
					SELECT COUNT(1)
					FROM forums_topics
						INNER JOIN forums_theme_parent ON forums_topics.theme_id = forums_theme_parent.forum_theme_id
					WHERE forums_theme_parent.parent_id = forums_themes.id
					  AND forums_topics.status IN (?, ?)
				),
				messages = (
				    SELECT COUNT(1)
				    FROM comment_message
				    	INNER JOIN forums_topics ON comment_message.item_id = forums_topics.id
				    	INNER JOIN forums_theme_parent ON forums_topics.theme_id = forums_theme_parent.forum_theme_id
				    WHERE comment_message.type_id = ? 
				      AND forums_theme_parent.parent_id = forums_themes.id
				      AND forums_topics.status IN (?, ?)
				)
			WHERE forums_themes.id = ?
		`,
		TopicStatusNormal, TopicStatusClosed,
		comments.TypeIDForums,
		TopicStatusNormal, TopicStatusClosed,
		themeID,
	)

	return err
}

func (s *Forums) Close(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE forums_topics SET status = ? WHERE id = ?",
		TopicStatusClosed, id,
	)

	return err
}

func (s *Forums) Open(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE forums_topics SET status = ? WHERE id = ?",
		TopicStatusNormal, id,
	)

	return err
}

func (s *Forums) Delete(ctx context.Context, id int64) error {
	var themeID int64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT theme_id FROM forums_topics WHERE id = ?`,
		id,
	).Scan(&themeID)
	if err != nil {
		return err
	}

	var needAttention bool

	err = s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM comment_message WHERE item_id = ? AND type_id = ? AND moderator_attention = ? LIMIT 1`,
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

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE forums_topics SET status = ? WHERE id = ?",
		TopicStatusDeleted, id,
	)
	if err != nil {
		return err
	}

	return s.updateThemeStat(ctx, themeID)
}

func (s *Forums) MoveTopic(ctx context.Context, id int64, themeID int64) error {
	var oldThemeID int64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT theme_id FROM forums_topics WHERE id = ?`,
		id,
	).Scan(&oldThemeID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE forums_topics SET theme_id = ? WHERE id = ?",
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
		From("forums_themes").Where(goqu.I("id").Eq(themeID))

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
		From("forums_themes").Order(goqu.I("position").Asc())

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
		"forums_topics.id", "forums_topics.name", "forums_topics.status", "forums_topics.add_datetime",
		"forums_topics.author_id", "forums_topics.theme_id",
	).
		From("forums_topics").
		Join(goqu.I("comment_topic"), goqu.On(
			goqu.I("forums_topics.id").Eq(goqu.I("comment_topic.item_id")),
			goqu.I("comment_topic.type_id").Eq(comments.TypeIDForums),
		)).
		Where(goqu.I("forums_topics.status").In([]string{TopicStatusNormal, TopicStatusClosed}))

	if !isModerator {
		sqSelect = sqSelect.
			Join(goqu.T("forums_themes"), goqu.On(goqu.I("forums_topics.theme_id").Eq(goqu.I("forums_themes.id")))).
			Where(goqu.L("NOT forums_themes.is_moderator"))
	}

	return sqSelect
}

func (s *Forums) Topic(ctx context.Context, topicID int64, userID int64, isModerator bool) (*ForumsTopic, error) {
	sqSelect := s.topicsSelect(isModerator).
		Where(goqu.I("forums_topics.id").Eq(topicID)).
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
			goqu.T("forums_theme_parent"),
			goqu.On(goqu.I("forums_topics.theme_id").Eq(goqu.I("forums_theme_parent.forum_theme_id"))),
		).
		Where(goqu.I("forums_theme_parent.parent_id").Eq(themeID)).
		Order(goqu.I("comment_topic.last_update").Desc()).
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
	sqSelect := s.db.Select("comment_message.id", "comment_message.datetime", "comment_message.author_id").
		From("comment_message").
		Join(
			goqu.T("forums_topics"),
			goqu.On(goqu.I("comment_message.item_id").Eq(goqu.I("forums_topics.id"))),
		).
		Join(
			goqu.T("forums_theme_parent"),
			goqu.On(goqu.I("forums_topics.theme_id").Eq(goqu.I("forums_theme_parent.forum_theme_id"))),
		).
		Where(
			goqu.I("forums_topics.status").In([]string{TopicStatusNormal, TopicStatusClosed}),
			goqu.I("forums_topics.id").Eq(topicID),
			goqu.I("comment_message.type_id").Eq(comments.TypeIDForums),
		).
		Order(goqu.I("comment_message.datetime").Desc()).
		Limit(1)

	if !isModerator {
		sqSelect = sqSelect.
			Join(goqu.T("forums_themes"), goqu.On(goqu.I("forums_theme_parent.parent_id").Eq(goqu.I("forums_themes.id")))).
			Where(goqu.L("NOT forums_themes.is_moderator"))
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
		Order(goqu.I("comment_topic.last_update").Desc())

	if themeID > 0 {
		sqSelect = sqSelect.Where(goqu.I("forums_topics.theme_id").Eq(themeID))
	}

	if subscription {
		sqSelect = sqSelect.Join(
			goqu.I("comment_topic_subscribe"),
			goqu.On(
				goqu.I("forums_topics.id").Eq(goqu.I("comment_topic_subscribe.item_id")),
				goqu.I("comment_topic_subscribe.type_id").Eq(comments.TypeIDForums),
			),
		).
			Where(goqu.I("comment_topic_subscribe.user_id").Eq(userID))
	}

	rows := make([]*ForumsTopic, 0)

	paginator := util.Paginator{
		SQLSelect: sqSelect,
	}

	sqSelect, err := paginator.GetItemsByPage(ctx, page)
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
