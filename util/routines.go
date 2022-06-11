package util

import (
	"database/sql"
	"io"

	"github.com/sirupsen/logrus"
)

// Close resource and prints error.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
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

func SQLNullInt64ToPtr(v sql.NullInt64) *int64 {
	var r *int64

	if v.Valid {
		return &v.Int64
	}

	return r
}
