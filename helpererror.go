package crudl

// HelperError wraps original error with operation/step where the error occured
// and optionally with a tag when parsing "crudl" failed
type HelperError struct {
	Op  string
	Tag string
	Err error
}

func (e HelperError) Error() string {
	return e.Err.Error()
}

func (e HelperError) Unwrap() error {
	return e.Err
}
