package pictures

import (
	"context"
	"database/sql"

	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type Status string

const IdentityLength = 6

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

type PictureRow struct {
	OwnerID            sql.NullInt64 `db:"owner_id"`
	ChangeStatusUserID sql.NullInt64 `db:"change_status_user_id"`
	Identity           string        `db:"identity"`
	Status             Status        `db:"status"`
	ImageID            int64         `db:"image_id"`
}

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

type ListOptions struct {
	Status         Status
	AncestorItemID int64
	HasCopyrights  bool
	UserID         int64
}

type RatingUser struct {
	OwnerID int64 `db:"owner_id"`
	Volume  int64 `db:"volume"`
}

type RatingFan struct {
	UserID int64 `db:"user_id"`
	Volume int64 `db:"volume"`
}

type Repository struct {
	db           *goqu.Database
	imageStorage *storage.Storage
}

func NewRepository(db *goqu.Database, imageStorage *storage.Storage) *Repository {
	return &Repository{
		db:           db,
		imageStorage: imageStorage,
	}
}

func (s *Repository) IncView(ctx context.Context, id int64) error {
	_, err := s.db.Insert(schema.PictureViewTable).
		Rows(goqu.Record{
			schema.PictureViewTablePictureIDColName: id,
			schema.PictureViewTableViewsColName:     1,
		}).
		OnConflict(goqu.DoUpdate(schema.PictureViewTablePictureIDColName, goqu.Record{
			schema.PictureViewTableViewsColName: goqu.L("? + 1", schema.PictureViewTableViewsCol),
		})).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) Status(ctx context.Context, id int64) (Status, error) {
	var status Status

	success, err := s.db.Select(schema.PictureTableStatusCol).
		From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(id)).
		ScanValContext(ctx, &status)
	if err != nil {
		return "", err
	}

	if !success {
		return "", sql.ErrNoRows
	}

	return status, nil
}

func (s *Repository) SetStatus(ctx context.Context, id int64, status Status, userID int64) error {
	_, err := s.db.Update(schema.PictureTable).
		Set(goqu.Record{
			schema.PictureTableStatusColName:             status,
			schema.PictureTableChangeStatusUserIDColName: userID,
		}).
		Where(schema.PictureTableIDCol.Eq(id)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) GetVote(ctx context.Context, id int64, userID int64) (*VoteSummary, error) {
	var value int32
	if userID > 0 {
		success, err := s.db.Select(schema.PictureVoteTableValueCol).
			From(schema.PictureVoteTable).
			Where(
				schema.PictureVoteTablePictureIDCol.Eq(id),
				schema.PictureVoteTableUserIDCol.Eq(userID),
			).
			ScanValContext(ctx, &value)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil, sql.ErrNoRows
		}
	}

	st := struct {
		Positive int32 `db:"positive"`
		Negative int32 `db:"negative"`
	}{}

	success, err := s.db.Select(schema.PictureVoteSummaryTablePositiveCol, schema.PictureVoteSummaryTableNegativeCol).
		From(schema.PictureVoteSummaryTable).
		Where(schema.PictureVoteSummaryTablePictureIDCol.Eq(id)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, sql.ErrNoRows
	}

	return &VoteSummary{
		Value:    value,
		Positive: st.Positive,
		Negative: st.Negative,
	}, nil
}

func (s *Repository) Vote(ctx context.Context, id int64, value int32, userID int64) error {
	normalizedValue := 1
	if value < 0 {
		normalizedValue = -1
	}

	_, err := s.db.Insert(schema.PictureVoteTable).Rows(goqu.Record{
		schema.PictureVoteTablePictureIDColName: id,
		schema.PictureVoteTableUserIDColName:    userID,
		schema.PictureVoteTableValueColName:     normalizedValue,
		schema.PictureVoteTableTimestampColName: goqu.Func("NOW"),
	}).OnConflict(goqu.DoUpdate(
		schema.PictureVoteTablePictureIDColName+","+schema.PictureVoteTableUserIDColName,
		goqu.Record{
			schema.PictureVoteTableValueColName:     goqu.Func("VALUES", goqu.C(schema.PictureVoteTableValueColName)),
			schema.PictureVoteTableTimestampColName: goqu.Func("VALUES", goqu.C(schema.PictureVoteTableTimestampColName)),
		},
	)).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	return s.updatePictureSummary(ctx, id)
}

