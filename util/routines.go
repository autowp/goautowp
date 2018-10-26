package util

import (
	"io"
	"log"
)

// Close resource and prints error
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Printf("%v\n", err)
	}
}
