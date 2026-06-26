//go:build !runtime && !gateway

package cli

import "github.com/orkspace/orkestra/pkg/utils"

// StartSpinner starts a terminal progress spinner with the given message.
// Call Success, Failure, or Stop on the returned value when done.
func StartSpinner(msg string) *utils.Spinner {
	return utils.StartSpinner(msg)
}
