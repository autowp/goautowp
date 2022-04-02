package util

import (
	"database/sql"
	"github.com/sirupsen/logrus"
	"io"
)

// Close resource and prints error
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		logrus.Error(err)
	}
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func SqlNullInt64ToPtr(v sql.NullInt64) *int64 {
	var r *int64
	if v.Valid {
		return &v.Int64
	}

	return r
}
