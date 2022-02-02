package httputil

import (
	"anki-rest-enhancer/ratelimit"
	"net/http"
	"time"
)

func NewThrottlingTransport(transport http.RoundTripper, delay time.Duration) http.RoundTripper {
	throttler := ratelimit.NewThrottler(delay)
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		throttler.Throttle()
		return transport.RoundTrip(req)
	})
}
