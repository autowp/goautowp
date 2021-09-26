package auth

import (
	"database/sql"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/autowp/goautowp/auth/oauth2server"
	"github.com/autowp/goautowp/auth/oauth2server/models"
	jsoniter "github.com/json-iterator/go"
)

// TokenStore PostgreSQL token store
type TokenStore struct {
	adapter *sql.DB

	gcDisabled bool
	gcInterval time.Duration
	ticker     *time.Ticker
}

// TokenStoreItem data item
type TokenStoreItem struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
	Code      string    `db:"code"`
	Access    string    `db:"access"`
	Refresh   string    `db:"refresh"`
	Data      []byte    `db:"data"`
}

// NewTokenStore creates PostgreSQL store instance
func NewTokenStore(adapter *sql.DB, options ...TokenStoreOption) (*TokenStore, error) {
	store := &TokenStore{
		adapter:    adapter,
		gcInterval: 10 * time.Minute,
	}

	for _, o := range options {
		o(store)
	}

	var err error

	if !store.gcDisabled {
		store.ticker = time.NewTicker(store.gcInterval)
		go store.gc()
	}

	return store, err
}

// Close close the store
func (s *TokenStore) Close() error {
	if !s.gcDisabled {
		s.ticker.Stop()
	}
	return nil
}

func (s *TokenStore) gc() {
	for range s.ticker.C {
		s.clean()
	}
}

func (s *TokenStore) clean() {
	now := time.Now()
	_, err := s.adapter.Exec("DELETE FROM tokens WHERE expires_at <= $1", now)
	if err != nil {
		logrus.Errorf("Error while cleaning out outdated entities: %+v", err)
	}
}

// Create creates and stores the new token information
func (s *TokenStore) Create(info oauth2server.TokenInfo) error {
	buf, err := jsoniter.Marshal(info)
	if err != nil {
		return err
	}

	item := &TokenStoreItem{
		Data:      buf,
		CreatedAt: time.Now(),
	}

	if code := info.GetCode(); code != "" {
		item.Code = code
		item.ExpiresAt = info.GetCodeCreateAt().Add(info.GetCodeExpiresIn())
	} else {
		item.Access = info.GetAccess()
		item.ExpiresAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn())

		if refresh := info.GetRefresh(); refresh != "" {
			item.Refresh = info.GetRefresh()
			item.ExpiresAt = info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		}
	}

	_, err = s.adapter.Exec(
		"INSERT INTO tokens (created_at, expires_at, code, access, refresh, data) VALUES ($1, $2, $3, $4, $5, $6)",
		item.CreatedAt,
		item.ExpiresAt,
		item.Code,
		item.Access,
		item.Refresh,
		item.Data,
	)

	return err
}

// RemoveByCode deletes the authorization code
func (s *TokenStore) RemoveByCode(code string) error {
	_, err := s.adapter.Exec("DELETE FROM tokens WHERE code = $1", code)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByAccess uses the access token to delete the token information
func (s *TokenStore) RemoveByAccess(access string) error {
	_, err := s.adapter.Exec("DELETE FROM tokens WHERE access = $1", access)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByRefresh uses the refresh token to delete the token information
func (s *TokenStore) RemoveByRefresh(refresh string) error {
	_, err := s.adapter.Exec("DELETE FROM tokens WHERE refresh = $1", refresh)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (s *TokenStore) toTokenInfo(data []byte) (oauth2server.TokenInfo, error) {
	var tm models.Token
	err := jsoniter.Unmarshal(data, &tm)
	return &tm, err
}

// GetByCode uses the authorization code for token information data
func (s *TokenStore) GetByCode(code string) (oauth2server.TokenInfo, error) {
	if code == "" {
		return nil, nil
	}

	row := s.adapter.QueryRow("SELECT id, created_at, expires_at, code, access, refresh, data FROM tokens WHERE code = $1", code)

	var item TokenStoreItem
	err := row.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt, &item.Code, &item.Access, &item.Refresh, &item.Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.toTokenInfo(item.Data)
}

// GetByAccess uses the access token for token information data
func (s *TokenStore) GetByAccess(access string) (oauth2server.TokenInfo, error) {
	if access == "" {
		return nil, nil
	}

	row := s.adapter.QueryRow("SELECT id, created_at, expires_at, code, access, refresh, data FROM tokens WHERE access = $1", access)

	var item TokenStoreItem
	err := row.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt, &item.Code, &item.Access, &item.Refresh, &item.Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.toTokenInfo(item.Data)
}

// GetByRefresh uses the refresh token for token information data
func (s *TokenStore) GetByRefresh(refresh string) (oauth2server.TokenInfo, error) {
	if refresh == "" {
		return nil, nil
	}

	row := s.adapter.QueryRow("SELECT id, created_at, expires_at, code, access, refresh, data FROM tokens WHERE refresh = $1", refresh)

	var item TokenStoreItem
	err := row.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt, &item.Code, &item.Access, &item.Refresh, &item.Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.toTokenInfo(item.Data)
}
