package password

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "too short",
			password: "Aa1!",
			wantErr:  errPasswordTooShort,
		},
		{
			name:     "too long",
			password: "Aa1!" + string(make([]byte, 125)),
			wantErr:  errPasswordTooLong,
		},
		{
			name:     "invalid characters",
			password: "ValidPassword1! ",
			wantErr:  errPasswordInvalidChars,
		},
		{
			name:     "missing uppercase",
			password: "validpassword1!",
			wantErr:  errPasswordMissingUpper,
		},
		{
			name:     "missing lowercase",
			password: "VALIDPASSWORD1!",
			wantErr:  errPasswordMissingLower,
		},
		{
			name:     "missing digit",
			password: "ValidPassword!",
			wantErr:  errPasswordMissingDigit,
		},
		{
			name:     "missing special",
			password: "ValidPassword123",
			wantErr:  errPasswordMissingSpecial,
		},
		{
			name:     "valid",
			password: "ValidPassword123!",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.password)

			if err != tt.wantErr {
				t.Errorf(
					"ValidatePassword() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
