package query

import (
	"errors"
	"time"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

var errNoTimezone = errors.New("timezone not provided")

const (
	PictureAlias = "p"
)

func AppendPictureAlias(alias string) string {
	return alias + "_" + PictureAlias
}

type PictureListOptions struct {
	Status                schema.PictureStatus
	Statuses              []schema.PictureStatus
	OwnerID               int64
	PictureItem           *PictureItemListOptions
	ID                    int64
	IDGt                  int64
	IDLt                  int64
	ExcludeID             int64
	HasCopyrights         bool
	Limit                 uint32
	Page                  uint32
	AcceptedInDays        int32
	AddDate               *util.Date
	AcceptDate            *util.Date
	AddedFrom             *util.Date
	Timezone              *time.Location
	Identity              string
	CommentTopic          *CommentTopicListOptions
	HasNoComments         bool
	HasPoint              bool
	HasNoPoint            bool
	HasNoPictureItem      bool
	ReplacePicture        *PictureListOptions
	HasNoReplacePicture   bool
	PictureModerVote      *PictureModerVoteListOptions
	HasNoPictureModerVote bool
	DfDistance            *DfDistanceListOptions
	HasSpecialName        bool
}

func (s *PictureListOptions) Select(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	return s.apply(
		alias,
		db.Select().From(schema.PictureTable.As(alias)),
	)
}

func (s *PictureListOptions) CountSelect(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db, alias)
	if err != nil {
		return nil, err
	}

	if s.IsIDUnique() {
		return sqSelect.Select(goqu.COUNT(goqu.Star())), nil
	}

	return sqSelect.Select(
		goqu.COUNT(goqu.DISTINCT(goqu.T(alias).Col(schema.PictureTableIDColName))),
	), nil
}

func (s *PictureListOptions) IsIDUnique() bool {
	return (s.PictureItem == nil || s.PictureItem.IsPictureIDUnique()) && s.PictureModerVote == nil && s.DfDistance == nil
}

func (s *PictureListOptions) JoinToIDAndApply(
	srcCol exp.IdentifierExpression, alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, error) {
	if s == nil {
		return sqSelect, nil
	}

	return s.apply(
		alias,
		sqSelect.Join(
			schema.PictureTable.As(alias),
			goqu.On(
				srcCol.Eq(goqu.T(alias).Col(schema.PictureTableIDColName)),
			),
		),
	)
}

