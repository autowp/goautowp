package goautowp

import (
	"context"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type StatisticsGRPCServer struct {
	UnimplementedStatisticsServer
	db        *goqu.Database
	lastColor int
}

type scanRow struct {
	UserId int64   `db:"user_id"`
	Date   string  `db:"date"`
	Value  float32 `db:"value"`
}

func NewStatisticsGRPCServer(
	db *goqu.Database,
) *StatisticsGRPCServer {
	return &StatisticsGRPCServer{
		db: db,
	}
}

func (s *StatisticsGRPCServer) randomColor() string {
	idx := s.lastColor % len(colors)
	s.lastColor++
	return colors[idx]
}

func (s *StatisticsGRPCServer) GetPulse(ctx context.Context, in *PulseRequest) (*PulseResponse, error) {
	now := time.Now()
	var from, to time.Time
	subPeriodMonth := 0
	subPeriodDay := 0
	var subPeriodHour time.Duration = 0
	var format string
	var dateExpr string

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
		GroupBy(goqu.I("user_id"), goqu.I("date")).Executor().ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	data := make(map[int64]map[string]float32)
	for _, row := range rows {
		_, ok := data[row.UserId]
		if !ok {
			data[row.UserId] = make(map[string]float32)
		}
		data[row.UserId][row.Date] = row.Value
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
