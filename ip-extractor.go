package goautowp

import (
	"context"
	"errors"
	"net"

	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IPExtractor struct {
	enforcer       *casbin.Enforcer
	banRepository  *ban.Repository
	userRepository *users.Repository
	userExtractor  *UserExtractor
}

func NewIPExtractor(
	enforcer *casbin.Enforcer,
	banRepository *ban.Repository,
	userRepository *users.Repository,
	userExtractor *UserExtractor,
) *IPExtractor {
	return &IPExtractor{
		enforcer:       enforcer,
		banRepository:  banRepository,
		userRepository: userRepository,
		userExtractor:  userExtractor,
	}
}

func (s *IPExtractor) Extract(
	ctx context.Context, ip net.IP, fields map[string]bool, userID int64, role string,
) (*APIIP, error) {
	result := APIIP{
		Address: ip.String(),
	}

	_, ok := fields["hostname"]
	if ok {
		host, err := net.LookupAddr(ip.String())
		if err != nil {
			logrus.Errorf("LookupAddr error: %v", err.Error())
		}

		if len(host) > 0 {
			result.Hostname = host[0]
		}
	}

	_, ok = fields["blacklist"]

	if ok {
		canView := len(role) > 0 && s.enforcer.Enforce(role, "global", "moderate")

		if canView {
			result.Blacklist = nil

			banItem, err := s.banRepository.Get(ctx, ip)
			if err != nil && !errors.Is(err, ban.ErrBanItemNotFound) {
				return nil, err
			}

			if banItem != nil {
				result.Blacklist = &APIBanItem{
					Until:    timestamppb.New(banItem.Until),
					ByUserId: banItem.ByUserID,
					ByUser:   nil,
					Reason:   banItem.Reason,
				}

				user, err := s.userRepository.User(ctx, users.GetUsersOptions{ID: banItem.ByUserID})
				if err != nil && !errors.Is(err, users.ErrUserNotFound) {
					return nil, err
				}

				if user != nil {
					apiUser, err := s.userExtractor.Extract(ctx, user, nil, userID, role)
					if err != nil {
						return nil, err
					}

					result.Blacklist.ByUser = apiUser
				}
			}
		}
	}

	_, ok = fields["rights"]
	if ok {
		canBan := len(role) > 0 && s.enforcer.Enforce(role, "user", "ban")

		result.Rights = &APIIPRights{
			AddToBlacklist:      canBan,
			RemoveFromBlacklist: canBan,
		}
	}

	return &result, nil
}
