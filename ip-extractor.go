package goautowp

import (
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
			result.Hostname = &host[0]
		}
	}

	_, ok = fields["blacklist"]
	if ok {

		enforcer, err := s.container.GetEnforcer()
		if err != nil {
			return nil, err
		}

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
				result.Blacklist = &APIIPBlacklist{
					Until:    ban.Until,
					ByUserID: ban.ByUserID,
					User:     nil,
					Reason:   ban.Reason,
				}

				userRepository, err := s.container.GetUserRepository()
				if err != nil {
					return nil, err
				}

				user, err := userRepository.GetUser(ban.ByUserID)
				if err != nil {
					return nil, err
				}

				if user != nil {
					userExtractor, err := s.container.GetUserExtractor()
					if err != nil {
						return nil, err
					}

					result.Blacklist.User, err = userExtractor.Extract(user)
					if err != nil {
						return nil, err
					}
				}

			}
		}
	}

	_, ok = fields["rights"]
	if ok {
		enforcer, err := s.container.GetEnforcer()
		if err != nil {
			return nil, err
		}

		canBan := len(role) > 0 && enforcer.Enforce(role, "user", "ban")

		result.Rights = &APIIPRights{
			AddToBlacklist:      canBan,
			RemoveFromBlacklist: canBan,
		}
	}

	return &result, nil
}
