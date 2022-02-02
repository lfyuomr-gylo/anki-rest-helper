package httputil

import "net/http"

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

var _ http.RoundTripper = (RoundTripperFunc)(nil)

func (r RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
