// pkg/spinner/spinner.go
package spinner

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// frames is the animation sequence used for the spinner.
var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner provides an animated progress indicator on the terminal.
type Spinner struct {
	message string        // current message to display
	stop    chan struct{} // closed when spinner should stop
	stopped bool          // true after Stop() is called
	mu      sync.Mutex
}

// Start begins a new spinner with the given message.
// If stdout is not a terminal, Start prints the message and returns a no‑op spinner.
func Start(message string) *Spinner {
	// Check if we are writing to a terminal (TTY)
	if !isTerminal() {
		fmt.Println(message)
		return &Spinner{stopped: true}
	}

	s := &Spinner{
		message: message,
		stop:    make(chan struct{}),
	}
	go s.run()
	return s
}

// run is the goroutine that updates the spinner animation.
func (s *Spinner) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-ticker.C:
			// Overwrite current line with spinning frame + message
			fmt.Printf("\r\x1b[K%s %s", frames[idx], s.getMessage())
			idx = (idx + 1) % len(frames)
		case <-s.stop:
			// Clear the line before exiting
			fmt.Print("\r\x1b[K")
			return
		}
	}
}

// getMessage safely returns the current message.
func (s *Spinner) getMessage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.message
}

// UpdateMessage changes the message displayed while the spinner is running.
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Stop terminates the spinner and clears its line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.stop)
}

// WithSuccess stops the spinner and prints a success checkmark with the final message.
// It also restores the line.
func (s *Spinner) WithSuccess() {
	s.Stop()
	if !s.stopped { // if we were already stopped, nothing to do
		return
	}
	if isTerminal() {
		fmt.Printf("\r\x1b[K ✓ %s\n", s.getMessage())
	} else {
		fmt.Printf("✓ %s\n", s.getMessage())
	}
}

// WithFailure stops the spinner and prints a failure cross with the final message.
func (s *Spinner) WithFailure() {
	s.Stop()
	if !s.stopped {
		return
	}
	if isTerminal() {
		fmt.Printf("\r\x1b[K❌ %s\n", s.getMessage())
	} else {
		fmt.Printf("❌ %s\n", s.getMessage())
	}
}

// isTerminal reports whether stdout is a terminal (TTY).
func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// On Unix, mode & ModeCharDevice != 0 indicates a terminal.
	// On Windows, we could use a more robust check, but for simplicity we keep this.
	return (info.Mode() & os.ModeCharDevice) != 0
}