func (s *PictureListOptions) apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	var (
		err        error
		aliasTable = goqu.T(alias)
		idCol      = aliasTable.Col(schema.PictureTableIDColName)
	)

	if s.ID != 0 {
		sqSelect = sqSelect.Where(idCol.Eq(s.ID))
	}

	if s.IDGt != 0 {
		sqSelect = sqSelect.Where(idCol.Gt(s.IDGt))
	}

	if s.IDLt != 0 {
		sqSelect = sqSelect.Where(idCol.Lt(s.IDLt))
	}

	if s.ExcludeID != 0 {
		sqSelect = sqSelect.Where(idCol.Neq(s.ExcludeID))
	}

	if s.Identity != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableIdentityColName).Eq(s.Identity))
	}

	if s.Status != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableStatusColName).Eq(s.Status))
	}

	if len(s.Statuses) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableStatusColName).In(s.Statuses))
	}

	if s.OwnerID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableOwnerIDColName).Eq(s.OwnerID))
	}

	sqSelect, err = s.PictureItem.JoinToPictureIDAndApply(
		idCol,
		AppendPictureItemAlias(alias),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	if s.HasCopyrights {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableCopyrightsTextIDColName).IsNotNull())
	}

	if s.AcceptedInDays > 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Gt(
				goqu.Func("DATE_SUB", goqu.Func("CURDATE"), goqu.L("INTERVAL ? DAY", s.AcceptedInDays)),
			),
		)
	}

	if s.AddDate != nil {
		sqSelect, err = s.setDateFilter(
			sqSelect, aliasTable.Col(schema.PictureTableAddDateColName), *s.AddDate, s.Timezone,
		)
		if err != nil {
			return nil, err
		}
	}

	if s.AcceptDate != nil {
		sqSelect, err = s.setDateFilter(
			sqSelect, aliasTable.Col(schema.PictureTableAcceptDatetimeColName), *s.AcceptDate, s.Timezone,
		)
		if err != nil {
			return nil, err
		}
	}

	if s.AddedFrom != nil {
		if s.Timezone == nil {
			return nil, errNoTimezone
		}

		start := time.Date(s.AddedFrom.Year, s.AddedFrom.Month, s.AddedFrom.Day, 0, 0, 0, 0, s.Timezone).In(time.UTC)

		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.PictureTableAddDateColName).Gte(start.Format(time.DateTime)),
		)
	}

	if s.CommentTopic != nil {
		s.CommentTopic.TypeID = schema.CommentMessageTypeIDPictures

		sqSelect = s.CommentTopic.JoinToItemIDAndApply(
			idCol,
			AppendCommentTopicAlias(alias),
			sqSelect,
		)
	}

	if s.HasNoComments {
		ctAlias := alias + "no_cm"
		ctAliasTable := goqu.T(ctAlias)

		sqSelect = sqSelect.LeftJoin(
			schema.CommentTopicTable.As(ctAlias),
			goqu.On(
				idCol.Eq(ctAliasTable.Col(schema.CommentTopicTableItemIDColName)),
				ctAliasTable.Col(schema.CommentTopicTableTypeIDColName).Eq(schema.CommentMessageTypeIDPictures),
			),
		).Where(
			goqu.Or(
				ctAliasTable.Col(schema.CommentTopicTableItemIDColName).IsNull(),
				ctAliasTable.Col(schema.CommentTopicTableMessagesColName).Eq(0),
			),
		)
	}

	if s.HasPoint {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTablePointColName).IsNotNull())
	}

	if s.HasNoPoint {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTablePointColName).IsNull())
	}

	if s.HasNoPictureItem {
		piAlias := alias + "no_pi"
		piAliasTable := goqu.T(piAlias)

		sqSelect = sqSelect.LeftJoin(
			schema.PictureItemTable.As(piAlias),
			goqu.On(idCol.Eq(piAliasTable.Col(schema.PictureItemTableItemIDColName))),
		).Where(piAliasTable.Col(schema.PictureItemTableItemIDColName).IsNull())
	}

	sqSelect, err = s.ReplacePicture.JoinToIDAndApply(
		aliasTable.Col(schema.PictureTableReplacePictureIDColName),
		AppendPictureAlias(alias),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	if s.HasNoReplacePicture {
		pAlias := alias + "no_p"
		pAliasTable := goqu.T(pAlias)

		sqSelect = sqSelect.LeftJoin(
			schema.PictureTable.As(pAlias),
			goqu.On(aliasTable.Col(schema.PictureTableReplacePictureIDColName).Eq(
				pAliasTable.Col(schema.PictureTableIDColName),
			)),
		).Where(pAliasTable.Col(schema.PictureTableIDColName).IsNull())
	}

	sqSelect = s.PictureModerVote.JoinToPictureIDAndApply(
		idCol,
		AppendPictureModerVoteAlias(alias),
		sqSelect,
	)

	if s.HasNoPictureModerVote {
		pmvAlias := alias + "no_pmv"
		pmvAliasTable := goqu.T(pmvAlias)

		sqSelect = sqSelect.LeftJoin(
			schema.PicturesModerVotesTable.As(pmvAlias),
			goqu.On(idCol.Eq(pmvAliasTable.Col(schema.PicturesModerVotesTablePictureIDColName))),
		).Where(pmvAliasTable.Col(schema.PicturesModerVotesTablePictureIDColName).IsNull())
	}

	sqSelect, err = s.DfDistance.JoinToSrcPictureIDAndApply(
		idCol,
		AppendDfDistanceAlias(alias),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	if s.HasSpecialName {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.PictureTableNameColName).IsNotNull(),
			aliasTable.Col(schema.PictureTableNameColName).Neq(""),
		)
	}

	return sqSelect, nil
}

func (s *PictureListOptions) setDateFilter(
	sqSelect *goqu.SelectDataset, column exp.IdentifierExpression, date util.Date, timezone *time.Location,
) (*goqu.SelectDataset, error) {
	if s.Timezone == nil {
		return nil, errNoTimezone
	}

	start := time.Date(date.Year, date.Month, date.Day, 0, 0, 0, 0, timezone).In(time.UTC)
	end := time.Date(date.Year, date.Month, date.Day, 23, 59, 59, 999, timezone).In(time.UTC)

	return sqSelect.Where(
		column.Between(goqu.Range(start.Format(time.DateTime), end.Format(time.DateTime))),
	), nil
}
