package goautowp

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/traffic"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const trafficTopLimit = 50

type TrafficGRPCServer struct {
	UnimplementedTrafficServer
	auth            *Auth
	usersRepository *users.Repository
	userExtractor   *UserExtractor
	traffic         *traffic.Traffic
}

func NewTrafficGRPCServer(
	auth *Auth,
	usersRepository *users.Repository,
	userExtractor *UserExtractor,
	traffic *traffic.Traffic,
) *TrafficGRPCServer {
	return &TrafficGRPCServer{
		auth:            auth,
		usersRepository: usersRepository,
		userExtractor:   userExtractor,
		traffic:         traffic,
	}
}

func (s *TrafficGRPCServer) GetTop(ctx context.Context, _ *emptypb.Empty) (*APITrafficTopResponse, error) {
	var err error

	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items, err := s.traffic.Monitoring.ListOfTop(ctx, trafficTopLimit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APITrafficTopItem, len(items))

	for idx, item := range items {
		banItem, banErr := s.traffic.Ban.Get(ctx, item.IP)
		if banErr != nil && !errors.Is(banErr, ban.ErrBanItemNotFound) {
			return nil, status.Error(codes.Internal, banErr.Error())
		}

		var (
			user       *schema.UsersRow
			topItemBan *APIBanItem
		)

		if banItem != nil {
			var extractedUser *APIUser

			if banItem.ByUserID > 0 {
				user, err = s.usersRepository.User(ctx, &query.UserListOptions{
					ID: banItem.ByUserID,
				}, users.UserFields{}, users.OrderByNone)
				if err != nil && !errors.Is(err, users.ErrUserNotFound) {
					return nil, status.Error(codes.Internal, err.Error())
				}

				if user != nil {
					extractedUser, err = s.userExtractor.Extract(ctx, user, nil, userCtx.UserID, userCtx.Roles)
					if err != nil {
						return nil, status.Error(codes.Internal, err.Error())
					}
				}
			}

			topItemBan = &APIBanItem{
				Until:    timestamppb.New(banItem.Until),
				ByUserId: banItem.ByUserID,
				ByUser:   extractedUser,
				Reason:   banItem.Reason,
			}
		}

		inWhitelist, err := s.traffic.Whitelist.Exists(ctx, item.IP)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		result[idx] = &APITrafficTopItem{
			Ip:          item.IP.String(),
			Count:       int32(item.Count), //nolint: gosec
			Ban:         topItemBan,
			InWhitelist: inWhitelist,
			WhoisUrl:    "https://nic.ru/whois/?query=" + url.QueryEscape(item.IP.String()),
		}
	}

	return &APITrafficTopResponse{
		Items: result,
	}, nil
}

func (s *TrafficGRPCServer) DeleteFromBlacklist(
	ctx context.Context,
	in *DeleteFromTrafficBlacklistRequest,
) (*emptypb.Empty, error) {
	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(userCtx.Roles, users.RoleUsersModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.GetIp())
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.traffic.Ban.Remove(ctx, ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) DeleteFromWhitelist(
	ctx context.Context,
	in *DeleteFromTrafficWhitelistRequest,
) (*emptypb.Empty, error) {
	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(userCtx.Roles, users.RoleModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.GetIp())
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	err = s.traffic.Whitelist.Remove(ctx, ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) AddToBlacklist(
	ctx context.Context,
	in *AddToTrafficBlacklistRequest,
) (*emptypb.Empty, error) {
	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(userCtx.Roles, users.RoleUsersModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.GetIp())
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	duration := time.Hour * time.Duration(in.GetPeriod())

	err = s.traffic.Ban.Add(ctx, ip, duration, userCtx.UserID, in.GetReason())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) AddToWhitelist(
	ctx context.Context,
	in *AddToTrafficWhitelistRequest,
) (*emptypb.Empty, error) {
	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(userCtx.Roles, users.RoleModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.GetIp())
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	ctx = context.WithoutCancel(ctx)

	err = s.traffic.Whitelist.Add(ctx, ip, "manual click")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.traffic.Ban.Remove(ctx, ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) GetTrafficWhitelist(
	ctx context.Context,
	_ *emptypb.Empty,
) (*APITrafficWhitelistItems, error) {
	userCtx, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !util.Contains(userCtx.Roles, users.RoleModer) {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	list, err := s.traffic.Whitelist.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APITrafficWhitelistItem, len(list))
	for idx, i := range list {
		result[idx] = &APITrafficWhitelistItem{
			Ip:          i.IP.String(),
			Description: i.Description,
		}
	}

	return &APITrafficWhitelistItems{
		Items: result,
	}, nil
}
