package jmbg

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestJMBGYear(t *testing.T) {
	tests := []struct {
		name string
		jmbg string
		want int
		ok   bool
	}{
		{
			name: "1900s",
			jmbg: "2812980710132",
			want: 1980,
			ok:   true,
		},
		{
			name: "2000s",
			jmbg: "0101001710132",
			want: 2001,
			ok:   true,
		},
		{
			name: "invalid century prefix",
			jmbg: "2818180710132",
			want: 0,
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jmbgYear(tt.jmbg)

			if got != tt.want || ok != tt.ok {
				t.Errorf("jmbgYear(%q) = (%d, %v), want (%d, %v)",
					tt.jmbg, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestJMBGGender(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber int
		want         JMBGGender
	}{
		{
			name:         "male",
			serialNumber: 499,
			want:         JMBGMale,
		},
		{
			name:         "female",
			serialNumber: 500,
			want:         JMBGFemale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jmbgGender(tt.serialNumber)

			if got != tt.want {
				t.Errorf("jmbgGender(%d) = %v, want %v",
					tt.serialNumber, got, tt.want)
			}
		})
	}
}

func TestParseJMBG(t *testing.T) {
	tests := []struct {
		name    string
		jmbg    string
		want    *JMBG
		wantErr error
	}{
		{
			name:    "invalid length",
			jmbg:    "123",
			wantErr: ErrInvalidLength,
		},
		{
			name:    "invalid digits",
			jmbg:    "15059457A2345",
			wantErr: ErrInvalidDigits,
		},
		{
			name:    "invalid date",
			jmbg:    "3104945712345",
			wantErr: ErrInvalidDate,
		},
		{
			name:    "invalid region",
			jmbg:    "1505940012345",
			wantErr: ErrInvalidRegion,
		},
		{
			name:    "invalid checksum",
			jmbg:    "1505945712346",
			wantErr: ErrInvalidChecksum,
		},
		{
			name: "valid JMBG",
			jmbg: "1505994711235",
			want: &JMBG{
				Date:         time.Date(1994, 5, 15, 0, 0, 0, 0, time.UTC),
				Region:       jmbgRegions["71"],
				RegionCode:   71,
				Gender:       JMBGMale,
				SerialNumber: 123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJMBG(tt.jmbg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseJMBG(%q) error = %v, want %v",
					tt.jmbg, err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if got != nil {
					t.Errorf("ParseJMBG(%q) = %+v, want nil",
						tt.jmbg, got)
				}
				return
			}

			if got == nil {
				t.Fatal("ParseJMBG() returned nil JMBG")
			}

			if !got.Date.Equal(tt.want.Date) {
				t.Errorf("Date = %v, want %v", got.Date, tt.want.Date)
			}

			if !reflect.DeepEqual(got.Region, tt.want.Region) {
				t.Errorf("Region = %+v, want %+v", got.Region, tt.want.Region)
			}

			if got.RegionCode != tt.want.RegionCode {
				t.Errorf("RegionCode = %d, want %d",
					got.RegionCode, tt.want.RegionCode)
			}

			if got.Gender != tt.want.Gender {
				t.Errorf("Gender = %v, want %v",
					got.Gender, tt.want.Gender)
			}

			if got.SerialNumber != tt.want.SerialNumber {
				t.Errorf("SerialNumber = %d, want %d",
					got.SerialNumber, tt.want.SerialNumber)
			}
		})
	}
}
