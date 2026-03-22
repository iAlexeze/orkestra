package utils

import (
	"errors"
	"math/rand/v2"
	"os"
	"time"
)

type Status string

const (
	// Status
	StatusReady   Status = "ready"
	StatusRunning Status = "running"
	StatusHealthy Status = "healthy"
	StatusOnline  Status = "online"

	StatusNotReady   Status = "not ready"
	StatusNotRunning Status = "not running"
	StatusNotHealthy Status = "not healthy"
	StatusOffline    Status = "offline"

	// HTTP
	ContentType     = "Content-Type"
	JSONContentType = "application/json"
)

func Sleep(n int) {
	time.Sleep(time.Duration(n) * time.Second)
}

func BoolPtr(b bool) *bool { return &b }

func Retry(fn func() error, attempts int, delay time.Duration) error {
	if attempts < 1 {
		return errors.New("attempts must be >= 1")
	}

	for i := 1; i <= attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if i == attempts {
			return err
		}

		time.Sleep(delay)
	}

	return nil
}

func Jitter(d time.Duration) time.Duration {
	// ±50% jitter
	j := rand.Float64()*float64(d) - float64(d)/2
	return d + time.Duration(j)
}

func RetryBackoff(fn func() error, attempts int, base time.Duration) error {
	delay := base

	for i := 1; i <= attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if i == attempts {
			return err
		}

		time.Sleep(Jitter(delay))
		delay *= 2
	}

	return nil
}

// Reversed reverses any given slice
func Reversed[T any](s []T) []T {
	out := make([]T, len(s))
	for i := range s {
		out[len(s)-1-i] = s[i]
	}
	return out
}

// Exit exits in error with a code
func Exit(err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
	}
	os.Exit(1)
}
