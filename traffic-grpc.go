package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/autowp/goautowp/traffic"

	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const trafficTopLimit = 50

var ErrUserNotFound = errors.New("user not found")

type TrafficGRPCServer struct {
	UnimplementedTrafficServer
	auth          *Auth
	db            *goqu.Database
	enforcer      *casbin.Enforcer
	userExtractor *UserExtractor
	traffic       *traffic.Traffic
}

func NewTrafficGRPCServer(
	auth *Auth,
	db *goqu.Database,
	enforcer *casbin.Enforcer,
	userExtractor *UserExtractor,
	traffic *traffic.Traffic,
) *TrafficGRPCServer {
	return &TrafficGRPCServer{
		auth:          auth,
		db:            db,
		enforcer:      enforcer,
		userExtractor: userExtractor,
		traffic:       traffic,
	}
}

func (s *TrafficGRPCServer) GetTop(ctx context.Context, _ *emptypb.Empty) (*APITrafficTopResponse, error) {
	var err error

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
			user       *users.DBUser
			topItemBan *APIBanItem
		)

		if banItem != nil {
			user, err = s.getUser(ctx, banItem.ByUserID)
			if err != nil && !errors.Is(err, ErrUserNotFound) {
				return nil, status.Error(codes.Internal, err.Error())
			}

			var extractedUser *APIUser

			if user != nil {
				extractedUser, err = s.userExtractor.Extract(ctx, user)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
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
			Count:       int32(item.Count),
			Ban:         topItemBan,
			InWhitelist: inWhitelist,
			WhoisUrl:    fmt.Sprintf("https://nic.ru/whois/?query=%s", url.QueryEscape(item.IP.String())),
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
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "user", "ban"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
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
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
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
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "user", "ban"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

	duration := time.Hour * time.Duration(in.Period)

	err = s.traffic.Ban.Add(ctx, ip, duration, userID, in.Reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) AddToWhitelist(
	ctx context.Context,
	in *AddToTrafficWhitelistRequest,
) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	ip := net.ParseIP(in.Ip)
	if ip == nil {
		return nil, status.Errorf(codes.InvalidArgument, "InvalidArgument")
	}

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
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
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

func (s *TrafficGRPCServer) getUser(ctx context.Context, id int64) (*users.DBUser, error) {
	var r users.DBUser

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, deleted, identity, last_online, role, specs_weight
		FROM users
		WHERE id = ?
	`, id).Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return &r, nil
}
