package auth

// ExternalService ...
type ExternalService string

// define authorization model
const (
	Google   ExternalService = "google-plus"
	Facebook ExternalService = "facebook"
	VK       ExternalService = "vk"
)

func (gt ExternalService) String() string {
	if gt == Google ||
		gt == Facebook ||
		gt == VK {
		return string(gt)
	}
	return ""
}

// UserInfo ...
type UserInfo struct {
	ID   string
	Name string
	URL  string
}

// FacebookUser ...
type FacebookUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VKGetUsers ...
type VKGetUsers struct {
	Response []VKUser `json:"response"`
}

// VKUser ...
type VKUser struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ScreenName string `json:"screen_name"`
}
