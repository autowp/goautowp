package goautowp

import (
	"context"

	"github.com/autowp/goautowp/itemofday"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DonationsGRPCServer struct {
	UnimplementedDonationsServer
	itemOfDay         *itemofday.Repository
	donationsVodPrice int32
}

func NewDonationsGRPCServer(itemOfDay *itemofday.Repository, donationsVodPrice int32) *DonationsGRPCServer {
	return &DonationsGRPCServer{
		itemOfDay:         itemOfDay,
		donationsVodPrice: donationsVodPrice,
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
