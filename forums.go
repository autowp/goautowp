package goautowp

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

const (
	TopicStatusNormal  = "normal"
	TopicStatusClosed  = "closed"
	TopicStatusDeleted = "deleted"
)

// Forums Main Object.
type Forums struct {
	db *goqu.Database
}

func NewForums(db *goqu.Database) *Forums {
	return &Forums{
		db: db,
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
	if err != nil {
		return 0, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	defer util.Close(rows)

	result := 0
	if rows.Next() {
		err = rows.Scan(&result)
		if err != nil {
			return 0, err
		}
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
