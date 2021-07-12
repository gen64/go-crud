package crud

// ErrHelper wraps original error with operation/step where the error occured
// and optionally with a tag when parsing "crud" failed
type ErrHelper struct {
	Op  string
	Tag string
	Err error
}

func (e ErrHelper) Error() string {
	return e.Err.Error()
}

func (e ErrHelper) Unwrap() error {
	return e.Err
}
