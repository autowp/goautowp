package itemofday

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/sirupsen/logrus"
)

const (
	colItemID               = "item_id"
	colUserID               = "user_id"
	colDayDate              = "day_date"
	defaultMinPictures      = 3
	YoomoneyLabelDateFormat = time.DateOnly
)

type Repository struct {
	db          *goqu.Database
	loc         *time.Location
	minPictures int
}

type NextDate struct {
	Date time.Time
	Free bool
}

type CandidateRecord struct {
	ItemID int64 `db:"id"`
	Count  int64 `db:"p_count"`
}

func NewRepository(db *goqu.Database) *Repository {
	loc, _ := time.LoadLocation("UTC")

	return &Repository{
		db:          db,
		loc:         loc,
		minPictures: defaultMinPictures,
	}
}

func (s *Repository) SetMinPictures(value int) {
	s.minPictures = value
}

func (s *Repository) NextDates(ctx context.Context) ([]NextDate, error) {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, s.loc)

	result := make([]NextDate, 0)

	for i := 0; i < 10; i++ {
		found := false

		_, err := s.db.Select(goqu.L("1")).From(schema.TableOfDay).Where(
			goqu.I(colDayDate).Eq(now.Format(time.DateOnly)),
			goqu.I(colItemID).IsNotNull(),
		).ScanValContext(ctx, &found)
		if err != nil {
			return nil, err
		}

		result = append(result, NextDate{
			Date: now,
			Free: !found,
		})

		now = now.AddDate(0, 0, 1)
	}

	return result, nil
}

func (s *Repository) IsAvailableDate(ctx context.Context, date time.Time) (bool, error) {
	dateStr := date.Format(time.DateOnly)

	nextDates, err := s.NextDates(ctx)
	if err != nil {
		return false, err
	}

	for _, nextDate := range nextDates {
		if nextDate.Date.Format(time.DateOnly) == dateStr {
			return true, nil
		}
	}

	return false, nil
}

func (s *Repository) Pick(ctx context.Context) (bool, error) {
	itemID, err := s.candidate(ctx)
	if err != nil {
		return false, err
	}

	if itemID <= 0 {
		logrus.Warning("ItemOfDay: candidate not found")

		return false, nil
	}

	logrus.Infof("ItemOfDay: candidate is `%d`", itemID)

	return s.SetItemOfDay(ctx, time.Now(), itemID, 0)
}

func (s *Repository) candidate(ctx context.Context) (int64, error) {
	sqSelect := s.CandidateQuery().
		Where(goqu.L(
			schema.ItemTableName + ".begin_year AND " + schema.ItemTableName + ".end_year OR " +
				schema.ItemTableName + ".begin_model_year AND " + schema.ItemTableName + ".end_model_year",
		)).
		Order(goqu.Func("RAND").Desc()).
		Limit(1)

	r := CandidateRecord{}

	success, err := sqSelect.Executor().ScanStructContext(ctx, &r)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return r.ItemID, nil
}

func (s *Repository) CandidateQuery() *goqu.SelectDataset {
	pTable := goqu.T(schema.TablePicture)
	ipcTable := goqu.T(schema.TableItemParentCache)
	piTable := goqu.T(schema.TablePictureItem)

	table := goqu.T(schema.TableOfDay)
	tableItemIDCol := table.Col(colItemID)

	const picturesCountAlias = "p_count"

	sqSelect := s.db.Select(
		schema.ItemTableColID,
		goqu.COUNT(goqu.DISTINCT(pTable.Col("id"))).As(picturesCountAlias),
	).
		From(schema.ItemTable).
		Join(ipcTable, goqu.On(schema.ItemTableColID.Eq(ipcTable.Col("parent_id")))).
		Join(piTable, goqu.On(ipcTable.Col("item_id").Eq(piTable.Col("item_id")))).
		Join(pTable, goqu.On(piTable.Col("picture_id").Eq(pTable.Col("id")))).
		Where(
			pTable.Col("status").Eq(pictures.StatusAccepted),
			schema.ItemTableColID.NotIn(
				s.db.Select(tableItemIDCol).From(table).Where(tableItemIDCol.IsNotNull()),
			),
		).
		GroupBy(schema.ItemTableColID).
		Having(goqu.I(picturesCountAlias).Gte(s.minPictures))

	return sqSelect
}

func (s *Repository) IsComplies(ctx context.Context, itemID int64) (bool, error) {
	if itemID == 0 {
		return false, errors.New("itemID must be defined")
	}

	sqSelect := s.CandidateQuery().Where(schema.ItemTableColID.Eq(itemID))

	r := CandidateRecord{}

	success, err := sqSelect.Executor().ScanStructContext(ctx, &r)
	if err != nil {
		return false, err
	}

	if !success {
		return false, nil
	}

	return r.ItemID != 0, nil
}

func (s *Repository) SetItemOfDay(ctx context.Context, dateTime time.Time, itemID int64, userID int64) (bool, error) {
	isComplies, err := s.IsComplies(ctx, itemID)
	if err != nil {
		return false, err
	}

	if !isComplies {
		return false, nil
	}

	table := goqu.T(schema.TableOfDay)

	dateStr := dateTime.Format(time.DateOnly)
	dateExpr := goqu.I(colDayDate).Eq(dateStr)

	sqSelect := s.db.Select(goqu.L(colItemID)).From(table).Where(dateExpr)

	var exists int64

	success, err := sqSelect.ScanValContext(ctx, &exists)
	if err != nil {
		return false, err
	}

	if success && exists > 0 {
		return false, nil
	}

	userIDVal := sql.NullInt64{
		Int64: userID,
		Valid: userID > 0,
	}

	if success {
		_, err = s.db.Update(table).Set(
			goqu.Record{colItemID: itemID, colUserID: userIDVal},
		).Where(dateExpr).Executor().Exec()
	} else {
		_, err = s.db.Insert(table).Rows(
			goqu.Record{colItemID: itemID, colUserID: userIDVal, colDayDate: dateStr},
		).Executor().Exec()
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
