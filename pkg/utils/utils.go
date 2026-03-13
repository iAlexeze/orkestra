package utils

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
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

// LoadFile loads a file from local disk or HTTP(S)
func LoadFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	// Remote file
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to download %s: %w", path, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("remote file returned status %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}

	// Local file
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", path)
	}

	return os.ReadFile(path)
}
