package generates

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/autowp/goautowp/auth/oauth2server"
	"github.com/autowp/goautowp/auth/oauth2server/utils/uuid"
)

// AccessGenerate generate the access token
type AccessGenerate struct {
}

// Token based on the UUID generated token
func (ag *AccessGenerate) Token(data *oauth2server.GenerateBasic, isGenRefresh bool) (string, string, error) {
	buf := bytes.NewBufferString(data.Client.GetID())
	buf.WriteString(strconv.FormatInt(data.UserID, 10))
	buf.WriteString(strconv.FormatInt(data.CreateAt.UnixNano(), 10))

	md5, err := uuid.NewMD5(uuid.Must(uuid.NewRandom()), buf.Bytes())
	if err != nil {
		return "", "", err
	}
	access := base64.URLEncoding.EncodeToString(md5.Bytes())
	access = strings.ToUpper(strings.TrimRight(access, "="))
	refresh := ""
	if isGenRefresh {
		sha1, err := uuid.NewSHA1(uuid.Must(uuid.NewRandom()), buf.Bytes())
		if err != nil {
			return "", "", err
		}
		refresh = base64.URLEncoding.EncodeToString(sha1.Bytes())
		refresh = strings.ToUpper(strings.TrimRight(refresh, "="))
	}

	return access, refresh, nil
}
