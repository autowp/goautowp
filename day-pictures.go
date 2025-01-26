package goautowp

import (
	"context"
	"database/sql"
	"time"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

const externalDateFormat = time.DateOnly

type DayPictures struct {
	currentDate        time.Time
	listOptions        *query.PictureListOptions
	timezone           *time.Location
	minDate            time.Time
	dbTimezone         *time.Location
	dbDateTimeFormat   string
	prevDate           time.Time
	nextDate           time.Time
	picturesRepository *pictures.Repository
}

func (s *DayPictures) HaveCurrentDate() bool {
	return !s.currentDate.IsZero()
}

func NewDayPictures(
	picturesRepository *pictures.Repository, timezone *time.Location, listOptions *query.PictureListOptions,
	currentDate time.Time,
) (*DayPictures, error) {
	return &DayPictures{
		timezone:           timezone,
		listOptions:        listOptions,
		dbTimezone:         time.UTC,
		dbDateTimeFormat:   time.DateTime,
		currentDate:        currentDate,
		picturesRepository: picturesRepository,
	}, nil
}

func (s *DayPictures) HaveCurrentDayPictures(ctx context.Context) (bool, error) {
	if s.currentDate.IsZero() {
		return false, nil
	}

	total, err := s.CurrentDateCount(ctx)
	if err != nil {
		return false, err
	}

	return total > 0, nil
}

func (s *DayPictures) LastDateStr(ctx context.Context) (string, error) {
	listOptions := *s.listOptions
	listOptions.Timezone = s.timezone

	lastDate, err := s.calcDate(ctx, &listOptions, pictures.OrderByAddDateDesc)
	if err != nil {
		return "", err
	}

	return lastDate.In(s.timezone).Format(externalDateFormat), nil
}

func (s *DayPictures) SetCurrentDate(date string) error {
	var (
		dateObj = time.Time{}
		err     error
	)

	if date != "" {
		dateObj, err = time.ParseInLocation(externalDateFormat, date, s.timezone)
		if err != nil {
			return err
		}
	}

	s.currentDate = dateObj

	s.reset()

	return nil
}

func (s *DayPictures) reset() {
	s.nextDate = time.Time{}
	s.prevDate = time.Time{}
}

func (s *DayPictures) endOfDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999, date.Location())
}

func (s *DayPictures) startOfDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func (s *DayPictures) PrevDate(ctx context.Context) (time.Time, error) {
	err := s.calcPrevDate(ctx)

	return s.prevDate, err
}

func (s *DayPictures) NextDate(ctx context.Context) (time.Time, error) {
	err := s.calcNextDate(ctx)

	return s.nextDate, err
}

func (s *DayPictures) calcDate(
	ctx context.Context, listOptions *query.PictureListOptions, orderBy pictures.OrderBy,
) (time.Time, error) {
	sqSelect, err := s.picturesRepository.PictureSelect(listOptions, pictures.PictureFields{}, orderBy)
	if err != nil {
		return time.Time{}, err
	}

	var date sql.NullTime

	success, err := sqSelect.Select(goqu.T(query.PictureAlias).Col(schema.PictureTableAddDateColName)).
		Limit(1).
		ScanValContext(ctx, &date)
	if err != nil {
		return time.Time{}, err
	}

	if success && date.Valid {
		return date.Time.In(s.timezone), nil
	}

	return time.Time{}, nil
}

func (s *DayPictures) calcPrevDate(ctx context.Context) error {
	if s.currentDate.IsZero() {
		return nil
	}

	if !s.prevDate.IsZero() {
		return nil
	}

	listOptions := *s.listOptions
	listOptions.Timezone = s.timezone
	listOptions.AddDateLt = util.TimePtr(s.startOfDay(s.currentDate))

	if !s.minDate.IsZero() {
		listOptions.AddDateGte = util.TimePtr(s.startOfDay(s.minDate))
	}

	var err error

	s.prevDate, err = s.calcDate(ctx, &listOptions, pictures.OrderByAddDateDesc)

	return err
}

func (s *DayPictures) calcNextDate(ctx context.Context) error {
	if s.currentDate.IsZero() {
		return nil
	}

	if !s.nextDate.IsZero() {
		return nil
	}

	listOptions := *s.listOptions
	listOptions.Timezone = s.timezone
	listOptions.AddDateGt = util.TimePtr(s.endOfDay(s.currentDate))

	var err error

	s.nextDate, err = s.calcDate(ctx, &listOptions, pictures.OrderByAddDateAsc)

	return err
}

func (s *DayPictures) CurrentDate() time.Time {
	return s.currentDate
}

func (s *DayPictures) PrevDateCount(ctx context.Context) (int32, error) {
	err := s.calcPrevDate(ctx)
	if err != nil {
		return 0, err
	}

	if s.prevDate.IsZero() {
		return 0, nil
	}

	return s.dateCount(ctx, s.prevDate)
}

func (s *DayPictures) CurrentDateCount(ctx context.Context) (int32, error) {
	if s.currentDate.IsZero() {
		return 0, nil
	}

	return s.dateCount(ctx, s.currentDate)
}

func (s *DayPictures) NextDateCount(ctx context.Context) (int32, error) {
	err := s.calcNextDate(ctx)
	if err != nil {
		return 0, err
	}

	if s.nextDate.IsZero() {
		return 0, nil
	}

	return s.dateCount(ctx, s.nextDate)
}

func (s *DayPictures) dateCount(ctx context.Context, date time.Time) (int32, error) {
	d := util.TimeToDate(date)

	listOptions := *s.listOptions
	listOptions.AddDate = &d
	listOptions.Timezone = s.timezone

	res, err := s.picturesRepository.Count(ctx, &listOptions)

	return int32(res), err //nolint: gosec
}
