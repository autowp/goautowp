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
