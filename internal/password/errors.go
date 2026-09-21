package password

import "errors"

var (
	errPasswordTooShort       = errors.New("password must be at least 12 characters")
	errPasswordTooLong        = errors.New("password must be at most 128 characters")
	errPasswordInvalidChars   = errors.New("password contains invalid characters")
	errPasswordMissingUpper   = errors.New("password must contain an uppercase letter")
	errPasswordMissingLower   = errors.New("password must contain a lowercase letter")
	errPasswordMissingDigit   = errors.New("password must contain a digit")
	errPasswordMissingSpecial = errors.New("password must contain a special character")
)