func (s *Repository) CreateModerVoteTemplate(ctx context.Context, tpl ModerVoteTemplate) (ModerVoteTemplate, error) {
	if tpl.Vote < 0 {
		tpl.Vote = -1
	}

	if tpl.Vote > 0 {
		tpl.Vote = 1
	}

	res, err := s.db.Insert(schema.PictureModerVoteTemplateTable).Rows(goqu.Record{
		schema.PictureModerVoteTemplateTableUserIDColName: tpl.UserID,
		schema.PictureModerVoteTemplateTableReasonColName: tpl.Message,
		schema.PictureModerVoteTemplateTableVoteColName:   tpl.Vote,
	}).Executor().ExecContext(ctx)
	if err != nil {
		return tpl, err
	}

	tpl.ID, err = res.LastInsertId()
	if err != nil {
		return tpl, err
	}

	return tpl, err
}

func (s *Repository) DeleteModerVoteTemplate(ctx context.Context, id int64, userID int64) error {
	_, err := s.db.Delete(schema.PictureModerVoteTemplateTable).
		Where(
			schema.PictureModerVoteTemplateTableUserIDCol.Eq(userID),
			schema.PictureModerVoteTemplateTableIDCol.Eq(id),
		).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) IsModerVoteTemplateExists(ctx context.Context, userID int64, reason string) (bool, error) {
	var id int64

	success, err := s.db.Select(schema.PictureModerVoteTemplateTableIDCol).
		From(schema.PictureModerVoteTemplateTable).
		Where(
			schema.PictureModerVoteTemplateTableUserIDCol.Eq(userID),
			schema.PictureModerVoteTemplateTableReasonCol.Eq(reason),
		).
		ScanValContext(ctx, &id)

	return success, err
}

func (s *Repository) GetModerVoteTemplates(ctx context.Context, userID int64) ([]ModerVoteTemplate, error) {
	rows, err := s.db.Select(
		schema.PictureModerVoteTemplateTableIDCol,
		schema.PictureModerVoteTemplateTableReasonCol,
		schema.PictureModerVoteTemplateTableVoteCol,
	).
		From(schema.PictureModerVoteTemplateTable).
		Where(schema.PictureModerVoteTemplateTableUserIDCol.Eq(userID)).
		Order(schema.PictureModerVoteTemplateTableReasonCol.Asc()).
		Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	var items []ModerVoteTemplate

	for rows.Next() {
		var row ModerVoteTemplate

		err = rows.Scan(&row.ID, &row.Message, &row.Vote)
		if err != nil {
			return nil, err
		}

		items = append(items, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Repository) updatePictureSummary(ctx context.Context, id int64) error {
	_, err := s.db.Insert(schema.PictureVoteSummaryTable).Rows(goqu.Record{
		schema.PictureVoteSummaryTablePictureIDColName: id,
		schema.PictureVoteSummaryTablePositiveColName: s.db.Select(goqu.COUNT(goqu.Star())).
			From(schema.PictureVoteTable).
			Where(schema.PictureVoteTablePictureIDCol.Eq(id), schema.PictureVoteTableValueCol.Gt(0)),
		schema.PictureVoteSummaryTableNegativeColName: s.db.Select(goqu.COUNT(goqu.Star())).
			From(schema.PictureVoteTable).
			Where(schema.PictureVoteTablePictureIDCol.Eq(id), schema.PictureVoteTableValueCol.Lt(0)),
	}).OnConflict(goqu.DoUpdate(schema.PictureVoteSummaryTablePictureIDColName, goqu.Record{
		schema.PictureVoteSummaryTablePositiveColName: goqu.Func(
			"VALUES", goqu.C(schema.PictureVoteSummaryTablePositiveColName),
		),
		schema.PictureVoteSummaryTableNegativeColName: goqu.Func(
			"VALUES", goqu.C(schema.PictureVoteSummaryTableNegativeColName),
		),
	})).Executor().ExecContext(ctx)

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

func (s *Repository) CountSelect(options ListOptions) (*goqu.SelectDataset, error) {
	alias := "p"

	sqSelect := s.db.Select(goqu.COUNT(goqu.DISTINCT(goqu.T(alias).Col("id")))).
		From(schema.PictureTable.As(alias))

	sqSelect = s.applyPicture(alias, sqSelect, &options)

	return sqSelect, nil
}

func (s *Repository) Count(ctx context.Context, options ListOptions) (int, error) {
	var err error

	sqSelect, err := s.CountSelect(options)
	if err != nil {
		return 0, err
	}

	var count int

	_, err = sqSelect.Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) applyPicture(
	alias string,
	sqSelect *goqu.SelectDataset,
	options *ListOptions,
) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if options.Status != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableStatusColName).Eq(options.Status))
	}

	if options.AncestorItemID != 0 {
		sqSelect = sqSelect.
			Join(schema.PictureItemTable, goqu.On(aliasTable.Col("id").Eq(schema.PictureItemTablePictureIDCol))).
			Join(
				schema.ItemParentCacheTable,
				goqu.On(schema.PictureItemTableItemIDCol.Eq(schema.ItemParentCacheTableItemIDCol)),
			).
			Where(schema.ItemParentCacheTableParentIDCol.Eq(options.AncestorItemID))
	}

	if options.HasCopyrights {
		sqSelect = sqSelect.Where(aliasTable.Col("copyrights_text_id").IsNotNull())
	}

	if options.UserID > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableOwnerIDColName).Eq(options.UserID))
	}

	return sqSelect
}

