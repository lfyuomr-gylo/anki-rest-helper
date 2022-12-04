package iox

import (
	"io"
	"log"
)

func Close(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("Error occurred while closing a closer: %+v", err)
	}
}
