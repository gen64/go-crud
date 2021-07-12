package crud

// ErrController wraps original error that occurred in Err with name of the
// operation/step that failed, which is in Op field
type ErrController struct {
	Op  string
	Err error
}

func (e *ErrController) Error() string {
	return e.Err.Error()
}

func (e *ErrController) Unwrap() error {
	return e.Err
}
