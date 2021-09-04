package server

import (
	"net/http"
	"time"

	"github.com/autowp/goautowp/auth/oauth2server"
)

// AuthorizeRequest authorization request
type AuthorizeRequest struct {
	ResponseType   oauth2server.ResponseType
	ClientID       string
	Scope          string
	State          string
	UserID         string
	AccessTokenExp time.Duration
	Request        *http.Request
}
