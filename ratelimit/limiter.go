package ratelimit

import (
	"time"
)

func NewThrottler(delay time.Duration) *Throttler {
	return &Throttler{
		delay: delay,
		timer: time.NewTimer(delay),
	}
}

type Throttler struct {
	delay time.Duration
	timer *time.Timer
}

func (l *Throttler) Throttle() {
	_ = <-l.timer.C
	l.timer.Reset(l.delay)
	return
}
