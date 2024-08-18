package pictures

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var (
	errIsAllowedForPictureItemContentOnly = errors.New("is allowed only for picture-item-content")
	errCombinationNotAllowed              = errors.New("combination not allowed")
)

const (
	IdentityLength                    = 6
	ModerVoteTemplateMessageMaxLength = 80
)

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
	Status         schema.PictureStatus
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

func (s *Repository) Status(ctx context.Context, id int64) (schema.PictureStatus, error) {
	var status schema.PictureStatus

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

func (s *Repository) SetStatus(ctx context.Context, id int64, status schema.PictureStatus, userID int64) error {
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

func (s *Repository) Picture(ctx context.Context, id int64) (*schema.PictureRow, error) {
	st := schema.PictureRow{}

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

func (s *Repository) SetPictureItemArea(
	ctx context.Context, pictureID int64, itemID int64, pictureItemType schema.PictureItemType, area PictureItemArea,
) error {
	if pictureItemType != schema.PictureItemContent {
		return errIsAllowedForPictureItemContentOnly
	}

	pic := schema.PictureRow{}

	success, err := s.db.Select(schema.PictureTableWidthCol, schema.PictureTableHeightCol).
		From(schema.PictureTable).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		ScanStructContext(ctx, &pic)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	picItem := schema.PictureItemRow{}

	success, err = s.db.Select(
		schema.PictureItemTableCropLeftCol, schema.PictureItemTableCropTopCol, schema.PictureItemTableCropWidthCol,
		schema.PictureItemTableCropHeightCol, schema.PictureItemTableTypeCol,
	).
		From(schema.PictureItemTable).
		Where(
			schema.PictureItemTablePictureIDCol.Eq(pictureID),
			schema.PictureItemTableItemIDCol.Eq(itemID),
			schema.PictureItemTableTypeCol.Eq(pictureItemType),
		).ScanStructContext(ctx, &picItem)
	if err != nil {
		return err
	}

	if !success {
		return sql.ErrNoRows
	}

	area.Left = util.Max(0, area.Left)
	area.Left = util.Min(pic.Width, area.Left)
	area.Width = util.Max(0, area.Width)
	area.Width = util.Min(pic.Width, area.Width)

	area.Top = util.Max(0, area.Top)
	area.Top = util.Min(pic.Height, area.Top)
	area.Height = util.Max(0, area.Height)
	area.Height = util.Min(pic.Height, area.Height)

	isFull := area.Left == 0 && area.Top == 0 && area.Width == pic.Width && area.Height == pic.Height
	isEmpty := area.Height == 0 || area.Width == 0
	valid := !isEmpty && !isFull

	picItem.CropLeft = sql.NullInt32{
		Valid: valid,
		Int32: int32(area.Left),
	}
	picItem.CropTop = sql.NullInt32{
		Valid: valid,
		Int32: int32(area.Top),
	}
	picItem.CropWidth = sql.NullInt32{
		Valid: valid,
		Int32: int32(area.Width),
	}
	picItem.CropHeight = sql.NullInt32{
		Valid: valid,
		Int32: int32(area.Height),
	}

	_, err = s.db.Update(schema.PictureItemTable).
		Set(goqu.Record{
			schema.PictureItemTableCropLeftColName:   picItem.CropLeft,
			schema.PictureItemTableCropTopColName:    picItem.CropTop,
			schema.PictureItemTableCropWidthColName:  picItem.CropWidth,
			schema.PictureItemTableCropHeightColName: picItem.CropHeight,
		}).
		Where(
			schema.PictureItemTablePictureIDCol.Eq(pictureID),
			schema.PictureItemTableItemIDCol.Eq(itemID),
			schema.PictureItemTableTypeCol.Eq(pictureItemType),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) SetPictureItemPerspective(
	ctx context.Context, pictureID int64, itemID int64, pictureItemType schema.PictureItemType, perspective int32,
) error {
	if pictureItemType != schema.PictureItemContent {
		return errIsAllowedForPictureItemContentOnly
	}

	_, err := s.db.Update(schema.PictureItemTable).
		Set(goqu.Record{
			schema.PictureItemTablePerspectiveIDColName: sql.NullInt32{
				Valid: perspective > 0,
				Int32: perspective,
			},
		}).
		Where(
			schema.PictureItemTablePictureIDCol.Eq(pictureID),
			schema.PictureItemTableItemIDCol.Eq(itemID),
			schema.PictureItemTableTypeCol.Eq(pictureItemType),
		).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) SetPictureItemItemID(
	ctx context.Context, pictureID int64, itemID int64, pictureItemType schema.PictureItemType, dstItemID int64,
) error {
	if pictureItemType != schema.PictureItemContent {
		return errIsAllowedForPictureItemContentOnly
	}

	isAllowed, err := s.isAllowedTypeByItemID(ctx, dstItemID, pictureItemType)
	if err != nil {
		return err
	}

	if !isAllowed {
		return errCombinationNotAllowed
	}

	res, err := s.db.Update(schema.PictureItemTable).
		Set(goqu.Record{
			schema.PictureItemTableItemIDColName: dstItemID,
		}).
		Where(
			schema.PictureItemTablePictureIDCol.Eq(pictureID),
			schema.PictureItemTableItemIDCol.Eq(itemID),
			schema.PictureItemTableTypeCol.Eq(pictureItemType),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected > 0 {
		err = s.updateContentCount(ctx, pictureID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) isAllowedTypeByItemID(
	ctx context.Context, itemID int64, pictureItemType schema.PictureItemType,
) (bool, error) {
	var itemTypeID schema.ItemTableItemTypeID

	success, err := s.db.Select(schema.ItemTableItemTypeIDCol).
		From(schema.ItemTable).Where(schema.ItemTableIDCol.Eq(itemID)).
		ScanValContext(ctx, &itemTypeID)
	if err != nil {
		return false, err
	}

	if !success {
		return false, sql.ErrNoRows
	}

	return s.isAllowedType(itemTypeID, pictureItemType), nil
}

func (s *Repository) isAllowedType(itemTypeID schema.ItemTableItemTypeID, pictureItemType schema.PictureItemType) bool {
	allowed := map[schema.ItemTableItemTypeID][]schema.PictureItemType{
		schema.ItemTableItemTypeIDBrand:    {schema.PictureItemContent, schema.PictureItemCopyrights},
		schema.ItemTableItemTypeIDCategory: {schema.PictureItemContent},
		schema.ItemTableItemTypeIDEngine:   {schema.PictureItemContent},
		schema.ItemTableItemTypeIDFactory:  {schema.PictureItemContent},
		schema.ItemTableItemTypeIDVehicle:  {schema.PictureItemContent},
		schema.ItemTableItemTypeIDTwins:    {schema.PictureItemContent},
		schema.ItemTableItemTypeIDMuseum:   {schema.PictureItemContent},
		schema.ItemTableItemTypeIDPerson: {
			schema.PictureItemContent, schema.PictureItemAuthor, schema.PictureItemCopyrights,
		},
		schema.ItemTableItemTypeIDCopyright: {schema.PictureItemCopyrights},
	}

	pictureItemTypes, ok := allowed[itemTypeID]
	if !ok {
		return false
	}

	return util.Contains(pictureItemTypes, pictureItemType)
}

func (s *Repository) DeletePictureItem(
	ctx context.Context, pictureID int64, itemID int64, pictureItemType schema.PictureItemType,
) (bool, error) {
	res, err := s.db.Delete(schema.PictureItemTable).Where(
		schema.PictureItemTablePictureIDCol.Eq(pictureID),
		schema.PictureItemTableItemIDCol.Eq(itemID),
		schema.PictureItemTableTypeCol.Eq(pictureItemType),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if affected > 0 {
		err = s.updateContentCount(ctx, pictureID)
		if err != nil {
			return false, err
		}
	}

	return affected > 0, nil
}

func (s *Repository) CreatePictureItem(
	ctx context.Context, pictureID int64, itemID int64, pictureItemType schema.PictureItemType, perspective int32,
) (bool, error) {
	if pictureItemType != schema.PictureItemContent {
		return false, errIsAllowedForPictureItemContentOnly
	}

	isAllowed, err := s.isAllowedTypeByItemID(ctx, itemID, pictureItemType)
	if err != nil {
		return false, err
	}

	if !isAllowed {
		return false, errCombinationNotAllowed
	}

	res, err := s.db.Insert(schema.PictureItemTable).Rows(goqu.Record{
		schema.PictureItemTablePictureIDColName: pictureID,
		schema.PictureItemTableItemIDColName:    itemID,
		schema.PictureItemTableTypeColName:      pictureItemType,
		schema.PictureItemTablePerspectiveIDColName: sql.NullInt32{
			Valid: perspective > 0,
			Int32: perspective,
		},
	}).OnConflict(goqu.DoNothing()).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if affected > 0 {
		err = s.updateContentCount(ctx, pictureID)
		if err != nil {
			return false, err
		}
	}

	return affected > 0, nil
}

func (s *Repository) updateContentCount(ctx context.Context, pictureID int64) error {
	_, err := s.db.Update(schema.PictureTable).
		Set(goqu.Record{
			schema.PictureTableContentCountColName: s.db.Select(goqu.COUNT(goqu.Star())).
				From(schema.PictureItemTable).
				Where(
					schema.PictureItemTablePictureIDCol.Eq(pictureID),
					schema.PictureItemTableTypeCol.Eq(schema.PictureItemContent),
				),
		}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)

	return err
}
