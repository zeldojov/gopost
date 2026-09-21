package jmbg

import (
	"strconv"
	"time"
)

func jmbgYear(jmbg string) (int, bool) {
	yearPart, _ := strconv.Atoi(jmbg[4:7])

	switch jmbg[4] {
	case '9':
		return 1900 + yearPart - 900, true
	case '0':
		return 2000 + yearPart, true
	default:
		return 0, false
	}
}

func jmbgGender(serialNumber int) JMBGGender {
	if serialNumber >= 500 {
		return JMBGFemale
	}

	return JMBGMale
}

func ParseJMBG(jmbg string) (*JMBG, error) {
	if len(jmbg) != 13 {
		return nil, ErrInvalidLength
	}

	if !isJMBGDigits(jmbg) {
		return nil, ErrInvalidDigits
	}

	if !validJMBGDate(jmbg) {
		return nil, ErrInvalidDate
	}

	if !validJMBGRegion(jmbg) {
		return nil, ErrInvalidRegion
	}

	if !validJMBGChecksum(jmbg) {
		return nil, ErrInvalidChecksum
	}

	day, _ := strconv.Atoi(jmbg[0:2])
	month, _ := strconv.Atoi(jmbg[2:4])
	year, _ := jmbgYear(jmbg)
	regionCode, _ := strconv.Atoi(jmbg[7:9])
	serialNumber, _ := strconv.Atoi(jmbg[9:12])

	return &JMBG{
		Date: time.Date(
			year,
			time.Month(month),
			day,
			0, 0, 0, 0,
			time.UTC,
		),
		Region:       jmbgRegions[jmbg[7:9]],
		RegionCode:   regionCode,
		Gender:       jmbgGender(serialNumber),
		SerialNumber: serialNumber,
	}, nil
}
