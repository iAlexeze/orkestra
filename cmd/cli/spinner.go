package cli

import "github.com/orkspace/orkestra/pkg/spinner"

// StartSpinner starts a terminal progress spinner with the given message.
// Call Success, Failure, or Stop on the returned value when done.
func StartSpinner(msg string) *spinner.Spinner {
	return spinner.Start(msg)
}
