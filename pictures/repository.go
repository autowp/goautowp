package pictures

import (
	"context"

	"github.com/autowp/goautowp/util"

	"github.com/autowp/goautowp/validation"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

const ModerVoteTemplateMessageMaxLength = 80

type ModerVoteTemplate struct {
	ID      int64
	UserID  int64
	Message string
	Vote    int32
}

type VoteSummary struct {
	Value    int32
	Positive int32
	Negative int32
}

type Repository struct {
	db *goqu.Database
}

func NewRepository(db *goqu.Database) *Repository {
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

func (s Repository) GetVote(ctx context.Context, id int64, userID int64) (*VoteSummary, error) {
	var value, positive, negative int32
	if userID > 0 {
		err := s.db.QueryRowContext(
			ctx,
			"SELECT value FROM picture_vote WHERE picture_id = ? AND user_id = ?",
			id, userID,
		).Scan(&value)
		if err != nil {
			return nil, err
		}
	}

	err := s.db.QueryRowContext(
		ctx,
		"SELECT positive, negative FROM picture_vote_summary WHERE picture_id = ?",
		id,
	).Scan(&positive, &negative)
	if err != nil {
		return nil, err
	}

	return &VoteSummary{
		Value:    value,
		Positive: positive,
		Negative: negative,
	}, nil
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

func (s Repository) CreateModerVoteTemplate(ctx context.Context, tpl ModerVoteTemplate) (ModerVoteTemplate, error) {
	if tpl.Vote < 0 {
		tpl.Vote = -1
	}

	if tpl.Vote > 0 {
		tpl.Vote = 1
	}

	r, err := s.db.ExecContext(ctx, `
        INSERT INTO picture_moder_vote_template (user_id, reason, vote)
		VALUES (?, ?, ?)
    `, tpl.UserID, tpl.Message, tpl.Vote)
	if err != nil {
		return tpl, err
	}

	tpl.ID, err = r.LastInsertId()

	if err != nil {
		return tpl, err
	}

	return tpl, err
}

func (s Repository) DeleteModerVoteTemplate(ctx context.Context, id int64, userID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM picture_moder_vote_template WHERE user_id = ? AND id = ?", userID, id)

	return err
}

func (s Repository) GetModerVoteTemplates(ctx context.Context, id int64) ([]ModerVoteTemplate, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT id, reason, vote FROM picture_moder_vote_template WHERE user_id = ? ORDER BY reason",
		id,
	)
	if err != nil {
		return nil, err
	}

	util.Close(rows)

	var items []ModerVoteTemplate

	for rows.Next() {
		var r ModerVoteTemplate
		err = rows.Scan(&r.ID, &r.Message, &r.Vote)

		if err != nil {
			return nil, err
		}

		items = append(items, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
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

func (s *ModerVoteTemplate) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	result := make([]*errdetails.BadRequest_FieldViolation, 0)

	var (
		problems []string
		err      error
	)

	messageInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.NotEmpty{},
			&validation.StringLength{Max: ModerVoteTemplateMessageMaxLength},
		},
	}
	s.Message, problems, err = messageInputFilter.IsValidString(s.Message)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "message",
			Description: fv,
		})
	}

	return result, nil
}
