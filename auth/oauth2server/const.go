package oauth2server

// ResponseType the type of authorization request
type ResponseType string

// define the type of authorization request
const (
	Token ResponseType = "token"
)

func (rt ResponseType) String() string {
	if rt == Token {
		return string(rt)
	}
	return ""
}

// GrantType authorization model
type GrantType string

// define authorization model
const (
	PasswordCredentials     GrantType = "password"
	Refreshing              GrantType = "refresh_token"
	SocialAuthorizationCode GrantType = "social_authorization_code"
)

func (gt GrantType) String() string {
	if gt == PasswordCredentials ||
		gt == SocialAuthorizationCode ||
		gt == Refreshing {
		return string(gt)
	}
	return ""
}
