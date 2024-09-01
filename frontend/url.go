package frontend

import (
	"net/url"
	"strconv"
)

const (
	picturePrefix = "picture"
	usersPrefix   = "users"
)

func PicturePath(identity string) string {
	return "/" + picturePrefix + "/" + url.QueryEscape(identity)
}

func PictureURL(url *url.URL, identity string) string {
	url.Path = PicturePath(identity)

	return url.String()
}

func PictureRoute(identity string) []string {
	return []string{"/" + picturePrefix, identity}
}

func PictureModerURL(url *url.URL, pictureID int64) string {
	url.Path = "/moder/pictures/" + strconv.FormatInt(pictureID, 10)

	return url.String()
}

func userIdentity(userID int64, identity *string) string {
	if identity == nil || len(*identity) == 0 {
		return "user" + strconv.FormatInt(userID, 10)
	}

	return *identity
}

func UserPath(userID int64, identity *string) string {
	return "/" + usersPrefix + "/" + url.QueryEscape(userIdentity(userID, identity))
}

func UserURL(url *url.URL, userID int64, identity *string) string {
	url.Path = UserPath(userID, identity)

	return url.String()
}

func UserRoute(userID int64, identity *string) []string {
	return []string{"/" + usersPrefix, userIdentity(userID, identity)}
}
