package pictures

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/autowp/goautowp/image/sampler"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/paulmach/orb"
)

var (
	errIsAllowedForPictureItemContentOnly     = errors.New("is allowed only for picture-item-content")
	errJoinNeededToSortByPerspective          = errors.New("can't sort by perspective: need a join with picture_item")
	errJoinNeededToSortByDfDistanceSimilarity = errors.New(
		"can't sort by df-distance-similarity: need a join with df_distance",
	)
	errJoinNeededToSortByPictureModerVote = errors.New(
		"can't sort by df-distance-similarity: need a join with picture_moder_vote",
	)
	errCombinationNotAllowed = errors.New("combination not allowed")
	errImageIDIsNil          = errors.New("image_id is null")
)

var (
	prefixedPerspectives = []int64{
		schema.PerspectiveInterior, schema.PerspectiveFrontPanel, schema.PerspectiveIDUnderTheHood,
		schema.PerspectiveDashboard, schema.PerspectiveBoot, schema.PerspectiveLogo, schema.PerspectiveMascot,
		schema.PerspectiveSketch, schema.PerspectiveChassis,
	}
	specsTopPerspectives = []int64{
		schema.PerspectiveFrontStrict, schema.PerspectiveFront, schema.Perspective3Div4Left,
		schema.Perspective3Div4Right, schema.PerspectiveLeftStrict, schema.PerspectiveRightStrict,
		schema.PerspectiveBack, schema.PerspectiveRight, schema.PerspectiveLeft, schema.PerspectiveBackStrict,
		schema.PerspectiveInterior,
	}
	specsBottomPerspectives = []int64{
		schema.PerspectiveBackStrict, schema.PerspectiveBack, schema.PerspectiveCutaway, schema.PerspectiveFrontPanel,
		schema.PerspectiveInterior,
	}
	frontPerspectives = []int64{schema.Perspective3Div4Left, schema.Perspective3Div4Right, schema.PerspectiveFront}
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
	itemsRepository       *items.Repository
}

type PictureFields struct {
	NameText bool
}

type OrderBy = int

const (
	OrderByNone OrderBy = iota
	OrderByAddDateDesc
	OrderByAddDateAsc
	OrderByAddDateStrictDesc
	OrderByAddDateStrictAsc
	OrderByResolutionDesc
	OrderByResolutionAsc
	OrderByFilesizeDesc
	OrderByFilesizeAsc
	OrderByComments
	OrderByViews
	OrderByModerVotes
	OrderByRemovingDate
	OrderByLikes
	OrderByDislikes
	OrderByStatus
	OrderByAcceptDatetimeDesc
	OrderByAcceptDatetimeAsc
	OrderByAcceptDatetimeStrictDesc
	OrderByAcceptDatetimeStrictAsc
	OrderByPerspectives
	OrderByDfDistanceSimilarity
	OrderByTopPerspectives
	OrderByBottomPerspectives
	OrderByIDDesc
	OrderByIDAsc
	OrderByFrontPerspectives
	OrderByPerspectivesGroupPerspectives
)

type PictureItemOrderBy = int

const (
	PictureItemOrderByNone PictureItemOrderBy = iota
	PictureItemOrderByFrontPerspectivesFirst
)

func NewRepository(
	db *goqu.Database, imageStorage *storage.Storage, textStorageRepository *textstorage.Repository,
	itemsRepository *items.Repository,
) *Repository {
	return &Repository{
		db:                    db,
		imageStorage:          imageStorage,
		textStorageRepository: textStorageRepository,
		itemsRepository:       itemsRepository,
	}
}

