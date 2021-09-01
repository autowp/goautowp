package goautowp

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
)

type IPExtractor struct {
	container *Container
}

func NewIPExtractor(container *Container) *IPExtractor {
	return &IPExtractor{
		container: container,
	}
}

func (s *IPExtractor) Extract(ip net.IP, fields map[string]bool, role string) (*APIIP, error) {
	result := APIIP{
		Address: ip.String(),
	}

	_, ok := fields["hostname"]
	if ok {
		host, err := net.LookupAddr(ip.String())
		if err != nil {
			log.Printf("LookupAddr error: %v", err.Error())
		}

		if len(host) > 0 {
			result.Hostname = host[0]
		}
	}

	_, ok = fields["blacklist"]
	if ok {

		enforcer := s.container.GetEnforcer()

		canView := len(role) > 0 && enforcer.Enforce(role, "global", "moderate")

		if canView {
			result.Blacklist = nil

			banRepository, err := s.container.GetBanRepository()
			if err != nil {
				return nil, err
			}

			ban, err := banRepository.Get(ip)
			if err != nil {
				return nil, err
			}

			if ban != nil {
				result.Blacklist = &APIBanItem{
					Until:    timestamppb.New(ban.Until),
					ByUserId: int32(ban.ByUserID),
					ByUser:   nil,
					Reason:   ban.Reason,
				}

				userRepository, err := s.container.GetUserRepository()
				if err != nil {
					return nil, err
				}

				user, err := userRepository.GetUser(GetUsersOptions{ID: ban.ByUserID})
				if err != nil {
					return nil, err
				}

				if user != nil {
					userExtractor := s.container.GetUserExtractor()

					result.Blacklist.ByUser, err = userExtractor.Extract(user, map[string]bool{})
					if err != nil {
						return nil, err
					}
				}

			}
		}
	}

	_, ok = fields["rights"]
	if ok {
		enforcer := s.container.GetEnforcer()

		canBan := len(role) > 0 && enforcer.Enforce(role, "user", "ban")

		result.Rights = &APIIPRights{
			AddToBlacklist:      canBan,
			RemoveFromBlacklist: canBan,
		}
	}

	return &result, nil
}
