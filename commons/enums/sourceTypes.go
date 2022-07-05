package enums

import "strings"

type Sources string

const (
	MeasurementAutomation = "MA"
	AutoImageSelection    = "AIS"
)

func SourcesList() []string {
	return []string{MeasurementAutomation, AutoImageSelection}
}

func (s Sources) String() string {
	l := SourcesList()
	x := strings.ToLower(string(s))
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