func (s *Repository) PictureViews(ctx context.Context, id int64) (int32, error) {
	var res int32

	success, err := s.db.Select(schema.PictureViewTableViewsCol).
		From(schema.PictureViewTable).
		Where(schema.PictureViewTablePictureIDCol.Eq(id)).
		ScanValContext(ctx, &res)
	if err != nil || !success {
		return 0, err
	}

	return res, nil
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
			value = 0
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
		st.Positive = 0
		st.Negative = 0
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
	var rows []schema.PictureModerVoteTemplateRow

	err := s.db.Select(
		schema.PictureModerVoteTemplateTableIDCol,
		schema.PictureModerVoteTemplateTableReasonCol,
		schema.PictureModerVoteTemplateTableVoteCol,
	).
		From(schema.PictureModerVoteTemplateTable).
		Where(schema.PictureModerVoteTemplateTableUserIDCol.Eq(userID)).
		Order(schema.PictureModerVoteTemplateTableReasonCol.Asc()).
		Executor().ScanStructsContext(ctx, &rows)

	return rows, err
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

	sqSelect, err := options.CountSelect(s.db, query.PictureAlias)
	if err != nil {
		return 0, err
	}

	success, err := sqSelect.Executor().ScanValContext(ctx, &count)
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

func (s *Repository) ModerVoteCount(ctx context.Context, pictureID int64) (int32, int32, error) {
	st := struct {
		Sum   sql.NullInt32 `db:"sum"`
		Count int32         `db:"count"`
	}{}

	success, err := s.db.Select(
		goqu.SUM(goqu.Func("IF", schema.PicturesModerVotesTableVoteCol, goqu.V(1), goqu.V(-1))).As("sum"),
		goqu.COUNT(goqu.Star()).As("count"),
	).
		From(schema.PicturesModerVotesTable).
		Where(schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID)).
		GroupBy().
		ScanStructContext(ctx, &st)
	if err != nil {
		return 0, 0, err
	}

	if !success {
		return 0, 0, nil
	}

	if !st.Sum.Valid {
		st.Sum.Int32 = 0
	}

	return st.Count, st.Sum.Int32, nil
}

func (s *Repository) HasModerVote(ctx context.Context, pictureID int64, userID int64) (bool, error) {
	res := false

	success, err := s.db.Select(goqu.V(true)).
		From(schema.PicturesModerVotesTable).
		Where(
			schema.PicturesModerVotesTablePictureIDCol.Eq(pictureID),
			schema.PicturesModerVotesTableUserIDCol.Eq(userID),
		).
		ScanValContext(ctx, &res)

	return success && res, err
}

