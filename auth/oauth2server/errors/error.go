package errors

import "errors"

// known errors
var (
	ErrInvalidAccessToken  = errors.New("invalid access token")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrExpiredAccessToken  = errors.New("expired access token")
	ErrExpiredRefreshToken = errors.New("expired refresh token")
)
