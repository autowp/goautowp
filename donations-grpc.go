package goautowp

import (
	"context"
	"time"

	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DonationsGRPCServer struct {
	UnimplementedDonationsServer
	itemOfDay         *itemofday.Repository
	donationsVodPrice int32
	db                *goqu.Database
}

func NewDonationsGRPCServer(
	itemOfDay *itemofday.Repository, donationsVodPrice int32, db *goqu.Database,
) *DonationsGRPCServer {
	return &DonationsGRPCServer{
		itemOfDay:         itemOfDay,
		donationsVodPrice: donationsVodPrice,
		db:                db,
	}
}

func (s *DonationsGRPCServer) GetVODData(ctx context.Context, _ *emptypb.Empty) (*VODDataResponse, error) {
	dates := make([]*VODDataDate, 0)

	nextDates, err := s.itemOfDay.NextDates(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, nextDate := range nextDates {
		dates = append(dates, &VODDataDate{
			Date: timestamppb.New(nextDate.Date),
			Free: nextDate.Free,
		})
	}

	return &VODDataResponse{
		Dates: dates,
		Sum:   s.donationsVodPrice,
	}, nil
}

func (s *DonationsGRPCServer) GetTransactions(
	ctx context.Context, _ *emptypb.Empty,
) (*DonationsTransactionsResponse, error) {
	var rows []struct {
		Sum         int32     `db:"sum"`
		Currency    string    `db:"currency"`
		Date        time.Time `db:"date"`
		Contributor string    `db:"contributor"`
		Purpose     string    `db:"purpose"`
	}

	err := s.db.Select(schema.TransactionTableSumCol, schema.TransactionTableCurrencyCol, schema.TransactionTableDateCol,
		schema.TransactionTableContributorCol, schema.TransactionTablePurposeCol).
		From(schema.TransactionTable).
		Order(schema.TransactionTableDateCol.Desc()).
		Where(schema.TransactionTableDateCol.Gt(goqu.L("CURRENT_DATE - INTERVAL '6 months'"))).
		ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	res := make([]*DonationsTransaction, 0, len(rows))
	for _, row := range rows {
		res = append(res, &DonationsTransaction{
			Sum:         row.Sum,
			Currency:    row.Currency,
			Date:        timestamppb.New(row.Date),
			Contributor: row.Contributor,
			Purpose:     row.Purpose,
		})
	}

	return &DonationsTransactionsResponse{Items: res}, nil
}
