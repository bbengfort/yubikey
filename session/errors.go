package session

import "errors"

var (
	ErrInsufficientBytesRead = errors.New("insufficient bytes read")
	ErrMarshal               = errors.New("error unmarshaling data")
)
