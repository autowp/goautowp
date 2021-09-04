package oauth2server

import (
	"time"
)

type (
	// ClientInfo the client information model interface
	ClientInfo interface {
		GetID() string
		GetSecret() string
		GetDomain() string
		GetUserID() string
	}

	// TokenInfo the token information model interface
	TokenInfo interface {
		New() TokenInfo

		GetClientID() string
		SetClientID(string)
		GetUserID() int64
		SetUserID(int64)
		GetScope() string
		SetScope(string)

		GetCode() string
		SetCode(string)
		GetCodeCreateAt() time.Time
		SetCodeCreateAt(time.Time)
		GetCodeExpiresIn() time.Duration
		SetCodeExpiresIn(time.Duration)

		GetAccess() string
		SetAccess(string)
		GetAccessCreateAt() time.Time
		SetAccessCreateAt(time.Time)
		GetAccessExpiresIn() time.Duration
		SetAccessExpiresIn(time.Duration)

		GetRefresh() string
		SetRefresh(string)
		GetRefreshCreateAt() time.Time
		SetRefreshCreateAt(time.Time)
		GetRefreshExpiresIn() time.Duration
		SetRefreshExpiresIn(time.Duration)
	}

	// PasswordCredentialsData ...
	PasswordCredentialsData struct {
		Scope    string `form:"scope"    json:"scope"`
		Username string `form:"username" json:"username"`
		Password string `form:"password" json:"password"`
	}

	// SocialAuthorizationCodeData ...
	SocialAuthorizationCodeData struct {
		Scope string `form:"scope" json:"scope"`
		Code  string `form:"code"  json:"code"`
		State string `form:"state" json:"state"`
	}

	// TokenRequestData ...
	TokenRequestData struct {
		GrantType    string `form:"grant_type"    json:"grant_type"`
		ClientID     string `form:"client_id"     json:"client_id"`
		ClientSecret string `form:"client_secret" json:"client_secret"`
		Username     string `form:"username"      json:"username"`
		Password     string `form:"password"      json:"password"`
		Code         string `form:"code"          json:"code"`
		State        string `form:"state"         json:"state"`
		Scope        string `form:"scope"         json:"scope"`
		RefreshToken string `form:"refresh_token" json:"refresh_token"`
		ClientIP     string
	}
)
