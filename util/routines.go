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

func StrPtrToStr(ptr *string) string {
	if ptr == nil {
		return ""
	}

	return *ptr
}
