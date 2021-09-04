package server

import (
	"net/http"
	"time"

	"github.com/autowp/goautowp/auth/oauth2server"
	"github.com/autowp/goautowp/auth/oauth2server/errors"
)

type (
	// ClientScopeHandler check the client allows to use scope
	ClientScopeHandler func(clientID, scope string) (allowed bool, err error)

	// UserAuthorizationHandler get user id from request authorization
	UserAuthorizationHandler func(w http.ResponseWriter, r *http.Request) (userID int64, err error)

	// PasswordAuthorizationHandler get user id from username and password
	PasswordAuthorizationHandler func(username, password string) (userID int64, err error)

	// SocialAuthorizationHandler ...
	SocialAuthorizationHandler func(code, stateID, remoteAddr string) (int64, string, error)

	// RefreshingScopeHandler check the scope of the refreshing token
	RefreshingScopeHandler func(newScope, oldScope string) (allowed bool, err error)

	// ResponseErrorHandler response error handing
	ResponseErrorHandler func(re *errors.Response)

	// InternalErrorHandler internal error handing
	InternalErrorHandler func(err error) (re *errors.Response)

	// AccessTokenExpHandler set expiration date for the access token
	AccessTokenExpHandler func(w http.ResponseWriter, r *http.Request) (exp time.Duration, err error)

	// ExtensionFieldsHandler in response to the access token with the extension of the field
	ExtensionFieldsHandler func(ti oauth2server.TokenInfo) (fieldsValue map[string]interface{})
)
