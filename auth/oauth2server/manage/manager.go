package manage

import (
	"time"

	"github.com/autowp/goautowp/auth/oauth2server"
	"github.com/autowp/goautowp/auth/oauth2server/errors"
	"github.com/autowp/goautowp/auth/oauth2server/models"
)

// NewManager create to authorization management instance
func NewManager() *Manager {
	return &Manager{
		gtcfg: make(map[oauth2server.GrantType]*Config),
	}
}

// Manager provide authorization management
type Manager struct {
	gtcfg          map[oauth2server.GrantType]*Config
	rcfg           *RefreshingConfig
	accessGenerate oauth2server.AccessGenerate
	tokenStore     oauth2server.TokenStore
	clientStore    oauth2server.ClientStore
}

// get grant type config
func (m *Manager) grantConfig(gt oauth2server.GrantType) *Config {
	if c, ok := m.gtcfg[gt]; ok && c != nil {
		return c
	}
	switch gt {
	case oauth2server.SocialAuthorizationCode:
		return DefaultSocialTokenCfg
	case oauth2server.PasswordCredentials:
		return DefaultPasswordTokenCfg
	}
	return &Config{}
}

// SetPasswordTokenCfg set the password grant token config
func (m *Manager) SetPasswordTokenCfg(cfg *Config) {
	m.gtcfg[oauth2server.PasswordCredentials] = cfg
}

// SetSocialTokenCfg set the social grant token config
func (m *Manager) SetSocialTokenCfg(cfg *Config) {
	m.gtcfg[oauth2server.SocialAuthorizationCode] = cfg
}

// SetRefreshTokenCfg set the refreshing token config
func (m *Manager) SetRefreshTokenCfg(cfg *RefreshingConfig) {
	m.rcfg = cfg
}

// MapAccessGenerate mapping the access token generate interface
func (m *Manager) MapAccessGenerate(gen oauth2server.AccessGenerate) {
	m.accessGenerate = gen
}

// MapClientStorage mapping the client store interface
func (m *Manager) MapClientStorage(stor oauth2server.ClientStore) {
	m.clientStore = stor
}

// MustClientStorage mandatory mapping the client store interface
func (m *Manager) MustClientStorage(stor oauth2server.ClientStore, err error) {
	if err != nil {
		panic(err.Error())
	}
	m.clientStore = stor
}

// MapTokenStorage mapping the token store interface
func (m *Manager) MapTokenStorage(stor oauth2server.TokenStore) {
	m.tokenStore = stor
}

// MustTokenStorage mandatory mapping the token store interface
func (m *Manager) MustTokenStorage(stor oauth2server.TokenStore, err error) {
	if err != nil {
		panic(err)
	}
	m.tokenStore = stor
}

// GetClient get the client information
func (m *Manager) GetClient(clientID string) (cli oauth2server.ClientInfo, err error) {
	cli, err = m.clientStore.GetByID(clientID)
	if err != nil {
		return
	} else if cli == nil {
		err = errors.ErrInvalidClient
	}
	return
}

// GenerateAccessToken generate the access token
func (m *Manager) GenerateAccessToken(gt oauth2server.GrantType, tgr *oauth2server.TokenGenerateRequest) (oauth2server.TokenInfo, error) {
	cli, err := m.GetClient(tgr.ClientID)
	if err != nil {
		return nil, err
	} else if tgr.ClientSecret != cli.GetSecret() {
		return nil, errors.ErrInvalidClient
	}

	ti := models.NewToken()
	ti.SetClientID(tgr.ClientID)
	ti.SetUserID(tgr.UserID)
	ti.SetScope(tgr.Scope)

	createAt := time.Now()
	ti.SetAccessCreateAt(createAt)

	// set access token expires
	gcfg := m.grantConfig(gt)
	aexp := gcfg.AccessTokenExp
	if exp := tgr.AccessTokenExp; exp > 0 {
		aexp = exp
	}
	ti.SetAccessExpiresIn(aexp)
	if gcfg.IsGenerateRefresh {
		ti.SetRefreshCreateAt(createAt)
		ti.SetRefreshExpiresIn(gcfg.RefreshTokenExp)
	}

	td := &oauth2server.GenerateBasic{
		Client:    cli,
		UserID:    tgr.UserID,
		CreateAt:  createAt,
		TokenInfo: ti,
		Request:   tgr.Request,
	}

	av, rv, err := m.accessGenerate.Token(td, gcfg.IsGenerateRefresh)
	if err != nil {
		return nil, err
	}
	ti.SetAccess(av)

	if rv != "" {
		ti.SetRefresh(rv)
	}

	err = m.tokenStore.Create(ti)
	if err != nil {
		return nil, err
	}

	return ti, nil
}

