package pictures

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/paulmach/orb"
)

var (
	errIsAllowedForPictureItemContentOnly = errors.New("is allowed only for picture-item-content")
	errCombinationNotAllowed              = errors.New("combination not allowed")
	errImageIDIsNil                       = errors.New("image_id is null")
)

type VoteSummary struct {
	Value    int32
	Positive int32
	Negative int32
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
	db                    *goqu.Database
	imageStorage          *storage.Storage
	textStorageRepository *textstorage.Repository
}

func NewRepository(
	db *goqu.Database, imageStorage *storage.Storage, textStorageRepository *textstorage.Repository,
) *Repository {
	return &Repository{
		db:                    db,
		imageStorage:          imageStorage,
		textStorageRepository: textStorageRepository,
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

func (s *Repository) CreateModerVoteTemplate(
	ctx context.Context, tpl schema.PictureModerVoteTemplateRow,
) (schema.PictureModerVoteTemplateRow, error) {
	if tpl.Vote < 0 {
		tpl.Vote = -1
	}

	if tpl.Vote > 0 {
		tpl.Vote = 1
	}

	res, err := s.db.Insert(schema.PictureModerVoteTemplateTable).Rows(tpl).Executor().ExecContext(ctx)
	if err != nil {
		return tpl, err
	}

	tpl.ID, err = res.LastInsertId()

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

func (s *Repository) GetModerVoteTemplates(
	ctx context.Context, userID int64,
) ([]schema.PictureModerVoteTemplateRow, error) {
	var items []schema.PictureModerVoteTemplateRow

	err := s.db.Select(
		schema.PictureModerVoteTemplateTableIDCol,
		schema.PictureModerVoteTemplateTableReasonCol,
		schema.PictureModerVoteTemplateTableVoteCol,
	).
		From(schema.PictureModerVoteTemplateTable).
		Where(schema.PictureModerVoteTemplateTableUserIDCol.Eq(userID)).
		Order(schema.PictureModerVoteTemplateTableReasonCol.Asc()).
		Executor().ScanStructsContext(ctx, &items)

	return items, err
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

func (s *Repository) Count(ctx context.Context, options *query.PictureListOptions) (int, error) {
	var count int

	success, err := options.CountSelect(s.db).Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return count, nil
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
		schema.PictureTableIDCol, schema.PictureTableOwnerIDCol, schema.PictureTableChangeStatusUserIDCol,
		schema.PictureTableIdentityCol, schema.PictureTableStatusCol, schema.PictureTableImageIDCol,
		schema.PictureTablePointCol, schema.PictureTableCopyrightsTextIDCol, schema.PictureTableAcceptDatetimeCol,
		schema.PictureTableReplacePictureIDCol,
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

	if pic.ImageID.Valid {
		if err = s.imageStorage.Normalize(ctx, int(pic.ImageID.Int64)); err != nil {
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

	if pic.ImageID.Valid {
		if err = s.imageStorage.Flop(ctx, int(pic.ImageID.Int64)); err != nil {
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

	area = PictureItemArea(util.IntersectBounds(util.Rect[uint16](area), util.Rect[uint16]{
		Left:   0,
		Top:    0,
		Width:  pic.Width,
		Height: pic.Height,
	}))

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

func (s *Repository) SetPictureCrop(ctx context.Context, pictureID int64, area sampler.Crop) error {
	pic, err := s.Picture(ctx, pictureID)
	if err != nil {
		return err
	}

	if !pic.ImageID.Valid {
		return errImageIDIsNil
	}

	return s.imageStorage.SetImageCrop(ctx, int(pic.ImageID.Int64), area)
}

func (s *Repository) ClearReplacePicture(ctx context.Context, pictureID int64) (bool, error) {
	res, err := s.db.Update(schema.PictureTable).
		Set(goqu.Record{
			schema.PictureTableReplacePictureIDColName: nil,
		}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) SetPicturePoint(ctx context.Context, pictureID int64, point *orb.Point) (bool, error) {
	var pointExpr goqu.Expression

	if point != nil {
		pointExpr = goqu.Func("Point", point.Lon(), point.Lat())
	}

	res, err := util.ExecAndRetryOnDeadlock(ctx,
		s.db.Update(schema.PictureTable).
			Set(goqu.Record{schema.PictureTablePointColName: pointExpr}).
			Where(schema.PictureTableIDCol.Eq(pictureID)).
			Executor(),
	)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) UpdatePicture(
	ctx context.Context, pictureID int64, name string, takenYear int16, takenMonth int8, takenDay int8,
) (bool, error) {
	res, err := s.db.Update(schema.PictureTable).
		Set(goqu.Record{
			schema.PictureTableNameColName: sql.NullString{
				String: name,
				Valid:  len(name) > 0,
			},
			schema.PictureTableTakenYearColName: sql.NullInt16{
				Int16: takenYear,
				Valid: takenYear > 0,
			},
			schema.PictureTableTakenMonthColName: sql.NullInt16{
				Int16: int16(takenMonth),
				Valid: takenMonth > 0,
			},
			schema.PictureTableTakenDayColName: sql.NullInt16{
				Int16: int16(takenDay),
				Valid: takenDay > 0,
			},
		}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) SetPictureCopyrights(
	ctx context.Context, pictureID int64, text string, userID int64,
) (bool, int32, error) {
	picture, err := s.Picture(ctx, pictureID)
	if err != nil {
		return false, 0, err
	}

	if picture.CopyrightsTextID.Valid {
		textID := picture.CopyrightsTextID.Int32

		err = s.textStorageRepository.SetText(ctx, textID, text, userID)
		if err != nil {
			return false, 0, err
		}

		return true, textID, nil
	}

	if text == "" {
		return false, 0, nil
	}

	textID, err := s.textStorageRepository.CreateText(ctx, text, userID)
	if err != nil {
		return false, 0, err
	}

	_, err = s.db.Update(schema.PictureTable).
		Set(goqu.Record{
			schema.PictureTableCopyrightsTextIDColName: textID,
		}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, 0, err
	}

	return true, textID, nil
}

func (s *Repository) CanAccept(ctx context.Context, row *schema.PictureRow) (bool, error) {
	if row.Status != schema.PictureStatusInbox {
		return false, nil
	}

	votes, err := s.NegativeVotesCount(ctx, row.ID)

	return votes <= 0, err
}

func (s *Repository) CanDelete(ctx context.Context, row *schema.PictureRow) (bool, error) {
	if row.Status != schema.PictureStatusInbox {
		return false, nil
	}

	votes, err := s.PositiveVotesCount(ctx, row.ID)

	return votes <= 0, err
}

func (s *Repository) NegativeVotes(ctx context.Context, pictureID int64) ([]schema.PictureModerVoteRow, error) {
	var sts []schema.PictureModerVoteRow

	err := s.db.Select(schema.PicturesModerVotesTableUserIDCol, schema.PicturesModerVotesTableReasonCol).
		From(schema.PicturesModerVotesTable).
		Where(
			schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
			schema.PicturesModerVotesTableVoteCol.Eq(0),
		).ScanStructsContext(ctx, &sts)

	return sts, err
}

func (s *Repository) NegativeVotesCount(ctx context.Context, pictureID int64) (int, error) {
	count, err := s.db.From(schema.PicturesModerVotesTable).Where(
		schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
		schema.PicturesModerVotesTableVoteCol.Eq(0),
	).CountContext(ctx)

	return int(count), err
}

func (s *Repository) PositiveVotesCount(ctx context.Context, pictureID int64) (int, error) {
	count, err := s.db.From(schema.PicturesModerVotesTable).Where(
		schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
		schema.PicturesModerVotesTableVoteCol.Gt(0),
	).CountContext(ctx)

	return int(count), err
}

func (s *Repository) HasVote(ctx context.Context, pictureID int64, userID int64) (bool, error) {
	var exists bool
	success, err := s.db.Select(goqu.V(true)).From(schema.PicturesModerVotesTable).Where(
		schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
		schema.PicturesModerVotesTableUserIDCol.Eq(userID),
	).ScanValContext(ctx, &exists)

	return success && exists, err
}

func (s *Repository) Accept(ctx context.Context, pictureID int64, userID int64) (bool, bool, error) {
	isFirstTimeAccepted := false

	picture, err := s.Picture(ctx, pictureID)
	if err != nil {
		return false, false, err
	}

	rec := goqu.Record{
		schema.PictureTableStatusColName:             schema.PictureStatusAccepted,
		schema.PictureTableChangeStatusUserIDColName: userID,
	}

	if !picture.AcceptDatetime.Valid {
		rec[schema.PictureTableAcceptDatetimeColName] = goqu.Func("NOW")
		isFirstTimeAccepted = true
	}

	res, err := s.db.Update(schema.PictureTable).Set(rec).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, false, err
	}

	affected, err := res.RowsAffected()

	return isFirstTimeAccepted, affected > 0, err
}

func (s *Repository) QueueRemove(ctx context.Context, pictureID int64, userID int64) (bool, error) {
	res, err := s.db.Update(schema.PictureTable).Set(goqu.Record{
		schema.PictureTableStatusColName:             schema.PictureStatusRemoving,
		schema.PictureTableRemovingDateColName:       goqu.Func("CURDATE"),
		schema.PictureTableChangeStatusUserIDColName: userID,
	}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) PictureItem(
	ctx context.Context, options query.PictureItemListOptions,
) (schema.PictureItemRow, error) {
	var row schema.PictureItemRow

	alias := "pi"
	aliasTable := goqu.T(alias)

	success, err := options.Apply(
		alias,
		s.db.Select(
			aliasTable.Col(schema.PictureItemTablePictureIDColName),
			aliasTable.Col(schema.PictureItemTableItemIDColName),
			aliasTable.Col(schema.PictureItemTableTypeColName),
			aliasTable.Col(schema.PictureItemTableCropLeftColName),
			aliasTable.Col(schema.PictureItemTableCropTopColName),
			aliasTable.Col(schema.PictureItemTableCropWidthColName),
			aliasTable.Col(schema.PictureItemTableCropHeightColName),
		).
			From(schema.PictureItemTable.As(alias)).
			Limit(1),
	).ScanStructContext(ctx, &row)
	if err != nil {
		return row, err
	}

	if !success {
		return row, sql.ErrNoRows
	}

	return row, nil
}
