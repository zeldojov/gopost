package jmbg

import "errors"

var (
	ErrInvalidLength   = errors.New("invalid JMBG length")
	ErrInvalidDigits   = errors.New("JMBG must contain only digits")
	ErrInvalidDate     = errors.New("invalid JMBG date")
	ErrInvalidRegion   = errors.New("invalid JMBG region")
	ErrInvalidChecksum = errors.New("invalid JMBG checksum")
)
