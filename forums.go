package goautowp

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
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