func (s *Repository) TopLikes(ctx context.Context, limit uint) ([]RatingUser, error) {
	rows := make([]RatingUser, 0)

	const volumeAlias = "volume"
	err := s.db.Select(schema.PictureTableOwnerIDCol, goqu.SUM(schema.PictureVoteTableValueCol).As(volumeAlias)).
		From(schema.PictureTable).
		Join(schema.PictureVoteTable, goqu.On(schema.PictureTableIDCol.Eq(schema.PictureVoteTablePictureIDCol))).
		Where(schema.PictureTableOwnerIDCol.Neq(schema.PictureVoteTableUserIDCol)).
		GroupBy(schema.PictureTableOwnerIDCol).
		Order(goqu.C(volumeAlias).Desc()).
		Limit(limit).
		ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) TopOwnerFans(ctx context.Context, userID int64, limit uint) ([]RatingFan, error) {
	rows := make([]RatingFan, 0)

	const volumeAlias = "volume"
	err := s.db.Select(schema.PictureVoteTableUserIDCol, goqu.COUNT(goqu.Star()).As(volumeAlias)).
		From(schema.PictureTable).
		Join(schema.PictureVoteTable, goqu.On(schema.PictureTableIDCol.Eq(schema.PictureVoteTablePictureIDCol))).
		Where(schema.PictureTableOwnerIDCol.Eq(userID)).
		GroupBy(schema.PictureVoteTableUserIDCol).
		Order(goqu.C(volumeAlias).Desc()).
		Limit(limit).
		ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) DeleteModerVote(ctx context.Context, pictureID int64, userID int64) (bool, error) {
	res, err := s.db.Delete(schema.PicturesModerVotesTable).
		Where(
			schema.PicturesModerVotesTableUserIDCol.Eq(userID),
			schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
		).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) CreateModerVote(
	ctx context.Context, pictureID int64, userID int64, vote bool, reason string,
) (bool, error) {
	res, err := s.db.Insert(schema.PicturesModerVotesTable).Rows(goqu.Record{
		schema.PicturesModerVotesTablePictureIDColName: pictureID,
		schema.PicturesModerVotesTableUserIDColName:    userID,
		schema.PicturesModerVotesTableVoteColName:      vote,
		schema.PicturesModerVotesTableReasonColName:    reason,
		schema.PicturesModerVotesTableDayDateColName:   goqu.Func("NOW"),
	}).OnConflict(
		goqu.DoUpdate(
			schema.PicturesModerVotesTablePictureIDColName+","+schema.PicturesModerVotesTableUserIDColName,
			goqu.Record{
				schema.PicturesModerVotesTableVoteColName: goqu.Func("VALUES",
					goqu.C(schema.PicturesModerVotesTableVoteColName)),
				schema.PicturesModerVotesTableReasonColName: goqu.Func("VALUES",
					goqu.C(schema.PicturesModerVotesTableReasonColName)),
				schema.PicturesModerVotesTableDayDateColName: goqu.Func("VALUES",
					goqu.C(schema.PicturesModerVotesTableDayDateColName)),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) Picture(ctx context.Context, id int64) (*PictureRow, error) {
	st := PictureRow{}

	success, err := s.db.Select(
		schema.PictureTableOwnerIDCol, schema.PictureTableChangeStatusUserIDCol, schema.PictureTableIdentityCol,
		schema.PictureTableStatusCol, schema.PictureTableImageIDCol,
	).
		From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(id)).
		ScanStructContext(ctx, &st)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, sql.ErrNoRows
	}

	return &st, nil
}

func (s *Repository) Normalize(ctx context.Context, id int64) error {
	pic, err := s.Picture(ctx, id)
	if err != nil {
		return err
	}

	if pic.ImageID != 0 {
		if err = s.imageStorage.Normalize(ctx, int(pic.ImageID)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) Flop(ctx context.Context, id int64) error {
	pic, err := s.Picture(ctx, id)
	if err != nil {
		return err
	}

	if pic.ImageID != 0 {
		if err = s.imageStorage.Flop(ctx, int(pic.ImageID)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) Repair(ctx context.Context, id int64) error {
	return s.imageStorage.Flush(ctx, storage.FlushOptions{Image: int(id)})
}