func (s *Repository) orderBy( //nolint: maintidx
	sqSelect *goqu.SelectDataset, options *query.PictureListOptions, order OrderBy, groupBy bool,
) (*goqu.SelectDataset, bool, error) {
	var (
		alias      = query.PictureAlias
		aliasTable = goqu.T(alias)
	)

	switch order {
	case OrderByAddDateStrictDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableAddDateColName).Desc())
	case OrderByAddDateStrictAsc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableAddDateColName).Asc())
	case OrderByAddDateDesc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
			aliasTable.Col(schema.PictureTableIDColName).Desc(),
		)
	case OrderByAddDateAsc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableAddDateColName).Asc(),
			aliasTable.Col(schema.PictureTableIDColName).Asc(),
		)
	case OrderByResolutionDesc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableWidthColName).Desc(),
			aliasTable.Col(schema.PictureTableHeightColName).Desc(),
			aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
			aliasTable.Col(schema.PictureTableIDColName).Desc(),
		)
	case OrderByResolutionAsc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableWidthColName).Asc(),
			aliasTable.Col(schema.PictureTableHeightColName).Asc(),
		)
	case OrderByFilesizeDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableFilesizeColName).Desc())
	case OrderByFilesizeAsc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableFilesizeColName).Asc())
	case OrderByComments:
		ctoAlias := alias + "cto"
		sqSelect = sqSelect.
			LeftJoin(schema.CommentTopicTable.As(ctoAlias), goqu.On(
				aliasTable.Col(schema.PictureTableIDColName).Eq(
					goqu.T(ctoAlias).Col(schema.CommentTopicTableItemIDColName),
				),
				goqu.T(ctoAlias).Col(schema.CommentTopicTableTypeIDColName).Eq(schema.CommentMessageTypeIDPictures),
			)).
			Order(goqu.T(ctoAlias).Col(schema.CommentTopicTableMessagesColName).Desc())
	case OrderByViews:
		pvoAlias := alias + "pvo"
		sqSelect = sqSelect.
			LeftJoin(schema.PictureViewTable.As(pvoAlias), goqu.On(
				aliasTable.Col(schema.PictureTableIDColName).Eq(
					goqu.T(pvoAlias).Col(schema.PictureViewTablePictureIDColName),
				),
			)).
			Order(goqu.T(pvoAlias).Col(schema.PictureViewTableViewsColName).Desc())
	case OrderByModerVotes:
		if options.PictureModerVote == nil {
			return nil, false, errJoinNeededToSortByPictureModerVote
		}

		pmvAlias := query.AppendPictureModerVoteAlias(alias)
		sqSelect = sqSelect.Order(
			goqu.MAX(goqu.T(pmvAlias).Col(schema.PicturesModerVotesTableDayDateColName)).Asc(),
		)
	case OrderByRemovingDate:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableRemovingDateColName).Desc(),
			aliasTable.Col(schema.PictureTableIDColName).Asc(),
		)
	case OrderByLikes:
		pvsAlias := alias + "pvs"
		sqSelect = sqSelect.
			LeftJoin(schema.PictureVoteSummaryTable.As(pvsAlias), goqu.On(
				aliasTable.Col(schema.PictureTableIDColName).Eq(
					goqu.T(pvsAlias).Col(schema.PictureVoteSummaryTablePictureIDColName),
				),
			)).
			Order(
				goqu.T(pvsAlias).Col(schema.PictureVoteSummaryTablePositiveColName).Desc(),
				aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
				aliasTable.Col(schema.PictureTableIDColName).Desc(),
			)
	case OrderByDislikes:
		pvsAlias := alias + "pvs"
		sqSelect = sqSelect.
			LeftJoin(schema.PictureVoteSummaryTable.As(pvsAlias), goqu.On(
				aliasTable.Col(schema.PictureTableIDColName).Eq(
					goqu.T(pvsAlias).Col(schema.PictureVoteSummaryTablePictureIDColName),
				),
			)).
			Order(
				goqu.T(pvsAlias).Col(schema.PictureVoteSummaryTableNegativeColName).Desc(),
				aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
				aliasTable.Col(schema.PictureTableIDColName).Desc(),
			)
	case OrderByStatus:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableStatusColName).Asc())
	case OrderByAcceptDatetimeStrictAsc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Asc())
	case OrderByAcceptDatetimeStrictDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Desc())
	case OrderByAcceptDatetimeAsc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Asc(),
			aliasTable.Col(schema.PictureTableAddDateColName).Asc(),
			aliasTable.Col(schema.PictureTableIDColName).Asc(),
		)
	case OrderByAcceptDatetimeDesc:
		sqSelect = sqSelect.Order(
			aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Desc(),
			aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
			aliasTable.Col(schema.PictureTableIDColName).Desc(),
		)
	case OrderByDfDistanceSimilarity:
		if options.DfDistance == nil {
			return nil, false, errJoinNeededToSortByDfDistanceSimilarity
		}

		dfDistanceAlias := query.AppendDfDistanceAlias(alias)
		sqSelect = sqSelect.Order(
			goqu.MIN(goqu.T(dfDistanceAlias).Col(schema.DfDistanceTableDistanceColName)).Asc(),
		)
	case OrderByPerspectivesGroupPerspectives:
		if len(options.PictureItem) == 0 || options.PictureItem[0] == nil {
			return nil, false, errJoinNeededToSortByPerspective
		}

		var exps []exp.OrderedExpression

		piAlias := options.PictureItemAlias(alias, 0)

		if options.PictureItem[0].ItemID == 0 && options.PictureItem[0].ItemParentCacheAncestor != nil {
			if options.PictureItem[0].ItemParentCacheAncestor.ItemsByItemID == nil {
				return nil, false, errJoinNeededToSortByPerspective
			}

			ipcaAlias := options.PictureItem[0].ItemParentCacheAncestorAlias(piAlias)
			iAlias := options.PictureItem[0].ItemParentCacheAncestor.ItemsByItemIDAlias(ipcaAlias)
			exps = append(exps, goqu.MAX(goqu.T(iAlias).Col(schema.ItemTableIsConceptColName)).Asc())
			exps = append(exps, goqu.MAX(goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableSportColName)).Asc())
			exps = append(exps, goqu.MAX(goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableTuningColName)).Asc())
		}

		if options.PictureItem[0].PerspectiveGroupPerspective != nil {
			pgpAlias := query.AppendPerspectiveGroupPerspectiveAlias(piAlias)
			exps = append(exps, goqu.MAX(goqu.T(pgpAlias).Col(schema.PerspectivesGroupsPerspectivesTablePositionColName)).Asc())
		}

		exps = append([]exp.OrderedExpression{aliasTable.Col(schema.PictureTableContentCountColName).Asc()}, exps...)
		exps = append(exps,
			aliasTable.Col(schema.PictureTableWidthColName).Desc(),
			aliasTable.Col(schema.PictureTableHeightColName).Desc(),
		)

		sqSelect = sqSelect.Order(exps...)
	case OrderByPerspectives:
		if len(options.PictureItem) == 0 || options.PictureItem[0] == nil {
			return nil, false, errJoinNeededToSortByPerspective
		}

		piAlias := options.PictureItemAlias(alias, 0)

		groupBy = true
		sqSelect = sqSelect.
			LeftJoin(schema.PerspectivesTable, goqu.On(
				goqu.T(piAlias).Col(schema.PictureItemTablePerspectiveIDColName).Eq(schema.PerspectivesTableIDCol),
			)).
			Order(
				goqu.MIN(schema.PerspectivesTablePositionCol).Asc(),
				aliasTable.Col(schema.PictureTableWidthColName).Desc(),
				aliasTable.Col(schema.PictureTableHeightColName).Desc(),
				aliasTable.Col(schema.PictureTableAddDateColName).Desc(),
				aliasTable.Col(schema.PictureTableIDColName).Desc(),
			)
	case OrderByTopPerspectives, OrderByBottomPerspectives, OrderByFrontPerspectives:
		if len(options.PictureItem) == 0 || options.PictureItem[0] == nil {
			return nil, false, errJoinNeededToSortByPerspective
		}

		perspectives := specsTopPerspectives
		if order == OrderByBottomPerspectives {
			perspectives = specsBottomPerspectives
		} else if order == OrderByFrontPerspectives {
			perspectives = frontPerspectives
		}

		orderExprs := make([]exp.OrderedExpression, 0, len(perspectives))
		piAlias := options.PictureItemAlias(alias, 0)

		for _, pid := range perspectives {
			var expr exp.Comparable = goqu.T(piAlias).Col(schema.PictureItemTablePerspectiveIDColName)

			if groupBy {
				expr = goqu.MAX(expr)
			}

			orderExprs = append(orderExprs, goqu.L("?", expr.Eq(pid)).Desc())
		}

		sqSelect = sqSelect.Order(orderExprs...)

	case OrderByIDDesc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableIDColName).Desc())

	case OrderByIDAsc:
		sqSelect = sqSelect.Order(aliasTable.Col(schema.PictureTableIDColName).Asc())

	case OrderByNone:
	}

	return sqSelect, groupBy, nil
}

