package goautowp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	errAuthTokenIsInvalid  = errors.New("authorization token is invalid")
	errFailedRoleDetection = errors.New("failed role detection")
)

const (
	authorizationHeader = "authorization"
	bearerSchema        = "Bearer"
)

type Auth struct {
	db          *goqu.Database
	keycloak    *gocloak.GoCloak
	keycloakCfg config.KeycloakConfig
	repository  *users.Repository
}

func NewAuth(
	db *goqu.Database,
	keycloak *gocloak.GoCloak,
	keycloakCfg config.KeycloakConfig,
	repository *users.Repository,
) *Auth {
	return &Auth{
		db:          db,
		keycloak:    keycloak,
		keycloakCfg: keycloakCfg,
		repository:  repository,
	}
}

func (s *Auth) ValidateREST(ctx *gin.Context) (int64, string, error) {
	header := ctx.GetHeader(authorizationHeader)

	if len(header) == 0 {
		return 0, "", nil
	}

	tokenString := strings.TrimPrefix(header, bearerSchema+" ")

	return s.ValidateToken(ctx, tokenString)
}

func (s *Auth) ValidateGRPC(ctx context.Context) (int64, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	lines := md[authorizationHeader]

	if len(lines) < 1 {
		return 0, "", nil
	}

	tokenString := strings.TrimPrefix(lines[0], bearerSchema+" ")

	return s.ValidateToken(ctx, tokenString)
}

func (s *Auth) ValidateToken(ctx context.Context, tokenString string) (int64, string, error) {
	if len(tokenString) == 0 {
		return 0, "", errAuthTokenIsInvalid
	}

	var claims users.Claims

	_, err := s.keycloak.DecodeAccessTokenCustomClaims(ctx, tokenString, s.keycloakCfg.Realm, &claims)
	if err != nil {
		return 0, "", err
	}

	id, role, err := s.repository.EnsureUserImported(ctx, claims)
	if err != nil {
		return 0, "", err
	}

	if role == "" {
		return 0, "", fmt.Errorf("%w: subject: `%v`", errFailedRoleDetection, claims.Subject)
	}

	err = s.repository.RegisterVisit(ctx, id)
	if err != nil {
		return 0, "", err
	}

	return id, role, nil
}
