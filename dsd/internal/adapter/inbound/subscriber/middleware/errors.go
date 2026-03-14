package middleware

import "errors"

// SkipRetryError adalah error wrapper yang menandakan bahwa error ini tidak boleh di-retry
type SkipRetryError struct {
	Err error
}

func (e *SkipRetryError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "skip retry error"
}

func (e *SkipRetryError) Unwrap() error {
	return e.Err
}

// NewSkipRetryError membuat error baru yang akan skip retry
func NewSkipRetryError(err error) error {
	return &SkipRetryError{Err: err}
}

// IsSkipRetryError mengecek apakah error adalah SkipRetryError
func IsSkipRetryError(err error) bool {
	var skipErr *SkipRetryError
	return errors.As(err, &skipErr)
}