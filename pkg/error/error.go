package orkerror

import "errors"

var (
	ErrFactoryAlreadyStarted = errors.New("factory already started")
	ErrSchemeNill            = errors.New("scheme cannot be nil")
	ErrCRDNotFound           = errors.New("CRD not found")
)
