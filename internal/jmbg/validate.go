package jmbg

import (
	"strconv"
	"time"
)

func jmbgChecksum(jmbg string) int {
	sum :=
		7*(int(jmbg[0]-'0')+int(jmbg[6]-'0')) +
			6*(int(jmbg[1]-'0')+int(jmbg[7]-'0')) +
			5*(int(jmbg[2]-'0')+int(jmbg[8]-'0')) +
			4*(int(jmbg[3]-'0')+int(jmbg[9]-'0')) +
			3*(int(jmbg[4]-'0')+int(jmbg[10]-'0')) +
			2*(int(jmbg[5]-'0')+int(jmbg[11]-'0'))

	remainder := sum % 11

	switch remainder {
	case 0:
		return 0
	case 1:
		return -1
	default:
		return 11 - remainder
	}
}

func validJMBGChecksum(jmbg string) bool {
	checksum := jmbgChecksum(jmbg)

	if checksum < 0 {
		return false
	}

	return checksum == int(jmbg[12]-'0')
}

func isJMBGDigits(jmbg string) bool {
	for i := 0; i < len(jmbg); i++ {
		if jmbg[i] < '0' || jmbg[i] > '9' {
			return false
		}
	}

	return true
}

func validJMBGDate(jmbg string) bool {
	day, _ := strconv.Atoi(jmbg[0:2])
	month, _ := strconv.Atoi(jmbg[2:4])

	year, ok := jmbgYear(jmbg)
	if !ok {
		return false
	}

	date := time.Date(
		year,
		time.Month(month),
		day,
		0, 0, 0, 0,
		time.UTC,
	)

	return date.Year() == year &&
		int(date.Month()) == month &&
		date.Day() == day
}

func validJMBGRegion(jmbg string) bool {
	_, ok := jmbgRegions[jmbg[7:9]]
	return ok
}

func ValidateJMBG(jmbg string) bool {
	if len(jmbg) != 13 {
		return false
	}

	if !isJMBGDigits(jmbg) {
		return false
	}

	if !validJMBGDate(jmbg) {
		return false
	}

	if !validJMBGRegion(jmbg) {
		return false
	}

	return validJMBGChecksum(jmbg)
}
