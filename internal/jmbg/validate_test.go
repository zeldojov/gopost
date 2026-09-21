package jmbg

import "testing"

func TestValidateJMBG(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want bool
	}{
		{
			name: "valid JMBG",
			jmbg: "1505994711235",
			want: true,
		},
		{
			name: "too short",
			jmbg: "150594571234",
			want: false,
		},
		{
			name: "contains letter",
			jmbg: "15059457A2345",
			want: false,
		},
		{
			name: "invalid date",
			jmbg: "3104945712345",
			want: false,
		},
		{
			name: "invalid region",
			jmbg: "1505940012345",
			want: false,
		},
		{
			name: "invalid checksum",
			jmbg: "1505945712346",
			want: false,
		},
		{
			name: "unsupported century",
			jmbg: "0101100710132",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateJMBG(tt.jmbg); got != tt.want {
				t.Errorf("ValidateJMBG(%q) = %v, want %v", tt.jmbg, got, tt.want)
			}
		})
	}
}

func TestJMBGChecksum(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want int
	}{
		{
			name: "remainder 0",
			jmbg: "010100071013",
			want: 0,
		},
		{
			name: "remainder 1",
			jmbg: "010100071005",
			want: -1,
		},
		{
			name: "remainder 2",
			jmbg: "010100071000",
			want: 9,
		},
		{
			name: "remainder 3",
			jmbg: "010100071006",
			want: 8,
		},
		{
			name: "remainder 4",
			jmbg: "010100071001",
			want: 7,
		},
		{
			name: "remainder 5",
			jmbg: "010100071007",
			want: 6,
		},
		{
			name: "remainder 6",
			jmbg: "010100071002",
			want: 5,
		},
		{
			name: "remainder 7",
			jmbg: "010100071008",
			want: 4,
		},
		{
			name: "remainder 8",
			jmbg: "010100071003",
			want: 3,
		},
		{
			name: "remainder 9",
			jmbg: "010100071009",
			want: 2,
		},
		{
			name: "remainder 10",
			jmbg: "010100071004",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jmbgChecksum(tt.jmbg); got != tt.want {
				t.Errorf("jmbgChecksum(%q) = %d, want %d", tt.jmbg, got, tt.want)
			}
		})
	}
}

func TestValidJMBGChecksum(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want bool
	}{
		{
			name: "checksum is invalid",
			jmbg: "0101000710050",
			want: false,
		},
		{
			name: "checksum digit is incorrect",
			jmbg: "0101000710131",
			want: false,
		},
		{
			name: "checksum is valid",
			jmbg: "0101000710130",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validJMBGChecksum(tt.jmbg); got != tt.want {
				t.Errorf("validJMBGChecksum(%q) = %v, want %v", tt.jmbg, got, tt.want)
			}
		})
	}
}

func TestIsJMBGDigits(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want bool
	}{
		{
			name: "all digits",
			jmbg: "2812980710132",
			want: true,
		},
		{
			name: "contains letter",
			jmbg: "281298071013A",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJMBGDigits(tt.jmbg)

			if got != tt.want {
				t.Errorf("isJMBGDigits(%q) = %v, want %v", tt.jmbg, got, tt.want)
			}
		})
	}
}
func TestValidJMBGDate(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want bool
	}{
		{
			name: "valid date",
			jmbg: "2812980710132",
			want: true,
		},
		{
			name: "invalid date",
			jmbg: "3113980710132",
			want: false,
		},
		{
			name: "invalid year prefix",
			jmbg: "2818180710132",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validJMBGDate(tt.jmbg)

			if got != tt.want {
				t.Errorf("validJMBGDate(%q) = %v, want %v",
					tt.jmbg, got, tt.want)
			}
		})
	}
}

func TestValidJMBGRegion(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want bool
	}{
		{
			name: "valid region",
			jmbg: "2812980710132",
			want: true,
		},
		{
			name: "invalid region",
			jmbg: "2812989990132",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validJMBGRegion(tt.jmbg)

			if got != tt.want {
				t.Errorf("validJMBGRegion(%q) = %v, want %v",
					tt.jmbg, got, tt.want)
			}
		})
	}
}
