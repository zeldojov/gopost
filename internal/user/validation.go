package user

import (
	"errors"
)

var (
	errUsernameTooShort          = errors.New("username must be at least 3 characters")
	errUsernameTooLong           = errors.New("username must be at most 32 characters")
	errUsernameInvalidChars      = errors.New("username contains invalid characters")
	errUsernameInvalidUnderscore = errors.New("username contains invalid underscore placement")
)

func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errUsernameTooShort
	}

	if len(username) > 32 {
		return errUsernameTooLong
	}

	for i := 0; i < len(username); i++ {
		c := username[i]

		switch {
		case c >= 65 && c <= 90:
		case c >= 97 && c <= 122:
		case c >= 48 && c <= 57:

		case c == 95:
			if i == 0 || i == len(username)-1 {
				return errUsernameInvalidUnderscore
			}

			if username[i-1] == 95 {
				return errUsernameInvalidUnderscore
			}

		default:
			return errUsernameInvalidChars
		}
	}

	return nil
}
