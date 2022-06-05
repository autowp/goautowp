package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"github.com/autowp/goautowp/config"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
	"time"
)

var colors = []string{
	"#FF0000",
	"#00FF00",
	"#0000FF",
	"#FFFF00",
	"#FF00FF",
	"#00FFFF",
	"#880000",
	"#008800",
	"#000088",
	"#888800",
	"#880088",
	"#008888",
}

const thousands = 1000
const tensOfThousands = 10 * thousands
const numberOfTopUploadersToShowInAboutUs = 20

var failedToFetchRow = errors.New("failed to fetch row")

func roundTo(value int32, to int32) int32 {
	var rest = value % to
	if rest > to/2 {
		value = value - rest + to
	} else {
		value -= rest
	}

	return value
}

func unique(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true

			list = append(list, entry)
		}
	}

	return list
}

type StatisticsGRPCServer struct {
	UnimplementedStatisticsServer
	db          *goqu.Database
	lastColor   int
	enforcer    *casbin.Enforcer
	aboutConfig config.AboutConfig
}

type scanRow struct {
	UserID int64   `db:"user_id"`
	Date   string  `db:"date"`
	Value  float32 `db:"value"`
}

type picturesStat struct {
	Count int32         `db:"count"`
	Size  sql.NullInt32 `db:"size"`
}

func NewStatisticsGRPCServer(
	db *goqu.Database,
	enforcer *casbin.Enforcer,
	aboutConfig config.AboutConfig,
) *StatisticsGRPCServer {
	return &StatisticsGRPCServer{
		db:          db,
		enforcer:    enforcer,
		aboutConfig: aboutConfig,
	}
}

func (s *StatisticsGRPCServer) randomColor() string {
	idx := s.lastColor % len(colors)
	s.lastColor++

	return colors[idx]
}

func (s *StatisticsGRPCServer) totalUsers(ctx context.Context) (int32, error) {
	var result int32

	success, err := s.db.Select(goqu.COUNT(goqu.Star())).
		From("users").
		Where(goqu.L("NOT deleted")).
		ScanValContext(ctx, &result)

	if err != nil {
		return 0, err
	}

	if !success {
		return 0, failedToFetchRow
	}

	return roundTo(result, thousands), nil
}

func (s *StatisticsGRPCServer) contributors(ctx context.Context) ([]string, error) {
	greenUserRoles := make([]string, 0)

	toFetch := []string{"green-user"}
	for len(toFetch) > 0 {
		ep := len(toFetch) - 1
		role := toFetch[ep]
		toFetch = toFetch[:ep]

		roles, err := s.enforcer.GetUsersForRole(role)
		if err != nil {
			return nil, err
		}

		toFetch = append(toFetch, roles...)

		greenUserRoles = append(greenUserRoles, roles...)
	}

	greenUserRoles = unique(greenUserRoles)

	contributors := make([]string, 0)

	if len(greenUserRoles) > 0 {
		err := s.db.Select("id").From("users").Where(
			goqu.I("deleted").IsFalse(),
			goqu.I("role").In(greenUserRoles),
			goqu.L("(identity is null or identity <> ?)", "autowp"),
			goqu.L("last_online > DATE_SUB(CURDATE(), INTERVAL 6 MONTH)"),
		).ScanValsContext(ctx, &contributors)

		if err != nil {
			return nil, err
		}
	}

	picturesUsers := make([]string, 0)
	err := s.db.Select("id").From("users").
		Where(goqu.I("deleted").IsFalse()).
		Order(goqu.I("pictures_total").Desc()).
		Limit(numberOfTopUploadersToShowInAboutUs).ScanValsContext(ctx, &picturesUsers)

	if err != nil {
		return nil, err
	}

	return unique(append(contributors, picturesUsers...)), nil
}

func (s *StatisticsGRPCServer) picturesStat(ctx context.Context) (int32, int32, error) {
	var picsStat picturesStat

	success, err := s.db.Select(
		goqu.COUNT(goqu.Star()).As("count"),
		goqu.L("SUM(filesize) / 1024 / 1024").As("size"),
	).
		From("pictures").
		ScanStructContext(ctx, &picsStat)

	if err != nil {
		return 0, 0, err
	}

	if !success {
		return 0, 0, failedToFetchRow
	}

	return roundTo(picsStat.Count, tensOfThousands), picsStat.Size.Int32, nil
}

func (s *StatisticsGRPCServer) totalItems(ctx context.Context) (int32, error) {
	var result int32

	success, err := s.db.Select(goqu.COUNT(goqu.Star())).
		From("item").
		ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}
	if !success {
		return 0, failedToFetchRow
	}

	return roundTo(result, thousands), nil
}

