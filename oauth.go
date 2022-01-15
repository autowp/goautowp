package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"github.com/sirupsen/logrus"
)

type OAuth struct {
	keycloakConfig  config.KeycloakConfig
	keyCloak        gocloak.GoCloak
	usersRepository *users.Repository
}

func NewOAuth(keycloakConfig config.KeycloakConfig, keyCloak gocloak.GoCloak, usersRepository *users.Repository) *OAuth {
	return &OAuth{
		keycloakConfig:  keycloakConfig,
		keyCloak:        keyCloak,
		usersRepository: usersRepository,
	}
}

func (s *OAuth) TokenByRefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error) {
	jwtToken, err := s.keyCloak.RefreshToken(
		ctx,
		refreshToken,
		s.keycloakConfig.ClientID,
		s.keycloakConfig.ClientSecret,
		s.keycloakConfig.Realm,
	)
	if err != nil {
		return nil, err
	}

	return jwtToken, nil
}

func (s *OAuth) TokenByPassword(ctx context.Context, username string, password string) (*gocloak.JWT, int64, error) {
	userId, err := s.usersRepository.UserByCredentials(username, password)
	if err != nil {
		return nil, 0, err
	}

	if userId == 0 {
		return nil, 0, nil
	}

	logrus.Debugf("Login `%s` to Keycloak by credentials", username)
	jwtToken, err := s.keyCloak.Login(
		ctx,
		s.keycloakConfig.ClientID,
		s.keycloakConfig.ClientSecret,
		s.keycloakConfig.Realm,
		username,
		password,
	)
	if err != nil {
		return nil, userId, err
	}

	return jwtToken, userId, nil
}