func (s *Repository) PictureSelect(
	options *query.PictureListOptions, _ *PictureFields, order OrderBy,
) (*goqu.SelectDataset, error) {
	var (
		err        error
		alias      = query.PictureAlias
		aliasTable = goqu.T(alias)
	)

	sqSelect, err := options.Select(s.db, alias)
	if err != nil {
		return nil, err
	}

	sqSelect = sqSelect.Select(
		aliasTable.Col(schema.PictureTableIDColName),
		aliasTable.Col(schema.PictureTableOwnerIDColName),
		aliasTable.Col(schema.PictureTableChangeStatusUserIDColName),
		aliasTable.Col(schema.PictureTableIdentityColName),
		aliasTable.Col(schema.PictureTableStatusColName),
		aliasTable.Col(schema.PictureTableImageIDColName),
		aliasTable.Col(schema.PictureTablePointColName),
		aliasTable.Col(schema.PictureTableCopyrightsTextIDColName),
		aliasTable.Col(schema.PictureTableAcceptDatetimeColName),
		aliasTable.Col(schema.PictureTableReplacePictureIDColName),
		aliasTable.Col(schema.PictureTableWidthColName),
		aliasTable.Col(schema.PictureTableHeightColName),
		aliasTable.Col(schema.PictureTableNameColName),
		aliasTable.Col(schema.PictureTableAddDateColName),
		aliasTable.Col(schema.PictureTableTakenDayColName),
		aliasTable.Col(schema.PictureTableTakenMonthColName),
		aliasTable.Col(schema.PictureTableTakenYearColName),
		aliasTable.Col(schema.PictureTableIPColName),
		aliasTable.Col(schema.PictureTableDPIXColName),
		aliasTable.Col(schema.PictureTableDPIYColName),
	)

	groupBy := !options.IsIDUnique()

	sqSelect, groupBy, err = s.orderBy(sqSelect, options, order, groupBy)
	if err != nil {
		return nil, err
	}

	if groupBy {
		sqSelect = sqSelect.GroupBy(aliasTable.Col(schema.PictureTableIDColName))
	}

	return sqSelect, nil
}

