package comments

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

type CommentType int32

const (
	TypeIDPictures CommentType = 1
	TypeIDItems    CommentType = 2
	TypeIDVotings  CommentType = 3
	TypeIDArticles CommentType = 4
	TypeIDForums   CommentType = 5
)

type ModeratorAttention int32

const (
	ModeratorAttentionNone      ModeratorAttention = 0
	ModeratorAttentionRequired  ModeratorAttention = 1
	ModeratorAttentionCompleted ModeratorAttention = 2
)

type GetVotesResult struct {
	PositiveVotes []users.DBUser
	NegativeVotes []users.DBUser
}

// Repository Main Object.
type Repository struct {
	db *goqu.Database
}

// NewRepository constructor.
func NewRepository(db *goqu.Database) *Repository {
	return &Repository{
		db: db,
	}
}

func (s *Repository) GetVotes(ctx context.Context, id int64) (*GetVotesResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT users.id, users.name, users.deleted, users.identity, users.last_online, users.role, 
            users.specs_weight, comment_vote.vote
		FROM comment_vote
			INNER JOIN users ON comment_vote.user_id = users.id
		WHERE comment_vote.comment_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	positiveVotes := make([]users.DBUser, 0)
	negativeVotes := make([]users.DBUser, 0)

	for rows.Next() {
		var (
			r    users.DBUser
			vote int
		)

		err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight, &vote)
		if err != nil {
			return nil, err
		}

		if vote > 0 {
			positiveVotes = append(positiveVotes, r)
		} else {
			negativeVotes = append(negativeVotes, r)
		}
	}

	return &GetVotesResult{
		PositiveVotes: positiveVotes,
		NegativeVotes: negativeVotes,
	}, nil
}

func (s *Repository) Subscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT IGNORE INTO comment_topic_subscribe (type_id, item_id, user_id, sent)
		VALUES (?, ?, ?, 0)
    `, commentsType, itemID, userID)

	return err
}

func (s *Repository) UnSubscribe(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"DELETE FROM comment_topic_subscribe WHERE type_id = ? AND item_id = ? AND user_id = ?",
		commentsType, itemID, userID,
	)

	return err
}

func (s *Repository) View(ctx context.Context, userID int64, commentsType CommentType, itemID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO comment_topic_view (user_id, type_id, item_id, timestamp)
            VALUES (?, ?, ?, NOW())
            ON DUPLICATE KEY UPDATE timestamp = values(timestamp)
        `,
		userID, commentsType, itemID,
	)

	return err
}

func (s *Repository) QueueDeleteMessage(ctx context.Context, commentID int64, byUserID int64) error {
	var moderatorAttention ModeratorAttention

	err := s.db.QueryRowContext(ctx, "SELECT moderator_attention FROM comment_message WHERE id = ?", commentID).
		Scan(&moderatorAttention)
	if err != nil {
		return err
	}

	if moderatorAttention == ModeratorAttentionRequired {
		return errors.New("comment with moderation attention requirement can't be deleted")
	}

	_, err = s.db.ExecContext(
		ctx,
		`
			UPDATE comment_message SET deleted = 1, deleted_by = ?, delete_date = NOW()
            WHERE id = ?
        `,
		byUserID, commentID,
	)

	return err
}

func (s *Repository) RestoreMessage(ctx context.Context, commentID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET deleted = 0, delete_date = null WHERE id = ?",
		commentID,
	)

	return err
}

func (s *Repository) GetCommentType(ctx context.Context, commentID int64) (CommentType, error) {
	var commentType CommentType
	err := s.db.QueryRowContext(ctx, "SELECT type_id FROM comment_message WHERE id = ?", commentID).Scan(&commentType)

	return commentType, err
}

func (s *Repository) MoveMessage(ctx context.Context, commentID int64, dstType CommentType, dstItemID int64) error {
	var (
		srcType   CommentType
		srcItemID int64
	)

	err := s.db.QueryRowContext(ctx, "SELECT type_id, item_id FROM comment_message WHERE id = ?", commentID).
		Scan(&srcType, &srcItemID)
	if err != nil {
		return err
	}

	if srcItemID == dstItemID && srcType == dstType {
		return nil
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET type_id = ?, item_id = ?, parent_id = null WHERE id = ?",
		dstType, dstItemID, commentID,
	)
	if err != nil {
		return err
	}

	err = s.moveMessageRecursive(ctx, commentID, dstType, dstItemID)
	if err != nil {
		return err
	}

	err = s.updateTopicStat(ctx, srcType, srcItemID)
	if err != nil {
		return err
	}

	return s.updateTopicStat(ctx, dstType, dstItemID)
}

func (s *Repository) moveMessageRecursive(
	ctx context.Context,
	parentID int64,
	dstType CommentType,
	dstItemID int64,
) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE comment_message SET type_id = ?, item_id = ? WHERE id = ?",
		dstType, dstItemID, parentID,
	)
	if err != nil {
		return err
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id FROM comment_message WHERE parent_id = ?", parentID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)

		if err != nil {
			return err
		}

		err = s.moveMessageRecursive(ctx, id, dstType, dstItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) updateTopicStat(ctx context.Context, commentType CommentType, itemID int64) error {
	var (
		messagesCount int
		lastUpdate    *string
	)

	err := s.db.QueryRowContext(
		ctx,
		"SELECT COUNT(1), MAX(datetime) FROM comment_message WHERE type_id = ? AND item_id = ?",
		commentType, itemID,
	).Scan(&messagesCount, &lastUpdate)
	if err != nil {
		return err
	}

	if messagesCount <= 0 {
		_, err = s.db.ExecContext(
			ctx,
			"DELETE FROM comment_topic WHERE type_id = ? AND item_id = ?",
			commentType, itemID,
		)

		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`
            INSERT INTO comment_topic (item_id, type_id, last_update, messages)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE last_update = VALUES(last_update), messages = VALUES(messages)
        `,
		itemID, commentType, lastUpdate, messagesCount,
	)

	return err
}

func (s *Repository) VoteComment(ctx context.Context, userID int64, commentID int64, vote int32) (int32, error) {
	if vote > 0 {
		vote = 1
	} else {
		vote = -1
	}

	var authorID int64

	err := s.db.QueryRowContext(
		ctx, "SELECT author_id FROM comment_message WHERE id = ?", commentID,
	).Scan(&authorID)
	if err != nil {
		return 0, err
	}

	if authorID == userID {
		return 0, errors.New("self-vote forbidden")
	}

	res, err := s.db.ExecContext(
		ctx,
		`
            INSERT INTO comment_vote (comment_id, user_id, vote)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE vote = VALUES(vote)
        `,
		commentID, userID, vote,
	)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if affected == 0 {
		return 0, errors.New("already voted")
	}

	newVote, err := s.updateVote(ctx, commentID)
	if err != nil {
		return 0, err
	}

	return newVote, nil
}

func (s *Repository) updateVote(ctx context.Context, commentID int64) (int32, error) {
	var count int32

	err := s.db.QueryRowContext(
		ctx,
		"SELECT sum(vote) FROM comment_vote WHERE comment_id = ?",
		commentID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	_, err = s.db.ExecContext(
		ctx, "UPDATE comment_message SET vote = ? WHERE id = ?", count, commentID,
	)

	return count, err
}
