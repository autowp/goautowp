package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/autowp/goautowp/auth/oauth2server"
	"github.com/autowp/goautowp/auth/oauth2server/errors"
	"github.com/gin-gonic/gin"
)

// NewServer create authorization server
func NewServer(manager oauth2server.Manager) *Server {
	srv := &Server{
		Manager: manager,
	}

	srv.UserAuthorizationHandler = func(w http.ResponseWriter, r *http.Request) (int64, error) {
		return 0, errors.ErrAccessDenied
	}

	srv.PasswordAuthorizationHandler = func(username, password string) (int64, error) {
		return 0, errors.ErrAccessDenied
	}

	srv.SocialAuthorizationHandler = func(code, stateID, remoteAddr string) (int64, string, error) {
		return 0, "", errors.ErrAccessDenied
	}

	return srv
}

// Server Provide authorization server
type Server struct {
	Manager                      oauth2server.Manager
	ClientScopeHandler           ClientScopeHandler
	UserAuthorizationHandler     UserAuthorizationHandler
	PasswordAuthorizationHandler PasswordAuthorizationHandler
	SocialAuthorizationHandler   SocialAuthorizationHandler
	RefreshingScopeHandler       RefreshingScopeHandler
	ResponseErrorHandler         ResponseErrorHandler
	InternalErrorHandler         InternalErrorHandler
	ExtensionFieldsHandler       ExtensionFieldsHandler
	AccessTokenExpHandler        AccessTokenExpHandler
}

// TokenError ...
func (s *Server) TokenError(c *gin.Context, err error) {
	data, statusCode, header := s.GetErrorData(err)
	s.Token(c, data, header, statusCode)
}

// Token ...
func (s *Server) Token(c *gin.Context, data map[string]interface{}, header http.Header, statusCode int) {
	c.Header("Content-Type", "application/json;charset=UTF-8")
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	for key := range header {
		c.Header(key, header.Get(key))
	}

	status := http.StatusOK
	if statusCode > 0 {
		status = statusCode
	}

	c.JSON(status, data)
}

// ValidationTokenRequest the token request validation
func (s *Server) ValidationTokenRequest(c *gin.Context, trd *oauth2server.TokenRequestData) (oauth2server.GrantType, *oauth2server.TokenGenerateRequest, string, error) {
	if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodGet {
		return "", nil, "", errors.ErrInvalidRequest
	}

	gt := oauth2server.GrantType(trd.GrantType)
	if gt.String() == "" {
		return "", nil, "", errors.ErrUnsupportedGrantType
	}

	if trd.ClientID == "" || trd.ClientSecret == "" {
		return "", nil, "", errors.ErrInvalidClient
	}

	tgr := &oauth2server.TokenGenerateRequest{
		ClientID:     trd.ClientID,
		ClientSecret: trd.ClientSecret,
		Request:      c.Request,
	}

	redirectURI := ""

	switch gt {

	case oauth2server.PasswordCredentials:
		tgr.Scope = trd.Scope

		userID, err := s.PasswordAuthorizationHandler(trd.Username, trd.Password)
		if err != nil {
			return "", nil, "", err
		}
		if userID == 0 {
			return "", nil, "", errors.ErrInvalidGrant
		}
		tgr.UserID = userID
	case oauth2server.SocialAuthorizationCode:
		tgr.Scope = trd.Scope

		var userID int64
		var err error

		userID, redirectURI, err = s.SocialAuthorizationHandler(trd.Code, trd.State, trd.ClientIP)
		if err != nil {
			return "", nil, "", err
		}
		if userID == 0 {
			return "", nil, "", errors.ErrInvalidGrant
		}
		tgr.UserID = userID
	case oauth2server.Refreshing:
		tgr.Refresh = trd.RefreshToken
		tgr.Scope = trd.Scope
		if tgr.Refresh == "" {
			return "", nil, "", errors.ErrInvalidRequest
		}
	}
	return gt, tgr, redirectURI, nil
}

