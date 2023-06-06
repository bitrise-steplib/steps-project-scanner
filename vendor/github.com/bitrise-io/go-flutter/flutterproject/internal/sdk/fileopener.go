package sdk

import (
	"io"
)

type FileOpener interface {
	OpenReaderIfExists(path string) (io.Reader, error)
}