func (s *Repository) Exists(ctx context.Context, options *query.PictureListOptions) (bool, error) {
	sqSelect, err := s.PictureSelect(options, nil, OrderByNone)
	if err != nil {
		return false, fmt.Errorf("PictureSelect(): %w", err)
	}

	exists := false

	success, err := s.db.Select(goqu.L("EXISTS ?", sqSelect.Select())).ScanValContext(ctx, &exists)

	return success && exists, err
}

func (s *Repository) PicturesPaginator(
	options *query.PictureListOptions, fields *PictureFields, order OrderBy,
) (*util.Paginator, error) {
	sqSelect, err := s.PictureSelect(options, fields, order)
	if err != nil {
		return nil, fmt.Errorf("PictureSelect(): %w", err)
	}

	return &util.Paginator{
		SQLSelect:         sqSelect,
		ItemCountPerPage:  int32(options.Limit), //nolint: gosec
		CurrentPageNumber: int32(options.Page),  //nolint: gosec
	}, nil
}

func (s *Repository) Pictures(
	ctx context.Context, options *query.PictureListOptions, fields *PictureFields, order OrderBy, pagination bool,
) ([]*schema.PictureRow, *util.Pages, error) {
	var (
		sqSelect *goqu.SelectDataset
		pages    *util.Pages
		err      error
		res      []*schema.PictureRow
	)

	if pagination {
		paginator, err := s.PicturesPaginator(options, fields, order)
		if err != nil {
			return nil, nil, fmt.Errorf("PicturesPaginator(): %w", err)
		}

		pages, err = paginator.GetPages(ctx)
		if err != nil {
			return nil, nil, err
		}

		sqSelect, err = paginator.GetCurrentItems(ctx)
		if err != nil {
			return nil, nil, err
		}
	} else {
		sqSelect, err = s.PictureSelect(options, fields, order)
		if err != nil {
			return nil, nil, fmt.Errorf("PictureSelect(): %w", err)
		}

		if options.Limit > 0 {
			sqSelect = sqSelect.Limit(uint(options.Limit))
		}
	}

	err = sqSelect.ScanStructsContext(ctx, &res)

	return res, pages, err
}

func (s *Repository) Picture(
	ctx context.Context, options *query.PictureListOptions, fields *PictureFields, order OrderBy,
) (*schema.PictureRow, error) {
	options.Limit = 1

	rows, _, err := s.Pictures(ctx, options, fields, order, false)
	if err != nil {
		return nil, fmt.Errorf("Pictures(): %w", err)
	}

	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}

	return rows[0], nil
}

