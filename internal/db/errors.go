package db

import "errors"

// Sentinel errors for the db package.
// Use errors.Is() to check for these.
var (
	ErrFilterNotFound   = errors.New("filter not found")
	ErrJobNotFound      = errors.New("job not found")
	ErrJobMatchNotFound = errors.New("job match not found")
	ErrDuplicateJob     = errors.New("job already exists")
	ErrInvalidLimit     = errors.New("invalid limit value")
	ErrInvalidInput     = errors.New("invalid input")
)