// RefreshAccessToken refreshing an access token
func (m *Manager) RefreshAccessToken(tgr *oauth2server.TokenGenerateRequest) (oauth2server.TokenInfo, error) {
	cli, err := m.GetClient(tgr.ClientID)
	if err != nil {
		return nil, err
	} else if tgr.ClientSecret != cli.GetSecret() {
		return nil, errors.ErrInvalidClient
	}

	ti, err := m.LoadRefreshToken(tgr.Refresh)
	if err != nil {
		return nil, err
	} else if ti.GetClientID() != tgr.ClientID {
		return nil, errors.ErrInvalidRefreshToken
	}

	oldAccess, oldRefresh := ti.GetAccess(), ti.GetRefresh()

	td := &oauth2server.GenerateBasic{
		Client:    cli,
		UserID:    ti.GetUserID(),
		CreateAt:  time.Now(),
		TokenInfo: ti,
		Request:   tgr.Request,
	}

	rcfg := DefaultRefreshTokenCfg
	if v := m.rcfg; v != nil {
		rcfg = v
	}

	ti.SetAccessCreateAt(td.CreateAt)
	if v := rcfg.AccessTokenExp; v > 0 {
		ti.SetAccessExpiresIn(v)
	}

	if v := rcfg.RefreshTokenExp; v > 0 {
		ti.SetRefreshExpiresIn(v)
	}

	if rcfg.IsResetRefreshTime {
		ti.SetRefreshCreateAt(td.CreateAt)
	}

	if scope := tgr.Scope; scope != "" {
		ti.SetScope(scope)
	}

	tv, rv, err := m.accessGenerate.Token(td, rcfg.IsGenerateRefresh)
	if err != nil {
		return nil, err
	}

	ti.SetAccess(tv)
	if rv != "" {
		ti.SetRefresh(rv)
	}

	if err := m.tokenStore.Create(ti); err != nil {
		return nil, err
	}

	if rcfg.IsRemoveAccess {
		// remove the old access token
		if err := m.tokenStore.RemoveByAccess(oldAccess); err != nil {
			return nil, err
		}
	}

	if rcfg.IsRemoveRefreshing && rv != "" {
		// remove the old refresh token
		if err := m.tokenStore.RemoveByRefresh(oldRefresh); err != nil {
			return nil, err
		}
	}

	if rv == "" {
		ti.SetRefresh("")
		ti.SetRefreshCreateAt(time.Now())
		ti.SetRefreshExpiresIn(0)
	}

	return ti, nil
}

// RemoveAccessToken use the access token to delete the token information
func (m *Manager) RemoveAccessToken(access string) error {
	if access == "" {
		return errors.ErrInvalidAccessToken
	}
	return m.tokenStore.RemoveByAccess(access)
}

// RemoveRefreshToken use the refresh token to delete the token information
func (m *Manager) RemoveRefreshToken(refresh string) error {
	if refresh == "" {
		return errors.ErrInvalidAccessToken
	}
	return m.tokenStore.RemoveByRefresh(refresh)
}

// LoadAccessToken according to the access token for corresponding token information
func (m *Manager) LoadAccessToken(access string) (oauth2server.TokenInfo, error) {
	if access == "" {
		return nil, errors.ErrInvalidAccessToken
	}

	ct := time.Now()
	ti, err := m.tokenStore.GetByAccess(access)
	if err != nil {
		return nil, err
	} else if ti == nil || ti.GetAccess() != access {
		return nil, errors.ErrInvalidAccessToken
	} else if ti.GetRefresh() != "" && ti.GetRefreshExpiresIn() != 0 &&
		ti.GetRefreshCreateAt().Add(ti.GetRefreshExpiresIn()).Before(ct) {
		return nil, errors.ErrExpiredRefreshToken
	} else if ti.GetAccessExpiresIn() != 0 &&
		ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn()).Before(ct) {
		return nil, errors.ErrExpiredAccessToken
	}
	return ti, nil
}

// LoadRefreshToken according to the refresh token for corresponding token information
func (m *Manager) LoadRefreshToken(refresh string) (oauth2server.TokenInfo, error) {
	if refresh == "" {
		return nil, errors.ErrInvalidRefreshToken
	}

	ti, err := m.tokenStore.GetByRefresh(refresh)
	if err != nil {
		return nil, err
	} else if ti == nil || ti.GetRefresh() != refresh {
		return nil, errors.ErrInvalidRefreshToken
	} else if ti.GetRefreshExpiresIn() != 0 && // refresh token set to not expire
		ti.GetRefreshCreateAt().Add(ti.GetRefreshExpiresIn()).Before(time.Now()) {
		return nil, errors.ErrExpiredRefreshToken
	}
	return ti, nil
}
