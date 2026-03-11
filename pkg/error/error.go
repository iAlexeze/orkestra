package crderror

import "errors"

var (
	ErrFactoryAlreadyStarted = errors.New("factory already started")
	ErrSchemeNill            = errors.New("scheme cannot be nil")
)