func (s *Repository) Normalize(ctx context.Context, id int64) error {
	if id == 0 {
		return sql.ErrNoRows
	}

	pic, err := s.Picture(ctx, &query.PictureListOptions{ID: id}, nil, OrderByNone)
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
	if id == 0 {
		return sql.ErrNoRows
	}

	pic, err := s.Picture(ctx, &query.PictureListOptions{ID: id}, nil, OrderByNone)
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
	if pictureItemType != schema.PictureItemTypeContent {
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
	if pictureItemType != schema.PictureItemTypeContent {
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
		schema.ItemTableItemTypeIDBrand:    {schema.PictureItemTypeContent, schema.PictureItemTypeCopyrights},
		schema.ItemTableItemTypeIDCategory: {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDEngine:   {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDFactory:  {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDVehicle:  {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDTwins:    {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDMuseum:   {schema.PictureItemTypeContent},
		schema.ItemTableItemTypeIDPerson: {
			schema.PictureItemTypeContent, schema.PictureItemTypeAuthor, schema.PictureItemTypeCopyrights,
		},
		schema.ItemTableItemTypeIDCopyright: {schema.PictureItemTypeCopyrights},
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
					schema.PictureItemTableTypeCol.Eq(schema.PictureItemTypeContent),
				),
		}).
		Where(schema.PictureTableIDCol.Eq(pictureID)).
		Executor().ExecContext(ctx)

	return err
}

func (s *Repository) SetPictureCrop(ctx context.Context, pictureID int64, area sampler.Crop) error {
	if pictureID == 0 {
		return sql.ErrNoRows
	}

	pic, err := s.Picture(ctx, &query.PictureListOptions{ID: pictureID}, nil, OrderByNone)
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
	if pictureID == 0 {
		return false, 0, sql.ErrNoRows
	}

	picture, err := s.Picture(ctx, &query.PictureListOptions{ID: pictureID}, nil, OrderByNone)
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

	if pictureID == 0 {
		return false, false, sql.ErrNoRows
	}

	picture, err := s.Picture(ctx, &query.PictureListOptions{ID: pictureID}, nil, OrderByNone)
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

func (s *Repository) PictureItemSelect(options *query.PictureItemListOptions) (*goqu.SelectDataset, error) {
	alias := query.PictureItemAlias
	aliasTable := goqu.T(alias)

	sqSelect, err := options.Select(s.db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(
		aliasTable.Col(schema.PictureItemTablePictureIDColName),
		aliasTable.Col(schema.PictureItemTableItemIDColName),
		aliasTable.Col(schema.PictureItemTableTypeColName),
		aliasTable.Col(schema.PictureItemTableCropLeftColName),
		aliasTable.Col(schema.PictureItemTableCropTopColName),
		aliasTable.Col(schema.PictureItemTableCropWidthColName),
		aliasTable.Col(schema.PictureItemTableCropHeightColName),
		aliasTable.Col(schema.PictureItemTablePerspectiveIDColName),
	), nil
}

func (s *Repository) PictureItem(
	ctx context.Context, options *query.PictureItemListOptions,
) (*schema.PictureItemRow, error) {
	var row schema.PictureItemRow

	sqSelect, err := s.PictureItemSelect(options)
	if err != nil {
		return nil, err
	}

	success, err := sqSelect.Limit(1).ScanStructContext(ctx, &row)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, sql.ErrNoRows
	}

	return &row, nil
}

func (s *Repository) PictureItemsBatch(
	ctx context.Context, options []*query.PictureItemListOptions, limit uint32,
) ([]*schema.PictureItemRow, error) {
	var (
		rows     []*schema.PictureItemRow
		sqSelect *goqu.SelectDataset
		err      error
	)

	for _, cOptions := range options {
		prev := sqSelect

		sqSelect, err = s.PictureItemSelect(cOptions)
		if err != nil {
			return nil, err
		}

		if limit > 0 {
			sqSelect = sqSelect.Limit(uint(limit))
		}

		if prev != nil {
			sqSelect = prev.UnionAll(sqSelect)
		}
	}

	if sqSelect == nil {
		return rows, nil
	}

	err = sqSelect.ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) PictureItems(
	ctx context.Context, options *query.PictureItemListOptions, order PictureItemOrderBy, limit uint32,
) ([]*schema.PictureItemRow, error) {
	var rows []*schema.PictureItemRow

	sqSelect, err := s.PictureItemSelect(options)
	if err != nil {
		return nil, err
	}

	switch order {
	case PictureItemOrderByNone:
	case PictureItemOrderByFrontPerspectivesFirst:
		perspectives := frontPerspectives

		orderExprs := make([]exp.OrderedExpression, 0, len(perspectives))

		for _, pid := range perspectives {
			orderExprs = append(orderExprs, goqu.L("?",
				goqu.T(query.PictureItemAlias).Col(schema.PictureItemTablePerspectiveIDColName).Eq(pid)).Desc())
		}

		sqSelect = sqSelect.Order(orderExprs...)
	}

	if limit > 0 {
		sqSelect = sqSelect.Limit(uint(limit))
	}

	err = sqSelect.ScanStructsContext(ctx, &rows)

	return rows, err
}

type NameDataOptions struct {
	Language string
}

func (s *Repository) NameData(
	ctx context.Context, rows []*schema.PictureRow, options NameDataOptions,
) (map[int64]PictureNameFormatterOptions, error) {
	var (
		result = make(map[int64]PictureNameFormatterOptions, len(rows))
		// prefetch
		itemIDs        = make(map[int64]int32)
		perspectiveIDs = make(map[int64]bool)
	)

	for _, row := range rows {
		var pictureItemRows []schema.PictureItemRow

		err := s.db.Select(
			schema.PictureItemTableItemIDCol, schema.PictureItemTableCropLeftCol,
			schema.PictureItemTablePerspectiveIDCol,
		).
			From(schema.PictureItemTable).
			Where(
				schema.PictureItemTablePictureIDCol.Eq(row.ID),
				schema.PictureItemTableTypeCol.Eq(schema.PictureItemTypeContent),
			).
			ScanStructsContext(ctx, &pictureItemRows)
		if err != nil {
			return nil, err
		}

		for _, pictureItemRow := range pictureItemRows {
			itemIDs[pictureItemRow.ItemID] = util.NullInt32ToScalar(pictureItemRow.CropLeft)

			if pictureItemRow.PerspectiveID.Valid && util.Contains(prefixedPerspectives, pictureItemRow.PerspectiveID.Int64) {
				perspectiveIDs[pictureItemRow.PerspectiveID.Int64] = true
			}
		}
	}

	itemsCache := make(map[int64]items.ItemNameFormatterOptions)

	if len(itemIDs) > 0 {
		itemRows, _, err := s.itemsRepository.List(ctx, &query.ItemListOptions{
			ItemIDs:  slices.Collect(maps.Keys(itemIDs)),
			Language: options.Language,
		}, &items.ListFields{
			NameOnly: true,
			NameText: true,
			NameHTML: true,
		}, items.OrderByNone, false)
		if err != nil {
			return nil, err
		}

		for _, row := range itemRows {
			itemsCache[row.ID] = items.ItemNameFormatterOptions{
				BeginModelYear:         util.NullInt32ToScalar(row.BeginModelYear),
				EndModelYear:           util.NullInt32ToScalar(row.EndModelYear),
				BeginModelYearFraction: util.NullStringToString(row.BeginModelYearFraction),
				EndModelYearFraction:   util.NullStringToString(row.EndModelYearFraction),
				Spec:                   row.SpecShortName,
				SpecFull:               row.SpecName,
				Body:                   row.Body,
				Name:                   row.NameOnly,
				BeginYear:              util.NullInt32ToScalar(row.BeginYear),
				EndYear:                util.NullInt32ToScalar(row.EndYear),
				Today:                  util.NullBoolToBoolPtr(row.Today),
				BeginMonth:             util.NullInt16ToScalar(row.BeginMonth),
				EndMonth:               util.NullInt16ToScalar(row.EndMonth),
			}
		}
	}

	perspectives, err := s.PerspectivesPairs(ctx, slices.Collect(maps.Keys(perspectiveIDs)))
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.Name.Valid && row.Name.String != "" {
			result[row.ID] = PictureNameFormatterOptions{
				Name: row.Name.String,
			}

			continue
		}

		pictureItemRows, err := s.PictureItems(ctx, &query.PictureItemListOptions{
			PictureID: row.ID,
			TypeID:    schema.PictureItemTypeContent,
		}, PictureItemOrderByNone, 0)
		if err != nil {
			return nil, err
		}

		slices.SortFunc(pictureItemRows, func(rowA, rowB *schema.PictureItemRow) int {
			cropLeftA, ok := itemIDs[rowA.ItemID]
			if !ok {
				cropLeftA = 0
			}

			cropLeftB, ok := itemIDs[rowB.ItemID]
			if !ok {
				cropLeftB = 0
			}

			if cropLeftA == cropLeftB {
				return 0
			}

			if cropLeftA < cropLeftB {
				return -1
			}

			return 1
		})

		resultItems := make([]PictureNameFormatterItem, 0)

		for _, pictureItemRow := range pictureItemRows {
			itemID := pictureItemRow.ItemID
			perspectiveID := pictureItemRow.PerspectiveID

			item, ok := itemsCache[itemID]
			if !ok {
				item = items.ItemNameFormatterOptions{}
			}

			perspective := ""

			if perspectiveID.Valid {
				if val, ok := perspectives[perspectiveID.Int64]; ok {
					perspective = val
				}
			}

			resultItems = append(resultItems, PictureNameFormatterItem{
				Item:        item,
				Perspective: perspective,
			})
		}

		result[row.ID] = PictureNameFormatterOptions{
			Items: resultItems,
		}
	}

	return result, nil
}

func (s *Repository) PerspectivesPairs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(ids))

	if len(ids) == 0 {
		return result, nil
	}

	var rows []schema.PerspectiveRow

	err := s.db.Select(schema.PerspectivesTableIDCol, schema.PerspectivesTableNameCol).
		From(schema.PerspectivesTable).
		Where(schema.PerspectivesTableIDCol.In(ids)).
		Order(schema.PerspectivesTablePositionCol.Asc()).
		ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ID] = row.Name
	}

	return result, nil
}

