package crderror

import "errors"

var (
	ErrFactoryAlreadyStarted = errors.New("factory already started")
)
