package base64x

import (
	"encoding/base64"
	"github.com/joomcode/errorx"
	"io"
	"strings"
)

// TODO: test me
func ReadAllEncodeToString(enc *base64.Encoding, input io.Reader) (string, error) {
	var dataBuilder strings.Builder
	dataEncoder := base64.NewEncoder(enc, &dataBuilder)
	if _, err := io.Copy(dataEncoder, input); err != nil {
		return "", errorx.ExternalError.Wrap(err, "failed to read from input")
	}
	if err := dataEncoder.Close(); err != nil {
		return "", errorx.Panic(errorx.IllegalState.New("failed to write to strings.Builder"))
	}
	return dataBuilder.String(), nil
}
