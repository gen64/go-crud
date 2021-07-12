package crud

// ErrValidation wraps error occuring during object validation
type ErrValidation struct {
	Fields []string
	Err    error
}

func (e ErrValidation) Error() string {
	return e.Err.Error()
}

func (e ErrValidation) Unwrap() error {
	return e.Err
}