func (s *StatisticsGRPCServer) totalComments(ctx context.Context) (int32, error) {
	var result int32

	success, err := s.db.Select(goqu.COUNT(goqu.Star())).
		From("comment_message").
		Where(goqu.I("deleted").IsFalse()).
		ScanValContext(ctx, &result)

	if err != nil {
		return 0, err
	}

	if !success {
		return 0, failedToFetchRow
	}

	return roundTo(result, thousands), nil
}

func (s *StatisticsGRPCServer) GetAboutData(ctx context.Context, _ *emptypb.Empty) (*AboutDataResponse, error) {
	response := AboutDataResponse{
		Developer:      s.aboutConfig.Developer,
		FrTranslator:   s.aboutConfig.FrTranslator,
		ZhTranslator:   s.aboutConfig.ZhTranslator,
		BeTranslator:   s.aboutConfig.BeTranslator,
		PtBrTranslator: s.aboutConfig.PtBrTranslator,
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		var err error
		response.TotalUsers, err = s.totalUsers(ctx)

		if err != nil {
			logrus.Error(err.Error())
		}

		wg.Done()
	}()

	wg.Add(1)

	go func() {
		var err error
		response.Contributors, err = s.contributors(ctx)

		if err != nil {
			logrus.Error(err.Error())
		}

		wg.Done()
	}()

	wg.Add(1)

	go func() {
		var err error
		response.TotalPictures, response.PicturesSize, err = s.picturesStat(ctx)

		if err != nil {
			logrus.Error(err.Error())
		}

		wg.Done()
	}()

	wg.Add(1)

	go func() {
		var err error
		response.TotalItems, err = s.totalItems(ctx)

		if err != nil {
			logrus.Error(err.Error())
		}

		wg.Done()
	}()

	wg.Add(1)

	go func() {
		var err error
		response.TotalComments, err = s.totalComments(ctx)

		if err != nil {
			logrus.Error(err.Error())
		}

		wg.Done()
	}()

	wg.Wait()

	return &response, nil
}

func (s *StatisticsGRPCServer) GetPulse(ctx context.Context, in *PulseRequest) (*PulseResponse, error) {
	now := time.Now()

	var from, to time.Time
	var subPeriodMonth, subPeriodDay int
	var subPeriodHour time.Duration
	var format, dateExpr string

	switch in.GetPeriod() {
	case PulseRequest_YEAR:
		from = time.Now().AddDate(-1, 0, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
		to = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		subPeriodMonth = 1
		format = "2006-01"
		dateExpr = "DATE_FORMAT(add_datetime, '%Y-%m')"

	case PulseRequest_MONTH:
		from = time.Now().AddDate(0, -1, 0)
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		to = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		subPeriodDay = 1
		format = "2006-01-02"
		dateExpr = "DATE_FORMAT(add_datetime, '%Y-%m-%d')"

	default:
		from = time.Now().AddDate(0, 0, -1)
		from = time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), 0, 0, 0, from.Location())
		to = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
		subPeriodHour = time.Hour
		format = "2006-01-02 15"
		dateExpr = "DATE_FORMAT(add_datetime, '%Y-%m-%d %H')"
	}

	var rows []scanRow

	err := s.db.Select(
		goqu.L("user_id").As("user_id"),
		goqu.L(dateExpr).As("date"),
		goqu.L("count(1)").As("value"),
	).From("log_events").
		Where(
			goqu.I("add_datetime").Gte(from),
			goqu.I("add_datetime").Lt(to),
		).
		GroupBy(goqu.I("user_id"), goqu.I("date")).ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	data := make(map[int64]map[string]float32)
	for _, row := range rows {
		_, ok := data[row.UserID]
		if !ok {
			data[row.UserID] = make(map[string]float32)
		}

		data[row.UserID][row.Date] = row.Value
	}

	grid := make([]*PulseGrid, 0)
	legend := make([]*PulseLegend, 0)

	for uid, dates := range data {
		line := make([]float32, 0)

		cDate := from
		for to.After(cDate) {
			dateStr := cDate.Format(format)

			line = append(line, dates[dateStr])

			cDate = cDate.AddDate(0, subPeriodMonth, subPeriodDay).Add(subPeriodHour)
		}

		color := s.randomColor()

		grid = append(grid, &PulseGrid{
			Line:   line,
			Color:  color,
			UserId: uid,
		})

		legend = append(legend, &PulseLegend{
			UserId: uid,
			Color:  color,
		})
	}

	labels := make([]string, 0)
	cDate := from

	for to.After(cDate) {
		labels = append(labels, cDate.Format(format))

		cDate = cDate.AddDate(0, subPeriodMonth, subPeriodDay).Add(subPeriodHour)
	}

	return &PulseResponse{
		Grid:   grid,
		Legend: legend,
		Labels: labels,
	}, nil
}
