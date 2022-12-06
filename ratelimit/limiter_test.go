package ratelimit

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestZeroDurationThrottler(t *testing.T) {
	throttler := NewThrottler(0)

	// when:
	start := time.Now()
	throttler.Throttle()
	throttler.Throttle()
	throttler.Throttle()
	duration := time.Now().Sub(start)

	//expect:
	require.True(t, duration < 10*time.Second)
}
