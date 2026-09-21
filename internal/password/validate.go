package password

func Validate(password string) error {
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
