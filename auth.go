package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

type Auth struct {
	db          *sql.DB
	keycloak    gocloak.GoCloak
	keycloakCfg config.KeycloakConfig
}

func NewAuth(db *sql.DB, keycloak gocloak.GoCloak, keycloakCfg config.KeycloakConfig) *Auth {
	return &Auth{
		db:          db,
		keycloak:    keycloak,
		keycloakCfg: keycloakCfg,
	}
}

func (s *Auth) ValidateGRPC(ctx context.Context) (int64, string, error) {
	const bearerSchema = "Bearer"

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	lines := md["authorization"]

	if len(lines) < 1 {
		return 0, "", nil
	}

	tokenString := strings.TrimPrefix(lines[0], bearerSchema+" ")

	return s.ValidateToken(ctx, tokenString)
}

func (s *Auth) ValidateToken(ctx context.Context, tokenString string) (int64, string, error) {
	if len(tokenString) <= 0 {
		return 0, "", fmt.Errorf("authorization token is invalid")
	}

	_, claims, err := s.keycloak.DecodeAccessToken(ctx, tokenString, s.keycloakCfg.Realm, "")
	if err != nil {
		return 0, "", err
	}

	guid := (*claims)["sub"].(string)

	var id int64
	role := ""
	err = s.db.QueryRow(`
		SELECT users.id, users.role
		FROM users
			JOIN user_account ON users.id = user_account.user_id
		WHERE user_account.external_id = ? AND user_account.service_id = ? AND not users.deleted
	`, guid, users.KeycloakExternalAccountID).Scan(&id, &role)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("user `%v` not found", guid)
	}

	if err != nil {
		return 0, "", err
	}

	if role == "" {
		return 0, "", fmt.Errorf("failed role detection for `%v`", guid)
	}

	return id, role, nil
}
