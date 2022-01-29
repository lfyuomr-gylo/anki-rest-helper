package httputil

import (
	"bytes"
	"github.com/joomcode/errorx"
	"io"
	"log"
	"net/http"
	"strings"
)

func NewLoggingRoundTripper(transport http.RoundTripper) *loggingRoundTripper {
	return &loggingRoundTripper{transport: transport}
}

type loggingRoundTripper struct {
	transport http.RoundTripper
}

var _ http.RoundTripper = (*loggingRoundTripper)(nil)

func (rt loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	logReq(req)
	resp, err := rt.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if err := logResp(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func logReq(req *http.Request) {
	var buf strings.Builder
	buf.WriteString("Request:\n")

	body, _ := io.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	_ = req.Write(&buf)
	req.Body = io.NopCloser(bytes.NewReader(body))

	log.Println(buf.String())
}

func logResp(resp *http.Response) error {
	var buf strings.Builder
	buf.WriteString("Response:\n")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorx.ExternalError.Wrap(err, "failed to read Azure response body")
	}
	_ = resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	_ = resp.Write(&buf)
	resp.Body = io.NopCloser(bytes.NewReader(body))

	log.Println(buf.String())
	return nil
}
