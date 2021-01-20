package crud

// ControllerError wraps original error that occurred in Err with name of the
// operation/step that failed, which is in Op field
type ControllerError struct {
	Op  string
	Err error
}

func (e *ControllerError) Error() string {
	return e.Err.Error()
}

func (e *ControllerError) Unwrap() error {
	return e.Err
}