func (s *Repository) DfDistanceSelect(options *query.DfDistanceListOptions) (*goqu.SelectDataset, error) {
	alias := query.DfDistanceAlias
	aliasTable := goqu.T(alias)

	sqSelect, err := options.Select(s.db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(
		aliasTable.Col(schema.DfDistanceTableSrcPictureIDColName),
		aliasTable.Col(schema.DfDistanceTableDstPictureIDColName),
		aliasTable.Col(schema.DfDistanceTableDistanceColName),
	), nil
}

func (s *Repository) DfDistances(
	ctx context.Context, options *query.DfDistanceListOptions, limit uint32,
) ([]*schema.DfDistanceRow, error) {
	var rows []*schema.DfDistanceRow

	sqSelect, err := s.DfDistanceSelect(options)
	if err != nil {
		return nil, err
	}

	err = sqSelect.Limit(uint(limit)).ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) PictureModerVoteSelect(options *query.PictureModerVoteListOptions) *goqu.SelectDataset {
	alias := query.PictureModerVoteAlias
	aliasTable := goqu.T(alias)

	return options.Select(s.db, alias).Select(
		aliasTable.Col(schema.PicturesModerVotesTablePictureIDColName),
		aliasTable.Col(schema.PicturesModerVotesTableUserIDColName),
		aliasTable.Col(schema.PicturesModerVotesTableVoteColName),
		aliasTable.Col(schema.PicturesModerVotesTableReasonColName),
	)
}

func (s *Repository) PictureModerVotes(
	ctx context.Context, options *query.PictureModerVoteListOptions,
) ([]*schema.PictureModerVoteRow, error) {
	var rows []*schema.PictureModerVoteRow

	err := s.PictureModerVoteSelect(options).ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) PerspectivePageGroupIDs(
	ctx context.Context, pageID int32,
) ([]int32, error) {
	var ids []int32

	err := s.db.Select(schema.PerspectivesGroupsTableIDCol).
		From(schema.PerspectivesGroupsTable).
		Where(schema.PerspectivesGroupsTablePageIDCol.Eq(pageID)).
		Order(schema.PerspectivesGroupsTablePositionCol.Asc()).
		ScanValsContext(ctx, &ids)

	return ids, err
}
