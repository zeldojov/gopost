package user

import "errors"

var (
	errPasswordTooShort       = errors.New("password must be at least 12 characters")
	errPasswordTooLong        = errors.New("password must be at most 128 characters")
	errPasswordInvalidChars   = errors.New("password contains invalid characters")
	errPasswordMissingUpper   = errors.New("password must contain an uppercase letter")
	errPasswordMissingLower   = errors.New("password must contain a lowercase letter")
	errPasswordMissingDigit   = errors.New("password must contain a digit")
	errPasswordMissingSpecial = errors.New("password must contain a special character")

	errUsernameTooShort          = errors.New("username must be at least 3 characters")
	errUsernameTooLong           = errors.New("username must be at most 32 characters")
	errUsernameInvalidChars      = errors.New("username contains invalid characters")
	errUsernameInvalidUnderscore = errors.New("username contains invalid underscore placement")
)

func ValidatePassword(password string) error {
	if len(password) < 12 {
		return errPasswordTooShort
	}

	if len(password) > 128 {
		return errPasswordTooLong
	}

	var upper, lower, digit, special bool

	for i := 0; i < len(password); i++ {
		c := password[i]

		switch {
		case c >= 65 && c <= 90:
			upper = true

		case c >= 97 && c <= 122:
			lower = true

		case c >= 48 && c <= 57:
			digit = true

		case c >= 33 && c <= 47 ||
			c >= 58 && c <= 64 ||
			c >= 91 && c <= 96 ||
			c >= 123 && c <= 126:
			special = true

		default:
			return errPasswordInvalidChars
		}
	}

	if !upper {
		return errPasswordMissingUpper
	}

	if !lower {
		return errPasswordMissingLower
	}

	if !digit {
		return errPasswordMissingDigit
	}

	if !special {
		return errPasswordMissingSpecial
	}

	return nil
}

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
