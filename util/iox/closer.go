package iox

import (
	"errors"
	"io"
	"log"
	"os"
)

func Close(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		if errors.Is(err, os.ErrClosed) {
			// it's fine trying to close an already closed file, so there's no need to
			// spam such errors in stdout
			return
		}
		log.Printf("Error occurred while closing a closer: %+v", err)
	}
}
