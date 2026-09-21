package jmbg

import (
	"time"
)

const (
	JMBGMale JMBGGender = iota
	JMBGFemale
)

type (
	JMBGGender uint8
	JMBGRegion struct {
		Name           string
		Municipalities []string
	}
	JMBG struct {
		Date         time.Time
		Region       JMBGRegion
		RegionCode   int
		Gender       JMBGGender
		SerialNumber int
	}
)
