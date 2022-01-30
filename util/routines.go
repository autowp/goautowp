package util

import (
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
