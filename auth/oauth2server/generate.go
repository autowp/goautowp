package oauth2server

import (
	"net/http"
	"time"
)

type (
	// GenerateBasic provide the basis of the generated token data
	GenerateBasic struct {
		Client    ClientInfo
		UserID    int64
		CreateAt  time.Time
		TokenInfo TokenInfo
		Request   *http.Request
	}

	// AccessGenerate generate the access and refresh tokens interface
	AccessGenerate interface {
		Token(data *GenerateBasic, isGenRefresh bool) (access, refresh string, err error)
	}
)
