package pictures

import (
	"context"
	"database/sql"
)

type Status string

const (
	StatusAccepted Status = "accepted"
	StatusRemoving Status = "removing"
	StatusRemoved  Status = "removed"
	StatusInbox    Status = "inbox"
)

type ItemPictureType int

const (
	ItemPictureContent    ItemPictureType = 1
	ItemPictureAuthor     ItemPictureType = 2
	ItemPictureCopyrights ItemPictureType = 3
)

type VoteSummary struct {
	Value    int32
	Positive int32
	Negative int32
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (s Repository) IncView(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO picture_view (picture_id, views)
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE views=views+1
    `, id)
	return err
}

func (s Repository) GetVote(ctx context.Context, id int64, userID int64) (error, *VoteSummary) {
	var value, positive, negative int32
	if userID > 0 {
		err := s.db.QueryRowContext(
			ctx,
			"SELECT value FROM picture_vote WHERE picture_id = ? AND user_id = ?",
			id, userID,
		).Scan(&value)
		if err != nil {
			return err, nil
		}
	}

	err := s.db.QueryRowContext(
		ctx,
		"SELECT positive, negative FROM picture_vote_summary WHERE picture_id = ?",
		id,
	).Scan(&positive, &negative)

	if err != nil {
		return err, nil
	}

	return nil, &VoteSummary{
		Value:    value,
		Positive: positive,
		Negative: negative,
	}
}

func (s Repository) Vote(ctx context.Context, id int64, value int32, userID int64) error {
	normalizedValue := 1
	if value < 0 {
		normalizedValue = -1
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO picture_vote (picture_id, user_id, value, timestamp)
		VALUES (?, ?, ?, now())
		ON DUPLICATE KEY UPDATE value = VALUES(value),
		timestamp = VALUES(timestamp)
    `, id, userID, normalizedValue)

	if err != nil {
		return err
	}

	return s.updatePictureSummary(ctx, id)
}

func (s Repository) updatePictureSummary(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
        insert into picture_vote_summary (picture_id, positive, negative)
		values (
			?,
			(select count(1) from picture_vote where picture_id = ? and value > 0),
			(select count(1) from picture_vote where picture_id = ? and value < 0)
		)
		on duplicate key update
			positive = VALUES(positive),
			negative = VALUES(negative)
    `, id, id, id)

	return err
}
