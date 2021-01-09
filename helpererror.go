package crudl

// HelperError has details on failure in reflecting the struct
type HelperError struct {
	Op  string
	Tag string
	err error
}

func (e HelperError) Error() string {
	return e.err.Error()
}

func (e HelperError) Unwrap() error {
	return e.err
}
