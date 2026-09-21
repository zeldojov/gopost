package password

import (
	"strings"
	"testing"
)

func TestVerifyPassword(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	encodedHash, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []struct {
		name        string
		password    string
		encodedHash string
		want        bool
	}{
		{
			name:        "valid password",
			password:    password,
			encodedHash: encodedHash,
			want:        true,
		},
		{
			name:        "wrong password",
			password:    "WrongPassword123!",
			encodedHash: encodedHash,
			want:        false,
		},
		{
			name:        "invalid number of parts",
			password:    password,
			encodedHash: "$argon2id$v=19$m=65536,t=3,p=2$abc",
			want:        false,
		},
		{
			name:        "invalid algorithm",
			password:    password,
			encodedHash: strings.Replace(encodedHash, "$argon2id$", "$argon2i$", 1),
			want:        false,
		},
		{
			name:        "invalid version",
			password:    password,
			encodedHash: strings.Replace(encodedHash, "$v=19$", "$v=18$", 1),
			want:        false,
		},
		{
			name:        "invalid parameters format",
			password:    password,
			encodedHash: "$argon2id$v=19$invalid$YWJjZGVmZ2hpamtsbW5vcA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo",
			want:        false,
		},
		{
			name:     "invalid memory",
			password: password,
			encodedHash: strings.Replace(
				encodedHash,
				"m=65536",
				"m=32768",
				1,
			),
			want: false,
		},
		{
			name:     "invalid iterations",
			password: password,
			encodedHash: strings.Replace(
				encodedHash,
				"t=3",
				"t=2",
				1,
			),
			want: false,
		},
		{
			name:     "invalid parallelism",
			password: password,
			encodedHash: strings.Replace(
				encodedHash,
				"p=2",
				"p=1",
				1,
			),
			want: false,
		},
		{
			name:     "invalid salt base64",
			password: password,
			encodedHash: strings.Join(
				append(strings.Split(encodedHash, "$")[:4], "!!!", strings.Split(encodedHash, "$")[5]),
				"$",
			),
			want: false,
		},
		{
			name:     "invalid salt length",
			password: password,
			encodedHash: func() string {
				parts := strings.Split(encodedHash, "$")
				parts[4] = "YWJj"
				return strings.Join(parts, "$")
			}(),
			want: false,
		},
		{
			name:     "invalid hash base64",
			password: password,
			encodedHash: func() string {
				parts := strings.Split(encodedHash, "$")
				parts[5] = "!!!"
				return strings.Join(parts, "$")
			}(),
			want: false,
		},
		{
			name:     "invalid hash length",
			password: password,
			encodedHash: func() string {
				parts := strings.Split(encodedHash, "$")
				parts[5] = "YWJj"
				return strings.Join(parts, "$")
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyPassword(tt.password, tt.encodedHash)

			if got != tt.want {
				t.Errorf(
					"VerifyPassword() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}
