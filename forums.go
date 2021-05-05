package goautowp

import (
	"database/sql"
	"github.com/autowp/goautowp/util"
)

// Forums Main Object
type Forums struct {
	db *sql.DB
}

func NewForums(db *sql.DB) *Forums {
	return &Forums{
		db: db,
	}
}

func (s *Forums) GetUserSummary(userID int) (int, error) {
	rows, err := s.db.Query(`
		SELECT count(1)
		FROM forums_topics
			JOIN comment_topic_subscribe ON forums_topics.id = comment_topic_subscribe.item_id
			JOIN comment_topic ON forums_topics.id = comment_topic.item_id
		WHERE comment_topic_subscribe.user_id = ?
		  	AND comment_topic.type_id = ?
			AND comment_topic_subscribe.type_id = ?
	`, userID, CommentsTypeForumTopicID, CommentsTypeForumTopicID)
	if err != nil {
		return 0, err
	}
	if err == sql.ErrNoRows {
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
