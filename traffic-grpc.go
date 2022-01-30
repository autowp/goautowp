package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"net/url"
	"time"
)

type TrafficGRPCServer struct {
	UnimplementedTrafficServer
	auth          *Auth
	db            *sql.DB
	enforcer      *casbin.Enforcer
	userExtractor *UserExtractor
	traffic       *Traffic
}

func NewTrafficGRPCServer(
	auth *Auth,
	db *sql.DB,
	enforcer *casbin.Enforcer,
	userExtractor *UserExtractor,
	traffic *Traffic,
) *TrafficGRPCServer {
	return &TrafficGRPCServer{
		auth:          auth,
		db:            db,
		enforcer:      enforcer,
		userExtractor: userExtractor,
		traffic:       traffic,
	}
}

func (s *TrafficGRPCServer) GetTrafficTop(_ context.Context, _ *emptypb.Empty) (*APITrafficTopResponse, error) {

	items, err := s.traffic.Monitoring.ListOfTop(50)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APITrafficTopItem, len(items))
	for idx, item := range items {

		ban, err := s.traffic.Ban.Get(item.IP)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		inWhitelist, err := s.traffic.Whitelist.Exists(item.IP)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		var user *users.DBUser
		var topItemBan *APIBanItem

		if ban != nil {
			user, err = s.getUser(ban.ByUserID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			extractedUser, err := s.userExtractor.Extract(user, map[string]bool{})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			topItemBan = &APIBanItem{
				Until:    timestamppb.New(ban.Until),
				ByUserId: ban.ByUserID,
				ByUser:   extractedUser,
				Reason:   ban.Reason,
			}
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

func (s *TrafficGRPCServer) DeleteFromTrafficBlacklist(ctx context.Context, in *DeleteFromTrafficBlacklistRequest) (*emptypb.Empty, error) {

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

	err = s.traffic.Ban.Remove(ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) DeleteFromTrafficWhitelist(ctx context.Context, in *DeleteFromTrafficWhitelistRequest) (*emptypb.Empty, error) {
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

	err = s.traffic.Whitelist.Remove(ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) AddToTrafficBlacklist(ctx context.Context, in *AddToTrafficBlacklistRequest) (*emptypb.Empty, error) {
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

	err = s.traffic.Ban.Add(ip, duration, userID, in.Reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) AddToTrafficWhitelist(ctx context.Context, in *AddToTrafficWhitelistRequest) (*emptypb.Empty, error) {
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

	err = s.traffic.Whitelist.Add(ip, "manual click")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.traffic.Ban.Remove(ip)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TrafficGRPCServer) GetTrafficWhitelist(ctx context.Context, _ *emptypb.Empty) (*APITrafficWhitelistItems, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
		return nil, status.Errorf(codes.PermissionDenied, "PermissionDenied")
	}

	list, err := s.traffic.Whitelist.List()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APITrafficWhitelistItems{
		Items: list,
	}, nil
}

func (s *TrafficGRPCServer) getUser(id int64) (*users.DBUser, error) {
	rows, err := s.db.Query(`
		SELECT id, name, deleted, identity, last_online, role, specs_weight
		FROM users
		WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	if !rows.Next() {
		return nil, nil
	}

	var r users.DBUser
	err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role, &r.SpecsWeight)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
