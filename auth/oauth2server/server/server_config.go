package server

// SetClientScopeHandler check the client allows to use scope
func (s *Server) SetClientScopeHandler(handler ClientScopeHandler) {
	s.ClientScopeHandler = handler
}

// SetUserAuthorizationHandler get user id from request authorization
func (s *Server) SetUserAuthorizationHandler(handler UserAuthorizationHandler) {
	s.UserAuthorizationHandler = handler
}

// SetPasswordAuthorizationHandler get user id from username and password
func (s *Server) SetPasswordAuthorizationHandler(handler PasswordAuthorizationHandler) {
	s.PasswordAuthorizationHandler = handler
}

// SetSocialAuthorizationHandler get user id from social network
func (s *Server) SetSocialAuthorizationHandler(handler SocialAuthorizationHandler) {
	s.SocialAuthorizationHandler = handler
}

// SetRefreshingScopeHandler check the scope of the refreshing token
func (s *Server) SetRefreshingScopeHandler(handler RefreshingScopeHandler) {
	s.RefreshingScopeHandler = handler
}

// SetResponseErrorHandler response error handling
func (s *Server) SetResponseErrorHandler(handler ResponseErrorHandler) {
	s.ResponseErrorHandler = handler
}

// SetInternalErrorHandler internal error handling
func (s *Server) SetInternalErrorHandler(handler InternalErrorHandler) {
	s.InternalErrorHandler = handler
}

// SetExtensionFieldsHandler in response to the access token with the extension of the field
func (s *Server) SetExtensionFieldsHandler(handler ExtensionFieldsHandler) {
	s.ExtensionFieldsHandler = handler
}

// SetAccessTokenExpHandler set expiration date for the access token
func (s *Server) SetAccessTokenExpHandler(handler AccessTokenExpHandler) {
	s.AccessTokenExpHandler = handler
}