// GetAccessToken access token
func (s *Server) GetAccessToken(gt oauth2server.GrantType, tgr *oauth2server.TokenGenerateRequest) (oauth2server.TokenInfo, error) {
	switch gt {

	case oauth2server.PasswordCredentials, oauth2server.SocialAuthorizationCode:
		if fn := s.ClientScopeHandler; fn != nil {
			allowed, err := fn(tgr.ClientID, tgr.Scope)
			if err != nil {
				return nil, err
			} else if !allowed {
				return nil, errors.ErrInvalidScope
			}
		}
		return s.Manager.GenerateAccessToken(gt, tgr)
	case oauth2server.Refreshing:
		// check scope
		if scope, scopeFn := tgr.Scope, s.RefreshingScopeHandler; scope != "" && scopeFn != nil {
			rti, err := s.Manager.LoadRefreshToken(tgr.Refresh)
			if err != nil {
				if err == errors.ErrInvalidRefreshToken || err == errors.ErrExpiredRefreshToken {
					return nil, errors.ErrInvalidGrant
				}
				return nil, err
			}

			allowed, err := scopeFn(scope, rti.GetScope())
			if err != nil {
				return nil, err
			} else if !allowed {
				return nil, errors.ErrInvalidScope
			}
		}

		ti, err := s.Manager.RefreshAccessToken(tgr)
		if err != nil {
			if err == errors.ErrInvalidRefreshToken || err == errors.ErrExpiredRefreshToken {
				return nil, errors.ErrInvalidGrant
			}
			return nil, err
		}
		return ti, nil
	default:
		return nil, errors.ErrUnsupportedGrantType
	}
}

// GetTokenData token data
func (s *Server) GetTokenData(ti oauth2server.TokenInfo) map[string]interface{} {
	data := map[string]interface{}{
		"access_token": ti.GetAccess(),
		"token_type":   "Bearer",
		"expires_in":   int64(ti.GetAccessExpiresIn() / time.Second),
	}

	if scope := ti.GetScope(); scope != "" {
		data["scope"] = scope
	}

	if refresh := ti.GetRefresh(); refresh != "" {
		data["refresh_token"] = refresh
	}

	if fn := s.ExtensionFieldsHandler; fn != nil {
		ext := fn(ti)
		for k, v := range ext {
			if _, ok := data[k]; ok {
				continue
			}
			data[k] = v
		}
	}
	return data
}

// GetErrorData get error response data
func (s *Server) GetErrorData(err error) (map[string]interface{}, int, http.Header) {
	var re errors.Response
	if v, ok := errors.Descriptions[err]; ok {
		re.Error = err
		re.Description = v
		re.StatusCode = errors.StatusCodes[err]
	} else {
		if fn := s.InternalErrorHandler; fn != nil {
			if v := fn(err); v != nil {
				re = *v
			}
		}

		if re.Error == nil {
			re.Error = errors.ErrServerError
			//re.Description = errors.Descriptions[errors.ErrServerError]
			re.Description = err.Error()
			re.StatusCode = errors.StatusCodes[errors.ErrServerError]
		}
	}

	if fn := s.ResponseErrorHandler; fn != nil {
		fn(&re)
	}

	data := make(map[string]interface{})
	if err := re.Error; err != nil {
		data["error"] = err.Error()
	}

	if v := re.ErrorCode; v != 0 {
		data["error_code"] = v
	}

	if v := re.Description; v != "" {
		data["error_description"] = v
	}

	if v := re.URI; v != "" {
		data["error_uri"] = v
	}

	statusCode := http.StatusInternalServerError
	if v := re.StatusCode; v > 0 {
		statusCode = v
	}

	return data, statusCode, re.Header
}

// BearerAuth parse bearer token
func (s *Server) BearerAuth(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	prefix := "Bearer "
	token := ""

	if auth != "" && strings.HasPrefix(auth, prefix) {
		token = auth[len(prefix):]
	}

	return token, token != ""
}

// ValidationBearerToken validation the bearer tokens
// https://tools.ietf.org/html/rfc6750
func (s *Server) ValidationBearerToken(r *http.Request) (oauth2server.TokenInfo, error) {
	accessToken, ok := s.BearerAuth(r)
	if !ok {
		return nil, errors.ErrInvalidAccessToken
	}

	return s.Manager.LoadAccessToken(accessToken)
}
