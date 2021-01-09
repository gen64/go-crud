package crudl

// ControllerError extends simple error with more details
type ControllerError struct {
	Op string
	Err error
}

func (e *ControllerError) Error() string {
	return e.Err.Error()
}

func (e *ControllerError) Unwrap() error {
	return e.Err
}
